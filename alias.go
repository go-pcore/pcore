// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import (
	"strings"
	"unicode"
)

// Loader is a type environment: a registry of named type aliases (and TypeSet
// members) that resolves alias references, including forward and recursive
// references. It mirrors the role of Puppet's Loader / TypeSet scope for the
// type calculus.
type Loader struct {
	aliases  map[string]*aliasType
	typesets map[string]*typeSetType
}

// NewLoader returns an empty type environment.
func NewLoader() *Loader {
	return &Loader{aliases: map[string]*aliasType{}, typesets: map[string]*typeSetType{}}
}

// aliasType is a named type alias (Puppet's PTypeAliasType). Its body is the
// resolved type expression, which may refer back to the alias for recursion.
type aliasType struct {
	name     string
	resolved Type
}

func (a *aliasType) body() Type { return a.resolved }

func (a *aliasType) Name() string   { return a.name }
func (a *aliasType) String() string { return a.name }

func (a *aliasType) isInstance(v Value, g *guard) bool {
	if g.enter(a.name, valueRepr(v)) {
		return false // co-inductive: an unproven recursive membership does not hold
	}
	return a.resolved.isInstance(v, g)
}

func (a *aliasType) isAssignable(other Type, g *guard) bool {
	if g.enter(a.name, other.String()) {
		return true // co-inductive: assume the subtype relation on re-entry
	}
	return asg(a.resolved, other, g)
}

// ref returns the alias registered under name, creating an unresolved
// placeholder for a forward or recursive reference.
func (l *Loader) ref(name string) *aliasType {
	if a, ok := l.aliases[name]; ok {
		return a
	}
	a := &aliasType{name: name}
	l.aliases[name] = a
	return a
}

// Declare registers a type alias from a declaration of the form
// "type Name = <type-expr>". Forward and recursive references are permitted; use
// [Loader.Parse] or [Loader.Validate] afterwards to confirm every reference
// resolves.
func (l *Loader) Declare(decl string) error {
	decl = strings.TrimSpace(decl)
	rest, ok := strings.CutPrefix(decl, "type")
	if !ok || rest == "" || !unicode.IsSpace(rune(rest[0])) {
		return &ParseError{Msg: "type alias must begin with 'type '"}
	}
	eq := assignIndex(rest)
	if eq < 0 {
		return &ParseError{Msg: "type alias must contain '='"}
	}
	name := strings.TrimSpace(rest[:eq])
	body := strings.TrimSpace(rest[eq+1:])
	if !isTypeName(name) {
		return &ParseError{Msg: "invalid type alias name " + name}
	}
	if isBuiltinName(name) {
		return &ParseError{Msg: "cannot redefine built-in type " + name}
	}
	if a, ok := l.aliases[name]; ok && a.resolved != nil {
		return &ParseError{Msg: "type " + name + " already declared"}
	}
	t, err := l.parseRaw(body)
	if err != nil {
		return err
	}
	l.ref(name).resolved = t
	return nil
}

// assignIndex returns the index of the assignment '=' (not part of '=>') in s,
// or -1.
func assignIndex(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			if i+1 < len(s) && s[i+1] == '>' {
				i++ // skip '=>'
				continue
			}
			return i
		}
	}
	return -1
}

// isTypeName reports whether s is a valid (capitalized) Pcore type name.
func isTypeName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsUpper(r) {
				return false
			}
			continue
		}
		if !(r == '_' || r == ':' || unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}

// parseRaw parses an expression in this loader's scope without validating that
// every reference resolves (used while declarations are still being added).
func (l *Loader) parseRaw(expr string) (Type, error) {
	toks, err := tokenize(expr)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks, loader: l}
	t, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if p.cur().kind != tEOF {
		return nil, p.errf("unexpected trailing input %q", p.cur().text)
	}
	return t, nil
}

// Parse parses a type expression in this loader's scope, resolving alias names,
// then validates that every referenced alias has been declared.
func (l *Loader) Parse(expr string) (Type, error) {
	t, err := l.parseRaw(expr)
	if err != nil {
		return nil, err
	}
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return t, nil
}

// Validate reports the first alias that is referenced but never declared, or one
// whose definition is a non-productive cycle (it resolves only through other
// aliases back to itself).
func (l *Loader) Validate() error {
	for name, a := range l.aliases {
		if a.resolved == nil {
			return &ParseError{Msg: "unresolved type reference " + name}
		}
	}
	for name, a := range l.aliases {
		if err := checkProductive(name, a); err != nil {
			return err
		}
	}
	return nil
}

// checkProductive follows an alias through pure alias indirection; a cycle that
// never reaches a concrete constructor is unresolvable.
func checkProductive(name string, a *aliasType) error {
	seen := map[*aliasType]bool{a: true}
	cur := a.resolved
	for {
		al, ok := cur.(*aliasType)
		if !ok {
			return nil // reached a real constructor
		}
		if seen[al] {
			return &ParseError{Msg: "type alias " + name + " cannot be resolved to a real type"}
		}
		seen[al] = true
		cur = al.resolved
	}
}

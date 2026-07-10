// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import (
	"strconv"
	"strings"
)

// --- generic value/literal parsing (shared by Object and TypeSet) ----------

// parseValue parses a Pcore literal: a scalar, a regexp, an array or hash
// literal, or a nested type expression.
func (p *parser) parseValue() (Value, error) {
	tok := p.cur()
	switch tok.kind {
	case tString:
		p.advance()
		return tok.text, nil
	case tInt:
		v, err := strconv.ParseInt(tok.text, 10, 64)
		if err != nil {
			return nil, p.errf("invalid integer %q", tok.text)
		}
		p.advance()
		return v, nil
	case tFloat:
		v, _ := strconv.ParseFloat(tok.text, 64)
		p.advance()
		return v, nil
	case tRegexp:
		re, err := NewRegexp(tok.text)
		if err != nil {
			return nil, p.errf("invalid regexp /%s/: %v", tok.text, err)
		}
		p.advance()
		return re, nil
	case tLBrace:
		return p.parseDataHash()
	case tLBrack:
		return p.parseDataArray()
	case tName:
		switch tok.text {
		case "true":
			p.advance()
			return true, nil
		case "false":
			p.advance()
			return false, nil
		case "default":
			p.advance()
			return Default, nil
		case "undef":
			p.advance()
			return Undef, nil
		default:
			return p.parseType()
		}
	default:
		return nil, p.errf("unexpected %q in value", tok.text)
	}
}

// parseHashKey parses a hash key: a quoted string or a bareword name.
func (p *parser) parseHashKey() (string, error) {
	tok := p.cur()
	switch tok.kind {
	case tString:
		p.advance()
		return tok.text, nil
	case tName:
		p.advance()
		return tok.text, nil
	default:
		return "", p.errf("expected a hash key, got %q", tok.text)
	}
}

// parseDataHash parses { key => value, ... } into an ordered *Hash.
func (p *parser) parseDataHash() (*Hash, error) {
	p.advance() // '{'
	var entries []HashEntry
	for {
		if p.cur().kind == tRBrace {
			p.advance()
			return NewHash(entries...), nil
		}
		key, err := p.parseHashKey()
		if err != nil {
			return nil, err
		}
		if p.cur().kind != tArrow {
			return nil, p.errf("expected '=>' after hash key %q", key)
		}
		p.advance()
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		entries = append(entries, HashEntry{Key: key, Value: val})
		switch p.cur().kind {
		case tComma:
			p.advance()
		case tRBrace:
			// handled at loop top
		default:
			return nil, p.errf("expected ',' or '}' in hash, got %q", p.cur().text)
		}
	}
}

// parseDataArray parses [ value, ... ] into an ordered []Value.
func (p *parser) parseDataArray() ([]Value, error) {
	p.advance() // '['
	var out []Value
	for {
		if p.cur().kind == tRBrack {
			p.advance()
			return out, nil
		}
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		switch p.cur().kind {
		case tComma:
			p.advance()
		case tRBrack:
			// handled at loop top
		default:
			return nil, p.errf("expected ',' or ']' in array, got %q", p.cur().text)
		}
	}
}

// --- Object ----------------------------------------------------------------

// parseObject parses "Object" or "Object[{...}]".
func (p *parser) parseObject() (Type, error) {
	if p.cur().kind != tLBrack {
		return &objectType{}, nil
	}
	p.advance() // '['
	if p.cur().kind != tLBrace {
		return nil, p.errf("Object requires a definition hash")
	}
	h, err := p.parseDataHash()
	if err != nil {
		return nil, err
	}
	if p.cur().kind != tRBrack {
		return nil, p.errf("expected ']' after Object definition")
	}
	p.advance()
	return buildObjectFromHash(h, p)
}

func buildObjectFromHash(h *Hash, p *parser) (Type, error) {
	t := &objectType{}
	nameV, ok := h.Get("name")
	if !ok {
		return nil, p.errf("Object requires a 'name'")
	}
	name, ok := nameV.(string)
	if !ok {
		return nil, p.errf("Object 'name' must be a string")
	}
	t.name = name
	if pv, ok := h.Get("parent"); ok {
		pt, ok := pv.(Type)
		if !ok {
			return nil, p.errf("Object 'parent' must be a type")
		}
		t.parent = pt
	}
	if av, ok := h.Get("attributes"); ok {
		ah, ok := av.(*Hash)
		if !ok {
			return nil, p.errf("Object 'attributes' must be a hash")
		}
		for _, e := range ah.entries {
			attr, err := buildAttr(e, p)
			if err != nil {
				return nil, err
			}
			t.attrs = append(t.attrs, attr)
		}
	}
	return t, nil
}

func buildAttr(e HashEntry, p *parser) (objAttr, error) {
	name := e.Key.(string)
	switch v := e.Value.(type) {
	case Type:
		return objAttr{name: name, typ: v}, nil
	case *Hash:
		tv, ok := v.Get("type")
		if !ok {
			return objAttr{}, p.errf("attribute %q requires a 'type'", name)
		}
		ty, ok := tv.(Type)
		if !ok {
			return objAttr{}, p.errf("attribute %q 'type' must be a type", name)
		}
		a := objAttr{name: name, typ: ty}
		if dv, ok := v.Get("value"); ok {
			a.hasDef = true
			a.defVal = canon(dv)
		}
		return a, nil
	default:
		return objAttr{}, p.errf("attribute %q must be a type or a hash", name)
	}
}

// --- TypeSet ---------------------------------------------------------------

// typeSetRef is a reference from a TypeSet to another named TypeSet.
type typeSetRef struct {
	alias        string
	name         string
	versionRange string
	target       *typeSetType
}

// typeSetType is a Pcore TypeSet: a named, versioned namespace of type
// definitions and references to other TypeSets.
type typeSetType struct {
	name         string
	version      string
	pcoreVersion string
	members      *Loader
	memberNames  []string
	refs         []*typeSetRef
	nameToRef    map[string]*typeSetRef
}

// Type returns a member type of the set by name.
func (t *typeSetType) Type(name string) (Type, bool) {
	a, ok := t.members.aliases[name]
	if !ok || a.resolved == nil {
		return nil, false
	}
	return a, true
}

// TypeSetName returns the set's name.
func (t *typeSetType) TypeSetName() string { return t.name }

// Version returns the set's version.
func (t *typeSetType) Version() string { return t.version }

func (*typeSetType) Name() string { return "TypeSet" }

func (t *typeSetType) String() string {
	var b strings.Builder
	b.WriteString("TypeSet[{name => ")
	b.WriteString(quote(t.name))
	if t.pcoreVersion != "" {
		b.WriteString(", pcore_version => " + quote(t.pcoreVersion))
	}
	if t.version != "" {
		b.WriteString(", version => " + quote(t.version))
	}
	if len(t.memberNames) > 0 {
		b.WriteString(", types => {")
		for i, n := range t.memberNames {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(n + " => " + t.members.aliases[n].resolved.String())
		}
		b.WriteString("}")
	}
	if len(t.refs) > 0 {
		b.WriteString(", references => {")
		for i, r := range t.refs {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(r.alias + " => {name => " + quote(r.name))
			if r.versionRange != "" {
				b.WriteString(", version_range => " + quote(r.versionRange))
			}
			b.WriteString("}")
		}
		b.WriteString("}")
	}
	b.WriteString("}]")
	return b.String()
}

// isInstance treats a TypeSet as the union of its member types.
func (t *typeSetType) isInstance(v Value, g *guard) bool {
	for _, n := range t.memberNames {
		if t.members.aliases[n].isInstance(v, g) {
			return true
		}
	}
	return false
}

// isAssignable is nominal: a TypeSet accepts another with the same name and
// version.
func (t *typeSetType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*typeSetType)
	return ok && t.name == o.name && t.version == o.version
}

// parseTypeSet parses "TypeSet[{...}]".
func (p *parser) parseTypeSet() (Type, error) {
	if p.cur().kind != tLBrack {
		return nil, p.errf("TypeSet requires a definition")
	}
	p.advance() // '['
	if p.cur().kind != tLBrace {
		return nil, p.errf("TypeSet requires a definition hash")
	}
	ts := &typeSetType{members: NewLoader(), nameToRef: map[string]*typeSetRef{}}
	p.advance() // '{'
	for {
		if p.cur().kind == tRBrace {
			p.advance()
			break
		}
		key, err := p.parseHashKey()
		if err != nil {
			return nil, err
		}
		if p.cur().kind != tArrow {
			return nil, p.errf("expected '=>' after TypeSet key %q", key)
		}
		p.advance()
		if err := p.parseTypeSetEntry(ts, key); err != nil {
			return nil, err
		}
		switch p.cur().kind {
		case tComma:
			p.advance()
		case tRBrace:
			// handled at loop top
		default:
			return nil, p.errf("expected ',' or '}' in TypeSet, got %q", p.cur().text)
		}
	}
	if p.cur().kind != tRBrack {
		return nil, p.errf("expected ']' after TypeSet definition")
	}
	p.advance()
	if err := ts.finalize(p.loader); err != nil {
		return nil, err
	}
	if p.loader != nil && ts.name != "" {
		p.loader.typesets[ts.name] = ts
	}
	return ts, nil
}

func (p *parser) parseTypeSetEntry(ts *typeSetType, key string) error {
	switch key {
	case "name":
		s, err := p.stringValue()
		ts.name = s
		return err
	case "version":
		s, err := p.stringValue()
		ts.version = s
		return err
	case "pcore_version":
		s, err := p.stringValue()
		ts.pcoreVersion = s
		return err
	case "types":
		return p.parseTypesBlock(ts)
	case "references":
		return p.parseReferencesBlock(ts)
	default:
		return p.errf("unknown TypeSet key %q", key)
	}
}

// stringValue parses a value that must be a string.
func (p *parser) stringValue() (string, error) {
	v, err := p.parseValue()
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", p.errf("expected a string value")
	}
	return s, nil
}

// parseTypesBlock parses { Name => <type-expr>, ... } in the member scope.
func (p *parser) parseTypesBlock(ts *typeSetType) error {
	if p.cur().kind != tLBrace {
		return p.errf("TypeSet 'types' must be a hash")
	}
	old := p.loader
	p.loader = ts.members
	defer func() { p.loader = old }()
	p.advance() // '{'
	for {
		if p.cur().kind == tRBrace {
			p.advance()
			return nil
		}
		name, err := p.parseHashKey()
		if err != nil {
			return err
		}
		if p.cur().kind != tArrow {
			return p.errf("expected '=>' after type name %q", name)
		}
		p.advance()
		typ, err := p.parseType()
		if err != nil {
			return err
		}
		if a, ok := ts.members.aliases[name]; ok && a.resolved != nil {
			return p.errf("duplicate type %q in TypeSet", name)
		}
		ts.members.ref(name).resolved = typ
		ts.memberNames = append(ts.memberNames, name)
		switch p.cur().kind {
		case tComma:
			p.advance()
		case tRBrace:
			// handled at loop top
		default:
			return p.errf("expected ',' or '}' in types, got %q", p.cur().text)
		}
	}
}

// parseReferencesBlock parses { Alias => {name => .., version_range => ..}, ... }.
func (p *parser) parseReferencesBlock(ts *typeSetType) error {
	h, err := p.parseValue()
	if err != nil {
		return err
	}
	refs, ok := h.(*Hash)
	if !ok {
		return p.errf("TypeSet 'references' must be a hash")
	}
	for _, e := range refs.entries {
		alias := e.Key.(string)
		body, ok := e.Value.(*Hash)
		if !ok {
			return p.errf("reference %q must be a hash", alias)
		}
		nameV, ok := body.Get("name")
		if !ok {
			return p.errf("reference %q requires a 'name'", alias)
		}
		name, ok := nameV.(string)
		if !ok {
			return p.errf("reference %q 'name' must be a string", alias)
		}
		ref := &typeSetRef{alias: alias, name: name}
		if vr, ok := body.Get("version_range"); ok {
			s, ok := vr.(string)
			if !ok {
				return p.errf("reference %q 'version_range' must be a string", alias)
			}
			ref.versionRange = s
		}
		ts.refs = append(ts.refs, ref)
		ts.nameToRef[alias] = ref
	}
	return nil
}

// finalize resolves cross-TypeSet references and validates the member scope.
func (t *typeSetType) finalize(outer *Loader) error {
	for _, r := range t.refs {
		if outer != nil {
			if target, ok := outer.typesets[r.name]; ok {
				r.target = target
			}
		}
	}
	// Resolve member references of the form Alias::MemberName.
	for name, a := range t.members.aliases {
		if a.resolved != nil {
			continue
		}
		parts := strings.SplitN(name, "::", 2)
		if len(parts) == 2 {
			if ref, ok := t.nameToRef[parts[0]]; ok && ref.target != nil {
				if mt, ok := ref.target.Type(parts[1]); ok {
					a.resolved = mt
					continue
				}
			}
		}
	}
	return t.members.Validate()
}

// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "strings"

// --- Runtime ---------------------------------------------------------------

// runtimeType is Runtime['runtime', 'name-or-pattern'] — a host-language object
// type (e.g. Runtime['go', '*bytes.Buffer']).
type runtimeType struct {
	runtime string
	name    string  // exact name, or "" when a pattern is used
	pat     *Regexp // name pattern, or nil
}

// RuntimeValue wraps a foreign (host-language) object with its runtime and name.
type RuntimeValue struct {
	runtime, name string
	obj           any
}

// NewRuntimeValue tags obj with a runtime and a type name.
func NewRuntimeValue(runtime, name string, obj any) *RuntimeValue {
	return &RuntimeValue{runtime: runtime, name: name, obj: obj}
}

// Unwrap returns the wrapped object.
func (r *RuntimeValue) Unwrap() any { return r.obj }

// String renders the runtime value's tag.
func (r *RuntimeValue) String() string { return "Runtime['" + r.runtime + "', '" + r.name + "']" }

func (*runtimeType) Name() string { return "Runtime" }

func (t *runtimeType) String() string {
	if t.runtime == "" {
		return "Runtime"
	}
	if t.pat != nil {
		return "Runtime[" + quote(t.runtime) + ", " + t.pat.String() + "]"
	}
	if t.name == "" {
		return "Runtime[" + quote(t.runtime) + "]"
	}
	return "Runtime[" + quote(t.runtime) + ", " + quote(t.name) + "]"
}

func (t *runtimeType) isInstance(v Value, _ *guard) bool {
	rv, ok := v.(*RuntimeValue)
	if !ok {
		return false
	}
	if t.runtime != "" && t.runtime != rv.runtime {
		return false
	}
	if t.pat != nil {
		return t.pat.MatchString(rv.name)
	}
	return t.name == "" || t.name == rv.name
}

func (t *runtimeType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*runtimeType)
	if !ok {
		return false
	}
	if t.runtime == "" {
		return true
	}
	if t.runtime != o.runtime {
		return false
	}
	if t.pat != nil {
		return o.name != "" && t.pat.MatchString(o.name)
	}
	return t.name == "" || t.name == o.name
}

// --- URI -------------------------------------------------------------------

// uriType is URI[scheme] — a Uniform Resource Identifier type. A non-empty
// scheme constrains the URI's scheme (Puppet's common use); otherwise it matches
// any URI value.
type uriType struct {
	scheme string // required scheme, or "" for any
}

// URI is a Pcore URI value.
type URI struct{ uri string }

// NewURI wraps s as a URI value.
func NewURI(s string) *URI { return &URI{uri: s} }

// Value returns the URI string.
func (u *URI) Value() string { return u.uri }

// String renders the URI value.
func (u *URI) String() string { return u.uri }

func (*uriType) Name() string { return "URI" }

func (t *uriType) String() string {
	if t.scheme != "" {
		return "URI[" + quote(t.scheme) + "]"
	}
	return "URI"
}

func (t *uriType) isInstance(v Value, _ *guard) bool {
	u, ok := v.(*URI)
	if !ok {
		return false
	}
	if t.scheme == "" {
		return true
	}
	return strings.HasPrefix(u.uri, t.scheme+":")
}

func (t *uriType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*uriType)
	if !ok {
		return false
	}
	return t.scheme == "" || t.scheme == o.scheme
}

// --- Iterable / Iterator ---------------------------------------------------

// iterableType is Iterable[T] — anything that can be iterated yielding Ts.
type iterableType struct{ elem Type }

// iteratorType is Iterator[T] — a lazy sequence of Ts.
type iteratorType struct{ elem Type }

// Iterator is a Pcore iterator value over a fixed element sequence.
type Iterator struct {
	elem  Type
	items []Value
}

// NewIterator builds an iterator over items with declared element type elem.
func NewIterator(elem Type, items ...Value) *Iterator {
	cs := make([]Value, len(items))
	for i, it := range items {
		cs[i] = canon(it)
	}
	return &Iterator{elem: elem, items: cs}
}

// Items returns the iterator's elements.
func (it *Iterator) Items() []Value { return it.items }

// String renders the iterator.
func (it *Iterator) String() string { return "Iterator[" + it.elem.String() + "]" }

func (*iterableType) Name() string { return "Iterable" }
func (*iteratorType) Name() string { return "Iterator" }

func (t *iterableType) String() string {
	if isAnyType(t.elem) {
		return "Iterable"
	}
	return "Iterable[" + t.elem.String() + "]"
}

func (t *iteratorType) String() string {
	if isAnyType(t.elem) {
		return "Iterator"
	}
	return "Iterator[" + t.elem.String() + "]"
}

func isAnyType(t Type) bool { _, ok := t.(*anyType); return ok }

func (t *iterableType) isInstance(v Value, g *guard) bool {
	switch x := v.(type) {
	case []Value:
		for _, e := range x {
			if !t.elem.isInstance(e, g) {
				return false
			}
		}
		return true
	case string:
		// A String iterates as single-character Strings.
		return t.elem.isInstance(x, g) || isAnyType(t.elem) || charElem(t.elem)
	case *Iterator:
		return asg(t.elem, x.elem, g)
	default:
		return false
	}
}

// charElem reports whether elem accepts single-character strings.
func charElem(elem Type) bool { return elem.isInstance("a", newGuard()) }

func (t *iteratorType) isInstance(v Value, g *guard) bool {
	it, ok := v.(*Iterator)
	if !ok {
		return false
	}
	return asg(t.elem, it.elem, g)
}

func (t *iterableType) isAssignable(other Type, g *guard) bool {
	switch o := other.(type) {
	case *iterableType:
		return asg(t.elem, o.elem, g)
	case *iteratorType:
		return asg(t.elem, o.elem, g)
	case *arrayType:
		return asg(t.elem, o.element, g)
	default:
		return false
	}
}

func (t *iteratorType) isAssignable(other Type, g *guard) bool {
	o, ok := other.(*iteratorType)
	return ok && asg(t.elem, o.elem, g)
}

// --- Error -----------------------------------------------------------------

// errorType is Error[kind, issue_code] — the Pcore error object type.
type errorType struct {
	kind  Type // Optional[String]/Enum/etc., or Any
	issue Type // Optional[String], or Any
}

// ErrorValue is a Pcore Error value.
type ErrorValue struct {
	message, kind, issueCode string
}

// NewError builds an Error value.
func NewError(message, kind, issueCode string) *ErrorValue {
	return &ErrorValue{message: message, kind: kind, issueCode: issueCode}
}

// Message returns the error message.
func (e *ErrorValue) Message() string { return e.message }

// Kind returns the error kind.
func (e *ErrorValue) Kind() string { return e.kind }

// IssueCode returns the error issue code.
func (e *ErrorValue) IssueCode() string { return e.issueCode }

// String renders the error.
func (e *ErrorValue) String() string { return "Error('" + e.message + "')" }

func (*errorType) Name() string { return "Error" }

func (t *errorType) String() string {
	kindAny, issueAny := isAnyType(t.kind), isAnyType(t.issue)
	if kindAny && issueAny {
		return "Error"
	}
	if issueAny {
		return "Error[" + t.kind.String() + "]"
	}
	return "Error[" + t.kind.String() + ", " + t.issue.String() + "]"
}

func (t *errorType) isInstance(v Value, g *guard) bool {
	e, ok := v.(*ErrorValue)
	if !ok {
		return false
	}
	if !t.kind.isInstance(errKindValue(e.kind), g) {
		return false
	}
	return t.issue.isInstance(errKindValue(e.issueCode), g)
}

// errKindValue maps an empty kind/issue string to Undef, else the string.
func errKindValue(s string) Value {
	if s == "" {
		return Undef
	}
	return s
}

func (t *errorType) isAssignable(other Type, g *guard) bool {
	o, ok := other.(*errorType)
	if !ok {
		return false
	}
	return asg(t.kind, o.kind, g) && asg(t.issue, o.issue, g)
}

// --- Callable --------------------------------------------------------------

// callableType is Callable[params..., min, max, block].
type callableType struct {
	params  []Type
	minSz   int64
	maxSz   int64
	hasSize bool
	block   Type // block/lambda type, or nil
	generic bool // Callable with no parameters at all (matches any callable)
}

// Callable is a Pcore callable value carrying its signature type.
type Callable struct{ sig *callableType }

// NewCallable builds a callable value whose signature is the Callable type t.
func NewCallable(t Type) (*Callable, error) {
	ct, ok := t.(*callableType)
	if !ok {
		return nil, &ParseError{Msg: "NewCallable requires a Callable type"}
	}
	return &Callable{sig: ct}, nil
}

// String renders the callable value by its signature.
func (c *Callable) String() string { return c.sig.String() }

func (*callableType) Name() string { return "Callable" }

func (t *callableType) String() string {
	if t.generic {
		return "Callable"
	}
	parts := make([]string, 0, len(t.params)+3)
	for _, p := range t.params {
		parts = append(parts, p.String())
	}
	if t.hasSize {
		parts = append(parts, itoa(t.minSz))
		if t.maxSz != t.minSz {
			parts = append(parts, sizeArgTail(t.maxSz))
		}
	}
	if t.block != nil {
		parts = append(parts, t.block.String())
	}
	return "Callable[" + strings.Join(parts, ", ") + "]"
}

// sizeArgTail renders the max element of a Callable arity.
func sizeArgTail(max int64) string {
	if max == maxInt {
		return "default"
	}
	return itoa(max)
}

func (t *callableType) isInstance(v Value, g *guard) bool {
	c, ok := v.(*Callable)
	if !ok {
		return false
	}
	return asg(t, c.sig, g)
}

func (t *callableType) isAssignable(other Type, g *guard) bool {
	o, ok := other.(*callableType)
	if !ok {
		return false
	}
	if t.generic {
		return true
	}
	if o.generic {
		return false
	}
	// Arity: the supertype's accepted arity must be covered by the subtype.
	ta, tb := t.arity()
	oa, ob := o.arity()
	if !szWithin(oa, ob, ta, tb) {
		return false
	}
	// Parameters are contravariant.
	n := len(t.params)
	if len(o.params) > n {
		n = len(o.params)
	}
	for i := 0; i < n; i++ {
		if !asg(o.paramAt(i), t.paramAt(i), g) {
			return false
		}
	}
	return true
}

func (t *callableType) arity() (int64, int64) {
	if t.hasSize {
		return t.minSz, t.maxSz
	}
	n := int64(len(t.params))
	return n, n
}

func (t *callableType) paramAt(i int) Type {
	if len(t.params) == 0 {
		return &anyType{}
	}
	if i < len(t.params) {
		return t.params[i]
	}
	return t.params[len(t.params)-1]
}

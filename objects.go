// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "strings"

// objAttr is one declared attribute of an Object type.
type objAttr struct {
	name   string
	typ    Type
	hasDef bool
	defVal Value
}

// objectType is a Pcore Object type: a nominal, optionally single-inheriting
// type with typed attributes. Assignability is nominal (by name and ancestry);
// instances are ObjectValue values of the type or a descendant.
type objectType struct {
	name   string
	parent Type // another *objectType (possibly via an alias), or nil
	attrs  []objAttr
}

// ObjectValue is an instance of an Object type: a typed record of attributes.
type ObjectValue struct {
	typ   *objectType
	attrs *Hash
}

// NewObjectValue builds an instance of the Object type t from the given
// attribute hash. It validates that every non-defaulted attribute is present and
// type-correct, filling in declared defaults for absent optional attributes.
func NewObjectValue(t Type, attrs *Hash) (*ObjectValue, error) {
	ot := asObjectType(t)
	if ot == nil {
		return nil, &ParseError{Msg: "NewObjectValue requires an Object type"}
	}
	if attrs == nil {
		attrs = NewHash()
	}
	filled := make([]HashEntry, 0)
	for _, a := range ot.allAttrs() {
		v, present := attrs.Get(a.name)
		if !present {
			if a.hasDef {
				filled = append(filled, HashEntry{Key: a.name, Value: a.defVal})
				continue
			}
			return nil, &ParseError{Msg: "missing attribute '" + a.name + "' for Object " + ot.name}
		}
		if !IsInstance(a.typ, v) {
			return nil, &ParseError{Msg: "attribute '" + a.name + "' of Object " + ot.name + " is not a " + a.typ.String()}
		}
		filled = append(filled, HashEntry{Key: a.name, Value: canon(v)})
	}
	return &ObjectValue{typ: ot, attrs: NewHash(filled...)}, nil
}

// Get returns the value of attribute name.
func (o *ObjectValue) Get(name string) (Value, bool) { return o.attrs.Get(name) }

// String renders the object as Name({...}).
func (o *ObjectValue) String() string { return o.typ.name + o.attrs.String() }

// allAttrs returns the attributes of the type including inherited ones, parents
// first.
func (t *objectType) allAttrs() []objAttr {
	var base []objAttr
	if p := asObjectType(t.parent); p != nil {
		base = p.allAttrs()
	}
	return append(base, t.attrs...)
}

// asObjectType unwraps a Type (possibly an alias) to an *objectType, or nil.
func asObjectType(t Type) *objectType {
	switch x := t.(type) {
	case *objectType:
		return x
	case *aliasType:
		return asObjectType(x.body())
	default:
		return nil
	}
}

func (t *objectType) Name() string { return "Object" }

func (t *objectType) String() string {
	var b strings.Builder
	b.WriteString("Object[{'name' => ")
	b.WriteString(quote(t.name))
	if t.parent != nil {
		b.WriteString(", 'parent' => ")
		b.WriteString(parentName(t.parent))
	}
	if len(t.attrs) > 0 {
		b.WriteString(", 'attributes' => {")
		for i, a := range t.attrs {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quote(a.name))
			b.WriteString(" => ")
			if a.hasDef {
				b.WriteString("{'type' => " + a.typ.String() + ", 'value' => " + valueLiteral(a.defVal) + "}")
			} else {
				b.WriteString(a.typ.String())
			}
		}
		b.WriteString("}")
	}
	b.WriteString("}]")
	return b.String()
}

// valueLiteral renders a scalar attribute default as a Pcore literal.
func valueLiteral(v Value) string {
	switch x := canon(v).(type) {
	case undefValue:
		return "undef"
	case string:
		return quote(x)
	case int64:
		return itoa(x)
	case float64:
		return ftoa(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		return quote(valueRepr(x))
	}
}

// parentName renders a parent reference: an alias by its name, an inline Object
// by its full form.
func parentName(t Type) string {
	if a, ok := t.(*aliasType); ok {
		return a.name
	}
	return t.String()
}

func (t *objectType) isInstance(v Value, _ *guard) bool {
	o, ok := v.(*ObjectValue)
	if !ok {
		return false
	}
	if t.name == "" { // the abstract base Object matches any object value
		return true
	}
	return o.typ.isNominalSubtype(t)
}

// isNominalSubtype reports whether t (or one of its ancestors) is the type a.
func (t *objectType) isNominalSubtype(a *objectType) bool {
	for cur := t; cur != nil; cur = asObjectType(cur.parent) {
		if cur == a || (cur.name == a.name && cur.name != "") {
			return true
		}
	}
	return false
}

func (t *objectType) isAssignable(other Type, _ *guard) bool {
	o := asObjectType(other)
	if o == nil {
		return false
	}
	if t.name == "" { // the abstract base Object accepts any object type
		return true
	}
	return o.isNominalSubtype(t)
}

// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "strings"

// isInstance answers whether the canonical value v is an instance of the type.
// The guard g terminates recursion through recursive type aliases.

func (anyType) isInstance(Value, *guard) bool { return true }

func (undefType) isInstance(v Value, _ *guard) bool {
	_, ok := v.(undefValue)
	return ok
}

func (defaultTypeT) isInstance(v Value, _ *guard) bool {
	_, ok := v.(defaultValue)
	return ok
}

func (booleanType) isInstance(v Value, _ *guard) bool {
	_, ok := v.(bool)
	return ok
}

func (t *integerType) isInstance(v Value, _ *guard) bool {
	i, ok := v.(int64)
	return ok && i >= t.min && i <= t.max
}

func (t *floatType) isInstance(v Value, _ *guard) bool {
	f, ok := v.(float64)
	return ok && f >= t.min && f <= t.max
}

func (numericType) isInstance(v Value, _ *guard) bool {
	switch v.(type) {
	case int64, float64:
		return true
	default:
		return false
	}
}

func (t *stringType) isInstance(v Value, _ *guard) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	n := int64(len([]rune(s)))
	return n >= t.minLen && n <= t.maxLen
}

func (t *enumType) isInstance(v Value, _ *guard) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	for _, e := range t.values {
		if e == s || (t.ci && strings.EqualFold(e, s)) {
			return true
		}
	}
	return false
}

func (t *patternType) isInstance(v Value, _ *guard) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	for _, p := range t.patterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}

func (t *regexpType) isInstance(v Value, _ *guard) bool {
	r, ok := v.(*Regexp)
	if !ok {
		return false
	}
	return t.pattern == nil || t.pattern.src == r.src
}

func (scalarDataType) isInstance(v Value, _ *guard) bool {
	switch v.(type) {
	case int64, float64, string, bool:
		return true
	default:
		return false
	}
}

func (scalarType) isInstance(v Value, _ *guard) bool {
	switch v.(type) {
	case int64, float64, string, bool, *Regexp, *Timestamp, *Timespan, *SemVer:
		return true
	default:
		return false
	}
}

func (dataType) isInstance(v Value, g *guard) bool {
	switch x := v.(type) {
	case int64, float64, string, bool:
		return true
	case undefValue:
		return true
	case []Value:
		for _, e := range x {
			if !dataType.isInstance(dataType{}, e, g) {
				return false
			}
		}
		return true
	case *Hash:
		for _, e := range x.entries {
			if _, ok := e.Key.(string); !ok {
				return false
			}
			if !dataType.isInstance(dataType{}, e.Value, g) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (t *richDataType) isInstance(v Value, g *guard) bool {
	switch x := v.(type) {
	case []Value:
		for _, e := range x {
			if !t.isInstance(e, g) {
				return false
			}
		}
		return true
	case *Hash:
		for _, e := range x.entries {
			if !(&richDataKeyType{}).isInstance(e.Key, g) {
				return false
			}
			if !t.isInstance(e.Value, g) {
				return false
			}
		}
		return true
	default:
		return richDataLeaf(v)
	}
}

// richDataLeaf reports whether v is a non-collection value RichData admits.
func richDataLeaf(v Value) bool {
	switch v.(type) {
	case int64, float64, string, bool, undefValue, defaultValue,
		*Regexp, *Binary, *Timestamp, *Timespan, *SemVer, *SemVerRange,
		*Sensitive, Type:
		return true
	default:
		return false
	}
}

func (richDataKeyType) isInstance(v Value, _ *guard) bool {
	switch v.(type) {
	case string, int64, float64:
		return true
	default:
		return false
	}
}

func (t *collectionType) isInstance(v Value, _ *guard) bool {
	if n, ok := collectionLen(v); ok {
		return n >= t.minSz && n <= t.maxSz
	}
	return false
}

// collectionLen returns the element count of an array or hash value.
func collectionLen(v Value) (int64, bool) {
	switch x := v.(type) {
	case []Value:
		return int64(len(x)), true
	case *Hash:
		return int64(len(x.entries)), true
	default:
		return 0, false
	}
}

func (t *arrayType) isInstance(v Value, g *guard) bool {
	a, ok := v.([]Value)
	if !ok {
		return false
	}
	n := int64(len(a))
	if n < t.minSz || n > t.maxSz {
		return false
	}
	for _, e := range a {
		if !t.element.isInstance(e, g) {
			return false
		}
	}
	return true
}

func (t *tupleType) isInstance(v Value, g *guard) bool {
	a, ok := v.([]Value)
	if !ok {
		return false
	}
	n := int64(len(a))
	if n < t.minSz || n > t.maxSz {
		return false
	}
	for i, e := range a {
		if !t.typeAt(i).isInstance(e, g) {
			return false
		}
	}
	return true
}

// typeAt returns the element type governing position i.
func (t *tupleType) typeAt(i int) Type {
	if i < len(t.types) {
		return t.types[i]
	}
	return t.types[len(t.types)-1]
}

func (t *hashType) isInstance(v Value, g *guard) bool {
	entries, ok := hashEntriesOf(v)
	if !ok {
		return false
	}
	n := int64(len(entries))
	if n < t.minSz || n > t.maxSz {
		return false
	}
	for _, e := range entries {
		if !t.key.isInstance(e.Key, g) || !t.value.isInstance(e.Value, g) {
			return false
		}
	}
	return true
}

func (t *structType) isInstance(v Value, g *guard) bool {
	entries, ok := hashEntriesOf(v)
	if !ok {
		return false
	}
	seen := make(map[string]Value, len(entries))
	for _, e := range entries {
		k, ok := e.Key.(string)
		if !ok {
			return false
		}
		seen[k] = e.Value
	}
	for _, m := range t.members {
		val, present := seen[m.name]
		if !present {
			if m.optional() {
				continue
			}
			return false
		}
		if !m.typ.isInstance(val, g) {
			return false
		}
		delete(seen, m.name)
	}
	// No keys beyond the declared members (Struct is closed).
	return len(seen) == 0
}

// optional reports whether the member may be absent from a matching hash.
func (m structMember) optional() bool { return m.keyOpt || acceptsUndef(m.typ) }

func (t *variantType) isInstance(v Value, g *guard) bool {
	for _, m := range t.types {
		if m.isInstance(v, g) {
			return true
		}
	}
	return false
}

func (t *optionalType) isInstance(v Value, g *guard) bool {
	if _, ok := v.(undefValue); ok {
		return true
	}
	return t.typ.isInstance(v, g)
}

func (t *notUndefType) isInstance(v Value, g *guard) bool {
	if _, ok := v.(undefValue); ok {
		return false
	}
	return t.typ.isInstance(v, g)
}

func (t *typeType) isInstance(v Value, g *guard) bool {
	ty, ok := v.(Type)
	if !ok {
		return false
	}
	return asg(t.typ, ty, g)
}

func (t *sensitiveType) isInstance(v Value, g *guard) bool {
	s, ok := v.(*Sensitive)
	if !ok {
		return false
	}
	return t.typ.isInstance(s.inner, g)
}

func (binaryType) isInstance(v Value, _ *guard) bool {
	_, ok := v.(*Binary)
	return ok
}

func (t *timestampType) isInstance(v Value, _ *guard) bool {
	ts, ok := v.(*Timestamp)
	if !ok {
		return false
	}
	n := ts.t.UnixNano()
	return n >= t.min && n <= t.max
}

func (t *timespanType) isInstance(v Value, _ *guard) bool {
	ts, ok := v.(*Timespan)
	if !ok {
		return false
	}
	n := int64(ts.d)
	return n >= t.min && n <= t.max
}

// acceptsUndef reports whether the undef value is an instance of t. Used both
// for Struct optionality and for the NotUndef assignability rule.
func acceptsUndef(t Type) bool { return t.isInstance(Undef, newGuard()) }

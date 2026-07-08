package pcore

import "strings"

// isInstance answers whether the canonical value v is an instance of the type.

func (anyType) isInstance(Value) bool { return true }

func (undefType) isInstance(v Value) bool {
	_, ok := v.(undefValue)
	return ok
}

func (defaultTypeT) isInstance(v Value) bool {
	_, ok := v.(defaultValue)
	return ok
}

func (booleanType) isInstance(v Value) bool {
	_, ok := v.(bool)
	return ok
}

func (t *integerType) isInstance(v Value) bool {
	i, ok := v.(int64)
	return ok && i >= t.min && i <= t.max
}

func (t *floatType) isInstance(v Value) bool {
	f, ok := v.(float64)
	return ok && f >= t.min && f <= t.max
}

func (numericType) isInstance(v Value) bool {
	switch v.(type) {
	case int64, float64:
		return true
	default:
		return false
	}
}

func (t *stringType) isInstance(v Value) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	n := int64(len([]rune(s)))
	return n >= t.minLen && n <= t.maxLen
}

func (t *enumType) isInstance(v Value) bool {
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

func (t *patternType) isInstance(v Value) bool {
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

func (t *regexpType) isInstance(v Value) bool {
	r, ok := v.(*Regexp)
	if !ok {
		return false
	}
	return t.pattern == nil || t.pattern.src == r.src
}

func (scalarDataType) isInstance(v Value) bool {
	switch v.(type) {
	case int64, float64, string, bool:
		return true
	default:
		return false
	}
}

func (scalarType) isInstance(v Value) bool {
	switch v.(type) {
	case int64, float64, string, bool, *Regexp, *Timestamp, *Timespan:
		return true
	default:
		return false
	}
}

func (dataType) isInstance(v Value) bool {
	switch x := v.(type) {
	case int64, float64, string, bool:
		return true
	case undefValue:
		return true
	case []Value:
		for _, e := range x {
			if !dataInstance(e) {
				return false
			}
		}
		return true
	case *Hash:
		for _, e := range x.entries {
			if _, ok := e.Key.(string); !ok {
				return false
			}
			if !dataInstance(e.Value) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func dataInstance(v Value) bool { return dataType{}.isInstance(v) }

func (t *collectionType) isInstance(v Value) bool {
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

func (t *arrayType) isInstance(v Value) bool {
	a, ok := v.([]Value)
	if !ok {
		return false
	}
	n := int64(len(a))
	if n < t.minSz || n > t.maxSz {
		return false
	}
	for _, e := range a {
		if !t.element.isInstance(e) {
			return false
		}
	}
	return true
}

func (t *tupleType) isInstance(v Value) bool {
	a, ok := v.([]Value)
	if !ok {
		return false
	}
	n := int64(len(a))
	if n < t.minSz || n > t.maxSz {
		return false
	}
	for i, e := range a {
		if !t.typeAt(i).isInstance(e) {
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

func (t *hashType) isInstance(v Value) bool {
	entries, ok := hashEntriesOf(v)
	if !ok {
		return false
	}
	n := int64(len(entries))
	if n < t.minSz || n > t.maxSz {
		return false
	}
	for _, e := range entries {
		if !t.key.isInstance(e.Key) || !t.value.isInstance(e.Value) {
			return false
		}
	}
	return true
}

func (t *structType) isInstance(v Value) bool {
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
		if !m.typ.isInstance(val) {
			return false
		}
		delete(seen, m.name)
	}
	// No keys beyond the declared members (Struct is closed).
	return len(seen) == 0
}

// optional reports whether the member may be absent from a matching hash.
func (m structMember) optional() bool { return m.keyOpt || acceptsUndef(m.typ) }

func (t *variantType) isInstance(v Value) bool {
	for _, m := range t.types {
		if m.isInstance(v) {
			return true
		}
	}
	return false
}

func (t *optionalType) isInstance(v Value) bool {
	if _, ok := v.(undefValue); ok {
		return true
	}
	return t.typ.isInstance(v)
}

func (t *notUndefType) isInstance(v Value) bool {
	if _, ok := v.(undefValue); ok {
		return false
	}
	return t.typ.isInstance(v)
}

func (t *typeType) isInstance(v Value) bool {
	ty, ok := v.(Type)
	if !ok {
		return false
	}
	return assignable(t.typ, ty)
}

func (t *sensitiveType) isInstance(v Value) bool {
	s, ok := v.(*Sensitive)
	if !ok {
		return false
	}
	return t.typ.isInstance(s.inner)
}

func (binaryType) isInstance(v Value) bool {
	_, ok := v.(*Binary)
	return ok
}

func (timestampType) isInstance(v Value) bool {
	_, ok := v.(*Timestamp)
	return ok
}

func (timespanType) isInstance(v Value) bool {
	_, ok := v.(*Timespan)
	return ok
}

// acceptsUndef reports whether the undef value is an instance of t. Used both
// for Struct optionality and for the NotUndef assignability rule.
func acceptsUndef(t Type) bool { return t.isInstance(Undef) }

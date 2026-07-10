// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

// assignable reports whether b is a subtype of a: every instance of b is an
// instance of a. It is the public entry point that seeds a fresh guard.
func assignable(a, b Type) bool { return asg(a, b, newGuard()) }

// asg is the guarded core of assignable. It resolves an aliased b to its body,
// then distributes over the union-like shapes of b (Variant, Optional, NotUndef)
// so each concrete type's isAssignable only ever sees a "simple" argument, then
// defers to a.isAssignable. An aliased a is handled by aliasType.isAssignable.
func asg(a, b Type, g *guard) bool {
	as, bs := a.String(), b.String()
	if as == bs {
		return true
	}
	// Resolve an aliased b to its body, guarding against infinite recursion.
	if rb, ok := b.(*aliasType); ok {
		if g.enter(as, bs) {
			return true
		}
		return asg(a, rb.body(), g)
	}
	switch bb := b.(type) {
	case *variantType:
		// An empty Variant has no instances and is vacuously a subtype of
		// everything; otherwise every member must be assignable.
		for _, m := range bb.types {
			if !asg(a, m, g) {
				return false
			}
		}
		return true
	case *optionalType:
		return asg(a, &undefType{}, g) && asg(a, bb.typ, g)
	case *notUndefType:
		return asg(a, bb.typ, g)
	}
	return a.isAssignable(b, g)
}

// allowsUndef reports whether a simple type's instance set includes undef. Its
// argument is always a de-aliased, non-union type (asg peels those away).
func allowsUndef(t Type) bool {
	switch t.(type) {
	case *anyType, *undefType, dataType, *richDataType:
		return true
	default:
		return false
	}
}

// numeric-range containment: outer ⊇ inner.
func intWithin(oMin, oMax, iMin, iMax int64) bool   { return oMin <= iMin && iMax <= oMax }
func fltWithin(oMin, oMax, iMin, iMax float64) bool { return oMin <= iMin && iMax <= oMax }
func szWithin(oMin, oMax, iMin, iMax int64) bool    { return oMin <= iMin && iMax <= oMax }

// isAssignable methods. `other` is always a simple type (never Variant,
// Optional, NotUndef or an alias — those are peeled off by asg).

func (anyType) isAssignable(Type, *guard) bool { return true }

func (undefType) isAssignable(other Type, _ *guard) bool { _, ok := other.(*undefType); return ok }

func (defaultTypeT) isAssignable(other Type, _ *guard) bool {
	_, ok := other.(*defaultTypeT)
	return ok
}

func (booleanType) isAssignable(other Type, _ *guard) bool {
	_, ok := other.(*booleanType)
	return ok
}

func (binaryType) isAssignable(other Type, _ *guard) bool { _, ok := other.(*binaryType); return ok }

func (t *timestampType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*timestampType)
	return ok && o.min >= t.min && o.max <= t.max
}

func (t *timespanType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*timespanType)
	return ok && o.min >= t.min && o.max <= t.max
}

func (numericType) isAssignable(other Type, _ *guard) bool {
	switch other.(type) {
	case *numericType, *integerType, *floatType:
		return true
	default:
		return false
	}
}

func (t *integerType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*integerType)
	return ok && intWithin(t.min, t.max, o.min, o.max)
}

func (t *floatType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*floatType)
	return ok && fltWithin(t.min, t.max, o.min, o.max)
}

func (scalarDataType) isAssignable(other Type, _ *guard) bool {
	switch other.(type) {
	case *scalarDataType, *integerType, *floatType, *numericType, *stringType, *enumType, *patternType, *booleanType:
		return true
	default:
		return false
	}
}

func (scalarType) isAssignable(other Type, _ *guard) bool {
	switch other.(type) {
	case *scalarType, *scalarDataType, *integerType, *floatType, *numericType,
		*stringType, *enumType, *patternType, *booleanType,
		*regexpType, *timestampType, *timespanType, *semVerType:
		return true
	default:
		return false
	}
}

func (dataType) isAssignable(other Type, g *guard) bool {
	switch o := other.(type) {
	case *scalarDataType, *integerType, *floatType, *numericType, *stringType, *enumType, *patternType, *booleanType:
		return true
	case *undefType, dataType:
		return true
	case *arrayType:
		return asg(dataType{}, o.element, g)
	case *tupleType:
		for _, e := range o.types {
			if !asg(dataType{}, e, g) {
				return false
			}
		}
		return true
	case *hashType:
		return asg(AnyString(), o.key, g) && asg(dataType{}, o.value, g)
	case *structType:
		for _, m := range o.members {
			if !asg(dataType{}, m.typ, g) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// richDataScalar reports whether t is one of the non-collection types that
// RichData admits in addition to plain Data.
func richDataScalar(t Type) bool {
	switch t.(type) {
	case *scalarType, *scalarDataType, *integerType, *floatType, *numericType,
		*stringType, *enumType, *patternType, *booleanType, *regexpType,
		*undefType, *defaultTypeT, *binaryType, *timestampType, *timespanType,
		*semVerType, *semVerRangeType, *typeType, *sensitiveType, dataType,
		*richDataType:
		return true
	default:
		return false
	}
}

func (t *richDataType) isAssignable(other Type, g *guard) bool {
	switch o := other.(type) {
	case *arrayType:
		return asg(t, o.element, g)
	case *tupleType:
		for _, e := range o.types {
			if !asg(t, e, g) {
				return false
			}
		}
		return true
	case *hashType:
		return asg(&richDataKeyType{}, o.key, g) && asg(t, o.value, g)
	case *structType:
		for _, m := range o.members {
			if !asg(t, m.typ, g) {
				return false
			}
		}
		return true
	default:
		return richDataScalar(o)
	}
}

func (richDataKeyType) isAssignable(other Type, _ *guard) bool {
	switch other.(type) {
	case *richDataKeyType, *stringType, *enumType, *patternType,
		*integerType, *floatType, *numericType:
		return true
	default:
		return false
	}
}

func (t *stringType) isAssignable(other Type, _ *guard) bool {
	switch o := other.(type) {
	case *stringType:
		return szWithin(t.minLen, t.maxLen, o.minLen, o.maxLen)
	case *enumType:
		for _, s := range o.values {
			n := int64(len([]rune(s)))
			if n < t.minLen || n > t.maxLen {
				return false
			}
		}
		return len(o.values) > 0
	case *patternType:
		// Pattern strings are unbounded in length, so only the default String
		// range accepts them.
		return t.minLen == 0 && t.maxLen == maxInt
	default:
		return false
	}
}

func (t *enumType) isAssignable(other Type, g *guard) bool {
	switch o := other.(type) {
	case *enumType:
		for _, s := range o.values {
			if !t.isInstance(s, g) {
				return false
			}
		}
		return len(o.values) > 0
	case *stringType:
		// A String type accepts many values; only a single fixed-length String
		// whose sole value is one of the enum members would qualify, which is not
		// decidable from the type alone.
		return false
	default:
		return false
	}
}

func (t *patternType) isAssignable(other Type, g *guard) bool {
	switch o := other.(type) {
	case *patternType:
		return t.String() == o.String()
	case *enumType:
		for _, s := range o.values {
			if !t.isInstance(s, g) {
				return false
			}
		}
		return len(o.values) > 0
	case *stringType:
		// A specific single-length String cannot be proven to match; a general
		// pattern set only accepts strings it matches, which is not decidable
		// from a String type. Not assignable.
		return false
	default:
		return false
	}
}

func (t *regexpType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*regexpType)
	if !ok {
		return false
	}
	if t.pattern == nil {
		return true
	}
	return o.pattern != nil && t.pattern.src == o.pattern.src
}

func (t *collectionType) isAssignable(other Type, _ *guard) bool {
	oMin, oMax, ok := collectionSize(other)
	return ok && szWithin(t.minSz, t.maxSz, oMin, oMax)
}

// collectionSize returns the size range of any collection-shaped type.
func collectionSize(t Type) (int64, int64, bool) {
	switch o := t.(type) {
	case *collectionType:
		return o.minSz, o.maxSz, true
	case *arrayType:
		return o.minSz, o.maxSz, true
	case *hashType:
		return o.minSz, o.maxSz, true
	case *tupleType:
		return o.minSz, o.maxSz, true
	case *structType:
		req := int64(0)
		for _, m := range o.members {
			if !m.optional() {
				req++
			}
		}
		return req, int64(len(o.members)), true
	default:
		return 0, 0, false
	}
}

func (t *arrayType) isAssignable(other Type, g *guard) bool {
	switch o := other.(type) {
	case *arrayType:
		return asg(t.element, o.element, g) && szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz)
	case *tupleType:
		for _, e := range o.types {
			if !asg(t.element, e, g) {
				return false
			}
		}
		return szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz)
	default:
		return false
	}
}

func (t *tupleType) isAssignable(other Type, g *guard) bool {
	switch o := other.(type) {
	case *tupleType:
		if !szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz) {
			return false
		}
		n := len(o.types)
		if len(t.types) > n {
			n = len(t.types)
		}
		for i := 0; i < n; i++ {
			if !asg(t.typeAt(i), o.typeAt(i), g) {
				return false
			}
		}
		return true
	case *arrayType:
		if !szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz) {
			return false
		}
		for _, e := range t.types {
			if !asg(e, o.element, g) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (t *hashType) isAssignable(other Type, g *guard) bool {
	switch o := other.(type) {
	case *hashType:
		return asg(t.key, o.key, g) && asg(t.value, o.value, g) &&
			szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz)
	case *structType:
		oMin, oMax, _ := collectionSize(o)
		if !szWithin(t.minSz, t.maxSz, oMin, oMax) {
			return false
		}
		for _, m := range o.members {
			if !asg(t.key, m.keyType(), g) {
				return false
			}
			if !asg(t.value, m.typ, g) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (t *structType) isAssignable(other Type, g *guard) bool {
	o, ok := other.(*structType)
	if !ok {
		return false
	}
	for _, m := range t.members {
		om, found := o.member(m.name)
		if !found {
			if m.optional() {
				continue
			}
			return false
		}
		if !asg(m.typ, om.typ, g) {
			return false
		}
		if !m.optional() && om.optional() {
			return false
		}
	}
	for _, om := range o.members {
		if _, found := t.member(om.name); !found {
			return false
		}
	}
	return true
}

func (t *structType) member(name string) (structMember, bool) {
	for _, m := range t.members {
		if m.name == name {
			return m, true
		}
	}
	return structMember{}, false
}

// keyType returns the String type describing a struct member's literal key.
func (m structMember) keyType() Type {
	n := int64(len([]rune(m.name)))
	return &stringType{minLen: n, maxLen: n}
}

func (t *typeType) isAssignable(other Type, g *guard) bool {
	o, ok := other.(*typeType)
	return ok && asg(t.typ, o.typ, g)
}

func (t *sensitiveType) isAssignable(other Type, g *guard) bool {
	o, ok := other.(*sensitiveType)
	return ok && asg(t.typ, o.typ, g)
}

func (t *variantType) isAssignable(other Type, g *guard) bool {
	for _, m := range t.types {
		if asg(m, other, g) {
			return true
		}
	}
	return false
}

func (t *optionalType) isAssignable(other Type, g *guard) bool {
	if _, ok := other.(*undefType); ok {
		return true
	}
	return asg(t.typ, other, g)
}

func (t *notUndefType) isAssignable(other Type, g *guard) bool {
	return !allowsUndef(other) && asg(t.typ, other, g)
}

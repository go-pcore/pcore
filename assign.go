package pcore

// assignable reports whether b is a subtype of a: every instance of b is an
// instance of a. It first distributes over the union-like shapes of b (Variant,
// Optional) so each concrete type's isAssignable only ever sees a "simple"
// argument, then defers to a.isAssignable.
func assignable(a, b Type) bool {
	if a.String() == b.String() {
		return true
	}
	switch bb := b.(type) {
	case *variantType:
		// An empty Variant has no instances and is vacuously a subtype of
		// everything; otherwise every member must be assignable.
		for _, m := range bb.types {
			if !assignable(a, m) {
				return false
			}
		}
		return true
	case *optionalType:
		return assignable(a, &undefType{}) && assignable(a, bb.typ)
	case *notUndefType:
		return assignable(a, bb.typ)
	}
	return a.isAssignable(b)
}

// allowsUndef reports whether a simple type's instance set includes undef.
func allowsUndef(t Type) bool {
	switch t.(type) {
	case *anyType, *undefType, dataType:
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
// Optional or NotUndef — those are peeled off by assignable).

func (anyType) isAssignable(Type) bool { return true }

func (undefType) isAssignable(other Type) bool { _, ok := other.(*undefType); return ok }

func (defaultTypeT) isAssignable(other Type) bool { _, ok := other.(*defaultTypeT); return ok }

func (booleanType) isAssignable(other Type) bool { _, ok := other.(*booleanType); return ok }

func (binaryType) isAssignable(other Type) bool { _, ok := other.(*binaryType); return ok }

func (timestampType) isAssignable(other Type) bool { _, ok := other.(*timestampType); return ok }

func (timespanType) isAssignable(other Type) bool { _, ok := other.(*timespanType); return ok }

func (numericType) isAssignable(other Type) bool {
	switch other.(type) {
	case *numericType, *integerType, *floatType:
		return true
	default:
		return false
	}
}

func (t *integerType) isAssignable(other Type) bool {
	o, ok := other.(*integerType)
	return ok && intWithin(t.min, t.max, o.min, o.max)
}

func (t *floatType) isAssignable(other Type) bool {
	o, ok := other.(*floatType)
	return ok && fltWithin(t.min, t.max, o.min, o.max)
}

func (scalarDataType) isAssignable(other Type) bool {
	switch other.(type) {
	case *scalarDataType, *integerType, *floatType, *numericType, *stringType, *enumType, *patternType, *booleanType:
		return true
	default:
		return false
	}
}

func (scalarType) isAssignable(other Type) bool {
	switch other.(type) {
	case *scalarType, *scalarDataType, *integerType, *floatType, *numericType,
		*stringType, *enumType, *patternType, *booleanType,
		*regexpType, *timestampType, *timespanType:
		return true
	default:
		return false
	}
}

func (dataType) isAssignable(other Type) bool {
	switch o := other.(type) {
	case *scalarDataType, *integerType, *floatType, *numericType, *stringType, *enumType, *patternType, *booleanType:
		return true
	case *undefType, dataType:
		return true
	case *arrayType:
		return assignable(dataType{}, o.element)
	case *tupleType:
		for _, e := range o.types {
			if !assignable(dataType{}, e) {
				return false
			}
		}
		return true
	case *hashType:
		return assignable(AnyString(), o.key) && assignable(dataType{}, o.value)
	case *structType:
		for _, m := range o.members {
			if !assignable(dataType{}, m.typ) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (t *stringType) isAssignable(other Type) bool {
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

func (t *enumType) isAssignable(other Type) bool {
	o, ok := other.(*enumType)
	if !ok {
		return false
	}
	for _, s := range o.values {
		if !t.isInstance(s) {
			return false
		}
	}
	return len(o.values) > 0
}

func (t *patternType) isAssignable(other Type) bool {
	switch o := other.(type) {
	case *patternType:
		return t.String() == o.String()
	case *enumType:
		for _, s := range o.values {
			if !t.isInstance(s) {
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

func (t *regexpType) isAssignable(other Type) bool {
	o, ok := other.(*regexpType)
	if !ok {
		return false
	}
	if t.pattern == nil {
		return true
	}
	return o.pattern != nil && t.pattern.src == o.pattern.src
}

func (t *collectionType) isAssignable(other Type) bool {
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

func (t *arrayType) isAssignable(other Type) bool {
	switch o := other.(type) {
	case *arrayType:
		return assignable(t.element, o.element) && szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz)
	case *tupleType:
		for _, e := range o.types {
			if !assignable(t.element, e) {
				return false
			}
		}
		return szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz)
	default:
		return false
	}
}

func (t *tupleType) isAssignable(other Type) bool {
	o, ok := other.(*tupleType)
	if !ok {
		return false
	}
	if !szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz) {
		return false
	}
	n := len(o.types)
	if len(t.types) > n {
		n = len(t.types)
	}
	for i := 0; i < n; i++ {
		if !assignable(t.typeAt(i), o.typeAt(i)) {
			return false
		}
	}
	return true
}

func (t *hashType) isAssignable(other Type) bool {
	switch o := other.(type) {
	case *hashType:
		return assignable(t.key, o.key) && assignable(t.value, o.value) &&
			szWithin(t.minSz, t.maxSz, o.minSz, o.maxSz)
	case *structType:
		oMin, oMax, _ := collectionSize(o)
		if !szWithin(t.minSz, t.maxSz, oMin, oMax) {
			return false
		}
		for _, m := range o.members {
			if !assignable(t.key, &stringType{minLen: int64(len([]rune(m.name))), maxLen: int64(len([]rune(m.name)))}) {
				return false
			}
			if !assignable(t.value, m.typ) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (t *structType) isAssignable(other Type) bool {
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
		if !assignable(m.typ, om.typ) {
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

func (t *typeType) isAssignable(other Type) bool {
	o, ok := other.(*typeType)
	return ok && assignable(t.typ, o.typ)
}

func (t *sensitiveType) isAssignable(other Type) bool {
	o, ok := other.(*sensitiveType)
	return ok && assignable(t.typ, o.typ)
}

func (t *variantType) isAssignable(other Type) bool {
	for _, m := range t.types {
		if assignable(m, other) {
			return true
		}
	}
	return false
}

func (t *optionalType) isAssignable(other Type) bool {
	if _, ok := other.(*undefType); ok {
		return true
	}
	return assignable(t.typ, other)
}

func (t *notUndefType) isAssignable(other Type) bool {
	return !allowsUndef(other) && assignable(t.typ, other)
}

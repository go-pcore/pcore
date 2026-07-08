package pcore

// Infer returns the most specific Pcore type of which v is an instance.
func Infer(v Value) Type {
	switch x := canon(v).(type) {
	case undefValue:
		return &undefType{}
	case defaultValue:
		return &defaultTypeT{}
	case bool:
		return &booleanType{}
	case int64:
		return &integerType{min: x, max: x}
	case float64:
		return &floatType{min: x, max: x}
	case string:
		n := int64(len([]rune(x)))
		return &stringType{minLen: n, maxLen: n}
	case []Value:
		return inferArray(x)
	case *Hash:
		return inferHash(x.entries)
	case *Regexp:
		return &regexpType{pattern: x}
	case *Binary:
		return &binaryType{}
	case *Timestamp:
		return &timestampType{}
	case *Timespan:
		return &timespanType{}
	case *Sensitive:
		return &sensitiveType{typ: Infer(x.inner)}
	case Type:
		return &typeType{typ: x}
	default:
		return &anyType{}
	}
}

func inferArray(a []Value) Type {
	n := int64(len(a))
	if n == 0 {
		return &arrayType{element: &anyType{}, minSz: 0, maxSz: 0}
	}
	el := Infer(a[0])
	for _, e := range a[1:] {
		el = CommonType(el, Infer(e))
	}
	return &arrayType{element: el, minSz: n, maxSz: n}
}

func inferHash(entries []HashEntry) Type {
	n := int64(len(entries))
	if n == 0 {
		return &hashType{key: &anyType{}, value: &anyType{}, minSz: 0, maxSz: 0}
	}
	k := Infer(entries[0].Key)
	val := Infer(entries[0].Value)
	for _, e := range entries[1:] {
		k = CommonType(k, Infer(e.Key))
		val = CommonType(val, Infer(e.Value))
	}
	return &hashType{key: k, value: val, minSz: n, maxSz: n}
}

// Generalize widens a type by dropping its range/size constraints and
// generalizing its parameters: Integer[3,3] becomes Integer, Array[Integer[1,1],
// 2, 2] becomes Array[Integer], and so on.
func Generalize(t Type) Type {
	switch x := t.(type) {
	case *integerType:
		return &integerType{min: minInt, max: maxInt}
	case *floatType:
		return &floatType{min: negInf(), max: posInf()}
	case *stringType:
		return &stringType{minLen: 0, maxLen: maxInt}
	case *enumType:
		return &stringType{minLen: 0, maxLen: maxInt}
	case *arrayType:
		return &arrayType{element: Generalize(x.element), minSz: 0, maxSz: maxInt}
	case *hashType:
		return &hashType{key: Generalize(x.key), value: Generalize(x.value), minSz: 0, maxSz: maxInt}
	case *tupleType:
		el := Type(&anyType{})
		if len(x.types) > 0 {
			el = Generalize(x.types[0])
			for _, e := range x.types[1:] {
				el = CommonType(el, Generalize(e))
			}
		}
		return &arrayType{element: el, minSz: 0, maxSz: maxInt}
	case *structType:
		k := Type(&stringType{minLen: 0, maxLen: maxInt})
		val := Type(&anyType{})
		if len(x.members) > 0 {
			val = Generalize(x.members[0].typ)
			for _, m := range x.members[1:] {
				val = CommonType(val, Generalize(m.typ))
			}
		}
		return &hashType{key: k, value: val, minSz: 0, maxSz: maxInt}
	case *optionalType:
		return &optionalType{typ: Generalize(x.typ)}
	case *variantType:
		vs := make([]Type, len(x.types))
		for i, m := range x.types {
			vs[i] = Generalize(m)
		}
		return &variantType{types: vs}
	case *sensitiveType:
		return &sensitiveType{typ: Generalize(x.typ)}
	default:
		return t
	}
}

// CommonType returns the narrowest single type that is a supertype of both a
// and b.
func CommonType(a, b Type) Type {
	if assignable(a, b) {
		return a
	}
	if assignable(b, a) {
		return b
	}
	// Same-kind numeric / string merges keep a tight result.
	if ai, ok := a.(*integerType); ok {
		if bi, ok := b.(*integerType); ok {
			return &integerType{min: minI(ai.min, bi.min), max: maxI(ai.max, bi.max)}
		}
	}
	if af, ok := a.(*floatType); ok {
		if bf, ok := b.(*floatType); ok {
			return &floatType{min: minF(af.min, bf.min), max: maxF(af.max, bf.max)}
		}
	}
	if isNumeric(a) && isNumeric(b) {
		return &numericType{}
	}
	if isStringy(a) && isStringy(b) {
		return &stringType{minLen: 0, maxLen: maxInt}
	}
	// Fall back to the narrowest well-known supertype accepting both.
	for _, super := range []Type{&scalarDataType{}, &scalarType{}, dataType{}, &collectionType{minSz: 0, maxSz: maxInt}} {
		if assignable(super, a) && assignable(super, b) {
			return super
		}
	}
	return &anyType{}
}

func isNumeric(t Type) bool {
	switch t.(type) {
	case *integerType, *floatType, *numericType:
		return true
	default:
		return false
	}
}

func isStringy(t Type) bool {
	switch t.(type) {
	case *stringType, *enumType, *patternType:
		return true
	default:
		return false
	}
}

func minI(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
func maxI(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

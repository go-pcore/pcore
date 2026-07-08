package pcore

import (
	"math"
	"strconv"
	"strings"
)

// Numeric bound sentinels for the default (unbounded) ends of a range.
const (
	minInt = int64(math.MinInt64)
	maxInt = int64(math.MaxInt64)
)

func negInf() float64 { return math.Inf(-1) }
func posInf() float64 { return math.Inf(1) }

// The concrete type structs. Each carries just enough state to reproduce its
// canonical string form and to answer instance/assignability queries.

type anyType struct{}
type scalarType struct{}
type scalarDataType struct{}
type dataType struct{}
type numericType struct{}
type booleanType struct{}
type undefType struct{}
type defaultTypeT struct{}
type binaryType struct{}
type timestampType struct{}
type timespanType struct{}

type integerType struct{ min, max int64 }
type floatType struct{ min, max float64 }
type stringType struct{ minLen, maxLen int64 }
type enumType struct {
	values []string
	ci     bool // case-insensitive
}
type patternType struct{ patterns []*Regexp }
type regexpType struct {
	pattern *Regexp // nil ⇒ generic Regexp
}
type collectionType struct{ minSz, maxSz int64 }
type arrayType struct {
	element      Type
	minSz, maxSz int64
}
type hashType struct {
	key, value   Type
	minSz, maxSz int64
}
type tupleType struct {
	types        []Type
	minSz, maxSz int64
}
type structMember struct {
	name   string
	keyOpt bool // key wrapped in Optional[...]
	typ    Type
}
type structType struct{ members []structMember }
type variantType struct{ types []Type }
type optionalType struct{ typ Type }
type notUndefType struct{ typ Type }
type typeType struct{ typ Type }
type sensitiveType struct{ typ Type }

// Exported constructors ------------------------------------------------------

// NewInteger returns Integer[min, max]. Use minInt / maxInt for open ends via
// [AnyInteger] instead when a fully generic Integer is wanted.
func NewInteger(min, max int64) Type { return &integerType{min: min, max: max} }

// AnyInteger returns the unbounded Integer type.
func AnyInteger() Type { return &integerType{min: minInt, max: maxInt} }

// NewFloat returns Float[min, max].
func NewFloat(min, max float64) Type { return &floatType{min: min, max: max} }

// AnyFloat returns the unbounded Float type.
func AnyFloat() Type { return &floatType{min: negInf(), max: posInf()} }

// NewString returns String[minLen, maxLen].
func NewString(minLen, maxLen int64) Type { return &stringType{minLen: minLen, maxLen: maxLen} }

// AnyString returns the unbounded String type.
func AnyString() Type { return &stringType{minLen: 0, maxLen: maxInt} }

// NewEnum returns Enum[values...].
func NewEnum(values ...string) Type { return &enumType{values: values} }

// NewArray returns Array[element, minSz, maxSz].
func NewArray(element Type, minSz, maxSz int64) Type {
	return &arrayType{element: element, minSz: minSz, maxSz: maxSz}
}

// NewHashType returns Hash[key, value, minSz, maxSz].
func NewHashType(key, value Type, minSz, maxSz int64) Type {
	return &hashType{key: key, value: value, minSz: minSz, maxSz: maxSz}
}

// NewVariant returns Variant[types...].
func NewVariant(types ...Type) Type { return &variantType{types: types} }

// NewOptional returns Optional[typ].
func NewOptional(typ Type) Type { return &optionalType{typ: typ} }

// Singleton accessors, handy for callers assembling types programmatically.

func AnyT() Type       { return &anyType{} }
func ScalarT() Type    { return &scalarType{} }
func DataT() Type      { return &dataType{} }
func BooleanT() Type   { return &booleanType{} }
func UndefT() Type     { return &undefType{} }
func DefaultT() Type   { return &defaultTypeT{} }
func NumericT() Type   { return &numericType{} }
func BinaryT() Type    { return &binaryType{} }
func TimestampT() Type { return &timestampType{} }
func TimespanT() Type  { return &timespanType{} }

// Name methods ---------------------------------------------------------------

func (anyType) Name() string         { return "Any" }
func (scalarType) Name() string      { return "Scalar" }
func (scalarDataType) Name() string  { return "ScalarData" }
func (dataType) Name() string        { return "Data" }
func (numericType) Name() string     { return "Numeric" }
func (booleanType) Name() string     { return "Boolean" }
func (undefType) Name() string       { return "Undef" }
func (defaultTypeT) Name() string    { return "Default" }
func (binaryType) Name() string      { return "Binary" }
func (timestampType) Name() string   { return "Timestamp" }
func (timespanType) Name() string    { return "Timespan" }
func (*integerType) Name() string    { return "Integer" }
func (*floatType) Name() string      { return "Float" }
func (*stringType) Name() string     { return "String" }
func (*enumType) Name() string       { return "Enum" }
func (*patternType) Name() string    { return "Pattern" }
func (*regexpType) Name() string     { return "Regexp" }
func (*collectionType) Name() string { return "Collection" }
func (*arrayType) Name() string      { return "Array" }
func (*hashType) Name() string       { return "Hash" }
func (*tupleType) Name() string      { return "Tuple" }
func (*structType) Name() string     { return "Struct" }
func (*variantType) Name() string    { return "Variant" }
func (*optionalType) Name() string   { return "Optional" }
func (*notUndefType) Name() string   { return "NotUndef" }
func (*typeType) Name() string       { return "Type" }
func (*sensitiveType) Name() string  { return "Sensitive" }

// String methods -------------------------------------------------------------

func (anyType) String() string        { return "Any" }
func (scalarType) String() string     { return "Scalar" }
func (scalarDataType) String() string { return "ScalarData" }
func (dataType) String() string       { return "Data" }
func (numericType) String() string    { return "Numeric" }
func (booleanType) String() string    { return "Boolean" }
func (undefType) String() string      { return "Undef" }
func (defaultTypeT) String() string   { return "Default" }
func (binaryType) String() string     { return "Binary" }
func (timestampType) String() string  { return "Timestamp" }
func (timespanType) String() string   { return "Timespan" }

func (t *integerType) String() string {
	if t.min == minInt && t.max == maxInt {
		return "Integer"
	}
	if t.max == maxInt {
		return "Integer[" + itoa(t.min) + "]"
	}
	lo := "default"
	if t.min != minInt {
		lo = itoa(t.min)
	}
	return "Integer[" + lo + ", " + itoa(t.max) + "]"
}

func (t *floatType) String() string {
	if math.IsInf(t.min, -1) && math.IsInf(t.max, 1) {
		return "Float"
	}
	if math.IsInf(t.max, 1) {
		return "Float[" + ftoa(t.min) + "]"
	}
	lo := "default"
	if !math.IsInf(t.min, -1) {
		lo = ftoa(t.min)
	}
	return "Float[" + lo + ", " + ftoa(t.max) + "]"
}

func (t *stringType) String() string {
	if t.minLen == 0 && t.maxLen == maxInt {
		return "String"
	}
	if t.maxLen == maxInt {
		return "String[" + itoa(t.minLen) + "]"
	}
	return "String[" + itoa(t.minLen) + ", " + itoa(t.maxLen) + "]"
}

func (t *enumType) String() string {
	parts := make([]string, 0, len(t.values)+1)
	for _, v := range t.values {
		parts = append(parts, quote(v))
	}
	if t.ci {
		parts = append(parts, "true")
	}
	return "Enum[" + strings.Join(parts, ", ") + "]"
}

func (t *patternType) String() string {
	parts := make([]string, len(t.patterns))
	for i, p := range t.patterns {
		parts[i] = p.String()
	}
	return "Pattern[" + strings.Join(parts, ", ") + "]"
}

func (t *regexpType) String() string {
	if t.pattern == nil {
		return "Regexp"
	}
	return "Regexp[" + t.pattern.String() + "]"
}

func (t *collectionType) String() string {
	if t.minSz == 0 && t.maxSz == maxInt {
		return "Collection"
	}
	return "Collection[" + sizeArgs(t.minSz, t.maxSz) + "]"
}

func (t *arrayType) String() string {
	genericEl := t.element.String() == "Any"
	defaultSz := t.minSz == 0 && t.maxSz == maxInt
	if genericEl && defaultSz {
		return "Array"
	}
	if defaultSz {
		return "Array[" + t.element.String() + "]"
	}
	return "Array[" + t.element.String() + ", " + sizeArgs(t.minSz, t.maxSz) + "]"
}

func (t *hashType) String() string {
	genericKV := t.key.String() == "Any" && t.value.String() == "Any"
	defaultSz := t.minSz == 0 && t.maxSz == maxInt
	if genericKV && defaultSz {
		return "Hash"
	}
	if defaultSz {
		return "Hash[" + t.key.String() + ", " + t.value.String() + "]"
	}
	return "Hash[" + t.key.String() + ", " + t.value.String() + ", " + sizeArgs(t.minSz, t.maxSz) + "]"
}

func (t *tupleType) String() string {
	parts := make([]string, 0, len(t.types)+2)
	for _, e := range t.types {
		parts = append(parts, e.String())
	}
	if !(t.minSz == int64(len(t.types)) && t.maxSz == int64(len(t.types))) {
		parts = append(parts, sizeArgs(t.minSz, t.maxSz))
	}
	return "Tuple[" + strings.Join(parts, ", ") + "]"
}

func (t *structType) String() string {
	parts := make([]string, len(t.members))
	for i, m := range t.members {
		key := quote(m.name)
		if m.keyOpt {
			key = "Optional[" + key + "]"
		}
		parts[i] = key + " => " + m.typ.String()
	}
	return "Struct[{" + strings.Join(parts, ", ") + "}]"
}

func (t *variantType) String() string {
	parts := make([]string, len(t.types))
	for i, e := range t.types {
		parts[i] = e.String()
	}
	return "Variant[" + strings.Join(parts, ", ") + "]"
}

func (t *optionalType) String() string { return "Optional[" + t.typ.String() + "]" }

func (t *notUndefType) String() string {
	if _, ok := t.typ.(*anyType); ok {
		return "NotUndef"
	}
	return "NotUndef[" + t.typ.String() + "]"
}

func (t *typeType) String() string {
	if _, ok := t.typ.(*anyType); ok {
		return "Type"
	}
	return "Type[" + t.typ.String() + "]"
}

func (t *sensitiveType) String() string {
	if _, ok := t.typ.(*anyType); ok {
		return "Sensitive"
	}
	return "Sensitive[" + t.typ.String() + "]"
}

// Small formatting helpers ---------------------------------------------------

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

// ftoa formats a finite float bound. Infinite ends are never passed here: the
// String methods render them as "default" or collapse them away.
func ftoa(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

// sizeArgs renders a size range as one or two arguments.
func sizeArgs(min, max int64) string {
	if max == maxInt {
		return itoa(min)
	}
	return itoa(min) + ", " + itoa(max)
}

// quote renders s as a single-quoted Puppet string.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\'':
			b.WriteString(`\'`)
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

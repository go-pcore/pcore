package pcore

import (
	"testing"
	"time"
)

// TestAllNames exercises Name on every concrete type.
func TestAllNames(t *testing.T) {
	want := map[string]string{
		"Any": "Any", "Scalar": "Scalar", "ScalarData": "ScalarData",
		"Data": "Data", "Numeric": "Numeric", "Boolean": "Boolean",
		"Undef": "Undef", "Default": "Default", "Binary": "Binary",
		"Timestamp": "Timestamp", "Timespan": "Timespan",
		"Integer[1, 2]": "Integer", "Float[1.0, 2.0]": "Float",
		"String[1, 2]": "String", "Enum['a']": "Enum",
		"Pattern[/a/]": "Pattern", "Regexp[/a/]": "Regexp",
		"Collection[1]": "Collection", "Array[Integer]": "Array",
		"Hash[String, Integer]": "Hash", "Tuple[Integer]": "Tuple",
		"Struct[{'a' => Integer}]": "Struct", "Variant[Integer, String]": "Variant",
		"Optional[Integer]": "Optional", "NotUndef[Integer]": "NotUndef",
		"Type[Integer]": "Type", "Sensitive[Integer]": "Sensitive",
	}
	for expr, name := range want {
		if got := mustParse(t, expr).Name(); got != name {
			t.Errorf("Name(%s) = %q, want %q", expr, got, name)
		}
	}
}

func TestExportedConstructors(t *testing.T) {
	if NewFloat(1, 2).String() != "Float[1, 2]" {
		t.Error("NewFloat")
	}
	if NewString(1, 2).String() != "String[1, 2]" {
		t.Error("NewString")
	}
	if NewInteger(1, 2).String() != "Integer[1, 2]" {
		t.Error("NewInteger")
	}
	if NewEnum("a", "b").String() != "Enum['a', 'b']" {
		t.Error("NewEnum")
	}
	if AnyT().String() != "Any" {
		t.Error("AnyT")
	}
}

// TestCommonTypeReversed hits the opposite arms of the min/max helpers.
func TestCommonTypeReversed(t *testing.T) {
	if got := CommonType(mustParse(t, "Integer[5, 7]"), mustParse(t, "Integer[1, 3]")).String(); got != "Integer[1, 7]" {
		t.Errorf("int reversed: %q", got)
	}
	if got := CommonType(mustParse(t, "Float[5.0, 7.0]"), mustParse(t, "Float[1.0, 3.0]")).String(); got != "Float[1, 7]" {
		t.Errorf("float reversed: %q", got)
	}
}

func TestInferMultiEntryHash(t *testing.T) {
	h := NewHash(
		HashEntry{"a", int64(1)},
		HashEntry{"bb", int64(5)},
	)
	// keys String[1,1] & String[2,2] -> String[1,2]; values Integer[1,1] & [5,5] -> [1,5]
	if got := Infer(h).String(); got != "Hash[String, Integer[1, 5], 2, 2]" {
		t.Errorf("Infer multi hash = %q", got)
	}
}

func TestDataHashNonDataValue(t *testing.T) {
	reAB, _ := NewRegexp("ab")
	if IsInstance(DataT(), NewHash(HashEntry{"a", reAB})) {
		t.Error("Data hash with non-data value should not be an instance")
	}
}

func TestCollectionFromStructWithOptional(t *testing.T) {
	// Struct with an optional member exercises the optional arm of the size count.
	if !IsAssignable(mustParse(t, "Collection"), mustParse(t, "Struct[{'a' => Integer, Optional['b'] => String}]")) {
		t.Error("Collection should accept a struct with an optional member")
	}
	if !IsAssignable(mustParse(t, "Hash[String, Data, 0, 5]"), mustParse(t, "Struct[{'a' => Integer, Optional['b'] => String}]")) {
		t.Error("Hash should accept a compatible struct with an optional member")
	}
}

func TestBuildBoundErrors(t *testing.T) {
	bad := []string{
		"Integer[1, 'x']",
		"Integer['x', 1]",
		"Float[1, 'x']",
		"Float['x', 1]",
		"String[1, 'x']",
		"Array[Integer, 1, 'x']",
		"Collection[1, 'x']",
		"Hash[String, Integer, 1, 'x']",
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestBuildBoundValidTwoArg(t *testing.T) {
	// second bound present and valid (case-2 success arms).
	for _, s := range []string{"Integer[2, 8]", "Float[2, 8]", "String[2, 8]", "Array[Integer, 2, 8]", "Collection[2, 8]"} {
		mustParse(t, s)
	}
}

func TestFloatExponentPlus(t *testing.T) {
	if got := mustParse(t, "Float[1e+3]").String(); got != "Float[1000]" {
		t.Errorf("Float[1e+3] = %q", got)
	}
}

func TestStringEscapes(t *testing.T) {
	ty := mustParse(t, `Enum['a\nb\tc\rd\qe']`)
	e := ty.(*enumType)
	if e.values[0] != "a\nb\tc\rdqe" { // \q is an unknown escape -> literal q
		t.Errorf("escapes decoded to %q", e.values[0])
	}
}

func TestUnterminatedBackslash(t *testing.T) {
	// a trailing backslash inside a string with no following char.
	if _, err := Parse("Enum['a\\"); err == nil {
		t.Error("trailing backslash string should error")
	}
}

func TestRegexpEscapes(t *testing.T) {
	// \/ becomes /, other escapes are preserved.
	ty := mustParse(t, `Regexp[/a\/b\.c/]`)
	if got := ty.(*regexpType).pattern.src; got != `a/b\.c` {
		t.Errorf("regexp escapes = %q", got)
	}
}

func TestMoreParseErrors(t *testing.T) {
	for _, s := range []string{
		"String['x', 1]",       // buildString case-2 low bound error
		"Regexp[/(/]",          // invalid regexp literal param
		"Variant[Foo]",         // nested unknown type
		"Struct[{'a' => Foo}]", // unknown struct value type
	} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestTimespanTimestampDisjoint(t *testing.T) {
	if IsAssignable(mustParse(t, "Timespan"), mustParse(t, "Timestamp")) {
		t.Error("Timespan should not be assignable from Timestamp")
	}
}

func TestCollectionFromCollection(t *testing.T) {
	if !IsAssignable(mustParse(t, "Collection[0, 5]"), mustParse(t, "Collection[1, 2]")) {
		t.Error("Collection[0,5] should accept Collection[1,2]")
	}
}

func TestToDataNonStringKeyBadValue(t *testing.T) {
	// non-string key hash whose value fails to serialize.
	if _, err := ToData(NewHash(HashEntry{int64(1), make(chan int)})); err == nil {
		t.Error("bad value under a non-string key should error")
	}
}

func TestFromDataNonStringPayloads(t *testing.T) {
	for _, ptype := range []string{"Binary", "Timestamp", "Timespan", "Type"} {
		v := NewHash(HashEntry{"__ptype", ptype}, HashEntry{"__pvalue", int64(1)})
		if _, err := FromData(v); err == nil {
			t.Errorf("FromData(%s with int payload) should error", ptype)
		}
	}
}

func TestIntegerOverflowParam(t *testing.T) {
	if _, err := Parse("Integer[99999999999999999999]"); err == nil {
		t.Error("int64-overflowing bound should error")
	}
}

func TestSizeParamsLowError(t *testing.T) {
	if _, err := Parse("Array[Integer, 'x', 1]"); err == nil {
		t.Error("bad low size bound should error")
	}
}

func TestReservedPValueKeyHash(t *testing.T) {
	orig := NewHash(HashEntry{"__pvalue", int64(1)}, HashEntry{"x", int64(2)})
	d, err := ToData(orig)
	if err != nil {
		t.Fatal(err)
	}
	back, err := FromData(d)
	if err != nil {
		t.Fatal(err)
	}
	if !equalValue(orig, back) {
		t.Error("reserved __pvalue key hash round-trip failed")
	}
}

func TestTupleSelfLongerAssignable(t *testing.T) {
	// self tuple has more declared types than the source; the repeating-position
	// walk must still run.
	a := mustParse(t, "Tuple[Integer, String, 0, 5]")
	b := mustParse(t, "Tuple[Integer, 1, 3]")
	if IsAssignable(a, b) {
		t.Error("Tuple[Integer,String] should not accept a Tuple of only Integers")
	}
}

func TestTupleVariableAssignable(t *testing.T) {
	// Variable-size tuple: the repeating last type governs extra positions.
	a := mustParse(t, "Tuple[Integer, 1, 3]")
	b := mustParse(t, "Tuple[Integer, Integer, Integer]")
	if !IsAssignable(a, b) {
		t.Error("variable tuple should accept a 3-integer tuple")
	}
}

func TestTimestampParsePrecision(t *testing.T) {
	ts := NewTimestamp(time.Date(2026, 3, 4, 5, 6, 7, 123456789, time.UTC))
	d, _ := ToData(ts)
	back, err := FromData(d)
	if err != nil {
		t.Fatal(err)
	}
	if !equalValue(ts, back) {
		t.Errorf("timestamp precision lost: %v vs %v", ts, back)
	}
}

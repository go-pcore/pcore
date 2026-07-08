package pcore

import "testing"

func TestIsAssignable(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		// Any is the top.
		{"Any", "Integer", true},
		{"Any", "Undef", true},
		{"Integer", "Any", false},
		// Reflexive.
		{"Integer", "Integer", true},
		// Integer ranges.
		{"Integer[0, 10]", "Integer[2, 5]", true},
		{"Integer[0, 10]", "Integer[2, 20]", false},
		{"Integer", "Integer[2, 5]", true},
		{"Integer", "Float", false},
		// Float ranges.
		{"Float[0.0, 10.0]", "Float[1.0, 2.0]", true},
		{"Float[0.0, 1.0]", "Float[1.0, 2.0]", false},
		// Numeric.
		{"Numeric", "Integer", true},
		{"Numeric", "Float", true},
		{"Numeric", "Numeric", true},
		{"Numeric", "String", false},
		// ScalarData / Scalar.
		{"ScalarData", "Integer[1, 3]", true},
		{"ScalarData", "String", true},
		{"ScalarData", "Boolean", true},
		{"ScalarData", "Numeric", true},
		{"ScalarData", "Enum['a']", true},
		{"ScalarData", "Regexp", false},
		{"Scalar", "Regexp", true},
		{"Scalar", "Timestamp", true},
		{"Scalar", "Timespan", true},
		{"Scalar", "ScalarData", true},
		{"Scalar", "Array[Integer]", false},
		// Data.
		{"Data", "Integer", true},
		{"Data", "Undef", true},
		{"Data", "Array[Integer]", true},
		{"Data", "Array[Any]", false},
		{"Data", "Hash[String, Integer]", true},
		{"Data", "Hash[Integer, Integer]", false},
		{"Data", "Tuple[Integer, String]", true},
		{"Data", "Tuple[Integer, Any]", false},
		{"Data", "Struct[{'a' => Integer}]", true},
		{"Data", "Struct[{'a' => Any}]", false},
		{"Data", "Data", true},
		{"Data", "Regexp", false},
		// String / Enum / Pattern.
		{"String", "String[1, 3]", true},
		{"String[1, 3]", "String", false},
		{"String", "Enum['a', 'bb']", true},
		{"String[1, 1]", "Enum['a', 'bb']", false},
		{"String", "Pattern[/x/]", true},
		{"String[1, 3]", "Pattern[/x/]", false},
		{"String", "Integer", false},
		{"Enum['a', 'b', 'c']", "Enum['a', 'b']", true},
		{"Enum['a']", "Enum['a', 'b']", false},
		{"Enum['a']", "String", false},
		{"Pattern[/a/]", "Pattern[/a/]", true},
		{"Pattern[/a/]", "Pattern[/b/]", false},
		{"Pattern[/^a/]", "Enum['abc', 'ax']", true},
		{"Pattern[/^a/]", "Enum['xyz']", false},
		{"Pattern[/a/]", "String", false},
		{"Pattern[/a/]", "Integer", false},
		// Regexp.
		{"Regexp", "Regexp[/a/]", true},
		{"Regexp[/a/]", "Regexp[/a/]", true},
		{"Regexp[/a/]", "Regexp", false},
		{"Regexp", "Integer", false},
		// Boolean / Undef / Default / Binary / Timestamp / Timespan.
		{"Boolean", "Boolean", true},
		{"Boolean", "Integer", false},
		{"Undef", "Undef", true},
		{"Undef", "Integer", false},
		{"Default", "Default", true},
		{"Default", "Integer", false},
		{"Binary", "Binary", true},
		{"Binary", "Integer", false},
		{"Timestamp", "Timestamp", true},
		{"Timespan", "Timespan", true},
		{"Timestamp", "Timespan", false},
		// Collection.
		{"Collection", "Array[Integer]", true},
		{"Collection", "Hash[String, Integer]", true},
		{"Collection", "Tuple[Integer]", true},
		{"Collection", "Struct[{'a' => Integer}]", true},
		{"Collection[3, 5]", "Array[Integer, 1, 2]", false},
		{"Collection", "Integer", false},
		// Array.
		{"Array[Integer]", "Array[Integer[1, 3]]", true},
		{"Array[Integer]", "Array[String]", false},
		{"Array[Integer, 0, 5]", "Array[Integer, 1, 3]", true},
		{"Array[Integer, 2, 5]", "Array[Integer, 1, 3]", false},
		{"Array[Integer]", "Tuple[Integer, Integer]", true},
		{"Array[Integer]", "Tuple[Integer, String]", false},
		{"Array[Integer]", "Integer", false},
		// Tuple.
		{"Tuple[Integer, String]", "Tuple[Integer, String]", true},
		{"Tuple[Integer, String]", "Tuple[Integer[1, 3], String]", true},
		{"Tuple[Integer, String]", "Tuple[Integer, Integer]", false},
		{"Tuple[Integer, String]", "Tuple[Integer]", false},
		{"Tuple[Integer, String]", "Array[Integer]", false},
		// Hash.
		{"Hash[String, Integer]", "Hash[String, Integer[1, 3]]", true},
		{"Hash[String, Integer]", "Hash[String, String]", false},
		{"Hash[String, Integer, 0, 5]", "Hash[String, Integer, 1, 2]", true},
		{"Hash[String, Integer, 2, 5]", "Hash[String, Integer, 1, 2]", false},
		{"Hash[String, Integer]", "Struct[{'a' => Integer}]", true},
		{"Hash[String, Integer]", "Struct[{'a' => String}]", false},
		{"Hash[String, Integer, 2, 5]", "Struct[{'a' => Integer}]", false},
		{"Hash[Enum['a'], Integer]", "Struct[{'b' => Integer}]", false},
		{"Hash[String, Integer]", "Integer", false},
		// Struct.
		{"Struct[{'a' => Integer}]", "Struct[{'a' => Integer[1, 3]}]", true},
		{"Struct[{'a' => Integer}]", "Struct[{'a' => String}]", false},
		{"Struct[{'a' => Integer, Optional['b'] => String}]", "Struct[{'a' => Integer}]", true},
		{"Struct[{'a' => Integer}]", "Struct[{'a' => Integer, 'b' => String}]", false}, // extra key
		{"Struct[{'a' => Integer}]", "Struct[{'b' => Integer}]", false},                // missing required
		{"Struct[{'a' => Integer}]", "Struct[{Optional['a'] => Integer}]", false},      // required vs optional
		{"Struct[{'a' => Integer}]", "Hash[String, Integer]", false},
		// Type.
		{"Type", "Type[Integer]", true},
		{"Type[Integer]", "Type[Integer[1, 3]]", true},
		{"Type[Integer]", "Type[String]", false},
		{"Type[Integer]", "Integer", false},
		// Sensitive.
		{"Sensitive", "Sensitive[Integer]", true},
		{"Sensitive[Integer]", "Sensitive[Integer[1, 3]]", true},
		{"Sensitive[Integer]", "Sensitive[String]", false},
		{"Sensitive[Integer]", "Integer", false},
		// Variant (target and source).
		{"Variant[Integer, String]", "Integer", true},
		{"Variant[Integer, String]", "Boolean", false},
		{"Variant[Integer, String]", "Variant[Integer, String]", true},
		{"Integer", "Variant[Integer[1, 3], Integer[5, 7]]", true},
		{"Integer", "Variant[Integer, String]", false},
		// Optional (target and source).
		{"Optional[Integer]", "Undef", true},
		{"Optional[Integer]", "Integer", true},
		{"Optional[Integer]", "String", false},
		{"Integer", "Optional[Integer]", false}, // Optional includes Undef
		{"Variant[Undef, Integer]", "Optional[Integer]", true},
		// NotUndef.
		{"NotUndef[Integer]", "Integer", true},
		{"NotUndef[Integer]", "Undef", false},
		{"NotUndef[Integer]", "Any", false}, // Any allows undef
		{"NotUndef", "Any", false},
		{"Integer", "NotUndef[Integer]", true},
	}
	for _, c := range cases {
		a, b := mustParse(t, c.a), mustParse(t, c.b)
		if got := IsAssignable(a, b); got != c.want {
			t.Errorf("IsAssignable(%s, %s) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestAssignableEmptyVariant(t *testing.T) {
	empty := &variantType{}
	if !IsAssignable(mustParse(t, "Integer"), empty) {
		t.Error("empty Variant should be assignable to any concrete type (vacuous)")
	}
	if !IsAssignable(mustParse(t, "Any"), empty) {
		t.Error("empty Variant should be assignable to Any")
	}
}

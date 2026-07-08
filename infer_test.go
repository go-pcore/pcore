package pcore

import (
	"testing"
	"time"
)

func TestInfer(t *testing.T) {
	reAB, _ := NewRegexp("ab")
	cases := []struct {
		val  Value
		want string
	}{
		{nil, "Undef"},
		{Undef, "Undef"},
		{Default, "Default"},
		{true, "Boolean"},
		{int64(5), "Integer[5, 5]"},
		{5, "Integer[5, 5]"},
		{1.5, "Float[1.5, 1.5]"},
		{"abc", "String[3, 3]"},
		{[]Value{}, "Array[Any, 0, 0]"},
		{[]Value{int64(1), int64(2)}, "Array[Integer[1, 2], 2, 2]"},
		{[]Value{int64(1), "x"}, "Array[ScalarData, 2, 2]"},
		{NewHash(), "Hash[Any, Any, 0, 0]"},
		{NewHash(HashEntry{"a", int64(1)}), "Hash[String[1, 1], Integer[1, 1], 1, 1]"},
		{reAB, "Regexp[/ab/]"},
		{NewBinary([]byte{1}), "Binary"},
		{NewTimestamp(time.Unix(0, 0)), "Timestamp"},
		{NewTimespan(time.Second), "Timespan"},
		{NewSensitive("x"), "Sensitive[String[1, 1]]"},
		{NewInteger(1, 3), "Type[Integer[1, 3]]"},
		{make(chan int), "Any"},
	}
	for _, c := range cases {
		if got := Infer(c.val).String(); got != c.want {
			t.Errorf("Infer(%v).String() = %q, want %q", c.val, got, c.want)
		}
	}
}

func TestGeneralize(t *testing.T) {
	cases := map[string]string{
		"Integer[1, 3]":                                       "Integer",
		"Float[1.0, 3.0]":                                     "Float",
		"String[1, 3]":                                        "String",
		"Enum['a', 'b']":                                      "String",
		"Array[Integer[1, 3], 2, 2]":                          "Array[Integer]",
		"Hash[String[1, 1], Integer[1, 1], 1, 1]":             "Hash[String, Integer]",
		"Tuple[Integer[1, 1], Integer[2, 2]]":                 "Array[Integer]",
		"Tuple[Integer[1, 1], String[1, 1]]":                  "Array[ScalarData]",
		"Struct[{'a' => Integer[1, 1]}]":                      "Hash[String, Integer]",
		"Struct[{'a' => Integer[1, 1], 'b' => String[1, 1]}]": "Hash[String, ScalarData]",
		"Optional[Integer[1, 3]]":                             "Optional[Integer]",
		"Variant[Integer[1, 1], Integer[2, 2]]":               "Variant[Integer, Integer]",
		"Sensitive[Integer[1, 3]]":                            "Sensitive[Integer]",
		"Boolean":                                             "Boolean",
	}
	for in, want := range cases {
		if got := Generalize(mustParse(t, in)).String(); got != want {
			t.Errorf("Generalize(%s).String() = %q, want %q", in, got, want)
		}
	}
}

func TestGeneralizeEmptyContainers(t *testing.T) {
	if got := Generalize(&tupleType{minSz: 0, maxSz: 0}).String(); got != "Array" {
		t.Errorf("Generalize(empty Tuple) = %q, want Array", got)
	}
	if got := Generalize(&structType{}).String(); got != "Hash[String, Any]" {
		t.Errorf("Generalize(empty Struct) = %q", got)
	}
}

func TestCommonType(t *testing.T) {
	cases := []struct {
		a, b string
		want string
	}{
		{"Integer[1, 3]", "Integer[1, 10]", "Integer[1, 10]"}, // a assignable from b? no; b from a yes
		{"Integer[1, 3]", "Integer[5, 7]", "Integer[1, 7]"},   // merge
		{"Float[1.0, 3.0]", "Float[5.0, 7.0]", "Float[1, 7]"},
		{"Integer[1, 3]", "Float[5.0, 7.0]", "Numeric"},
		{"Enum['a']", "Enum['b']", "String"},
		{"String[1, 3]", "Pattern[/x/]", "String"},
		{"Integer", "String", "ScalarData"},
		{"Regexp", "Integer", "Scalar"},
		{"Array[Integer]", "Integer", "Data"},
		{"Array[Any]", "Hash[Any, Any]", "Collection"},
		{"Array[Any]", "Integer", "Any"},
	}
	for _, c := range cases {
		got := CommonType(mustParse(t, c.a), mustParse(t, c.b)).String()
		if got != c.want {
			t.Errorf("CommonType(%s, %s) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

func TestCommonTypeSuperDirection(t *testing.T) {
	// a assignable from b ⇒ a is the common type.
	if got := CommonType(mustParse(t, "Integer"), mustParse(t, "Integer[1, 3]")).String(); got != "Integer" {
		t.Errorf("got %q, want Integer", got)
	}
}

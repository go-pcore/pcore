package pcore

import "testing"

func mustParse(t *testing.T, s string) Type {
	t.Helper()
	ty, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", s, err)
	}
	return ty
}

// TestRoundTrip verifies every canonical String form parses back to itself.
func TestRoundTrip(t *testing.T) {
	canonical := []string{
		"Any", "Scalar", "ScalarData", "Data", "Numeric", "Boolean",
		"Undef", "Default", "Binary", "Timestamp", "Timespan",
		"Integer", "Integer[1]", "Integer[1, 10]", "Integer[default, 10]",
		"Float", "Float[1.5]", "Float[1.5, 3.5]", "Float[default, 3.5]",
		"String", "String[3]", "String[1, 10]",
		"Enum['a', 'b']", "Enum['a', true]",
		"Pattern[/ab/, /cd/]",
		"Regexp", "Regexp[/ab/]",
		"Collection", "Collection[1]", "Collection[1, 3]",
		"Array", "Array[Integer]", "Array[Integer, 1]", "Array[Integer, 1, 3]",
		"Array[Any, 2, 2]",
		"Hash", "Hash[String, Integer]", "Hash[String, Integer, 1]", "Hash[String, Integer, 1, 3]",
		"Hash[Any, Any, 1, 3]",
		"Tuple[Integer, String]", "Tuple[Integer, 1, 3]",
		"Struct[{'a' => Integer, Optional['b'] => String}]",
		"Variant[Integer, String]",
		"Optional[Integer]",
		"NotUndef", "NotUndef[String]",
		"Type", "Type[Integer]",
		"Sensitive", "Sensitive[String]",
	}
	for _, s := range canonical {
		ty := mustParse(t, s)
		if got := ty.String(); got != s {
			t.Errorf("round-trip mismatch: Parse(%q).String() = %q", s, got)
		}
	}
}

// TestParseNormalization checks non-canonical spellings still parse correctly.
func TestParseNormalization(t *testing.T) {
	cases := map[string]string{
		"Integer[1,10]":                      "Integer[1, 10]",
		"  Array[ Integer ] ":                "Array[Integer]",
		"Enum[ 'x' ]":                        "Enum['x']",
		"String[default, 5]":                 "String[0, 5]",
		"Float[1e3]":                         "Float[1000]",
		"Float[1.5e-3, 2]":                   "Float[0.0015, 2]",
		"Regexp['ab']":                       "Regexp[/ab/]",
		"Pattern['ab']":                      "Pattern[/ab/]",
		`Enum["dq"]`:                         "Enum['dq']",
		"Struct[{mode => Integer}]":          "Struct[{'mode' => Integer}]",
		"Struct[{NotUndef['k'] => Integer}]": "Struct[{'k' => Integer}]",
	}
	for in, want := range cases {
		if got := mustParse(t, in).String(); got != want {
			t.Errorf("Parse(%q).String() = %q, want %q", in, got, want)
		}
	}
}

func TestStringEscaping(t *testing.T) {
	ty := NewEnum("a'b", `c\d`)
	want := `Enum['a\'b', 'c\\d']`
	if ty.String() != want {
		t.Fatalf("escaping: got %q want %q", ty.String(), want)
	}
	if mustParse(t, want).String() != want {
		t.Fatalf("escaped round-trip failed")
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{
		"",                 // empty
		"123",              // not a name
		"Foo",              // unknown type
		"Integer x",        // trailing input
		"Integer[]",        // empty params
		"Any[1]",           // nullary with params
		"Integer[1, 2, 3]", // too many
		"Integer['x']",     // bad bound
		"Integer[1 2]",     // missing comma
		"Float[1, 2, 3]",
		"Float['x']",
		"String[1, 2, 3]",
		"String['x']",
		"Enum",                 // needs a value
		"Enum[/x/]",            // not a string
		"Enum[true]",           // only ci, no values
		"Enum['a', false]",     // trailing false
		"Enum['a', true, 'b']", // true not last
		"Pattern",
		"Pattern[1]",
		"Pattern['(']", // invalid regexp string
		"Regexp[/a/, /b/]",
		"Regexp[1]",
		"Regexp['(']",
		"Collection[1, 2, 3]",
		"Array[1]", // element not a type
		"Array[Integer, 'x']",
		"Array[Integer, 1, 2, 3]",
		"Hash[String]", // needs both
		"Hash[1, 2]",   // key/value not types
		"Hash[String, Integer, 'x']",
		"Tuple",
		"Tuple[1]", // no leading type
		"Tuple[Integer, 'x']",
		"Struct[Integer]", // not a struct body
		"Struct[{'a' => Integer}, 1]",
		"Variant",
		"Variant[1]",
		"Optional[Integer, String]",
		"Optional[1]",
		"Type[1]",
		"Sensitive[1, 2]",
		"NotUndef[1]",
		"Enum['a",                                // unterminated string
		"Regexp[/a",                              // unterminated regexp
		"Struct[{'a' = Integer}]",                // bad '='
		"Integer[#]",                             // unexpected char
		"Array[Integer, -]",                      // malformed number
		"Array[,]",                               // unexpected token in params
		"Struct[{'a' Integer}]",                  // missing arrow
		"Struct[{1 => Integer}]",                 // bad key
		"Struct[{Optional 'a' => Integer}]",      // Optional without bracket
		"Struct[{Optional[1] => Integer}]",       // Optional key not string
		"Struct[{Optional['a' => Integer}]",      // Optional key missing ']'
		"Struct[{'a' => Integer",                 // unterminated body
		"Struct[{'a' => Integer 'b' => String}]", // missing comma
	}
	for _, s := range bad {
		if ty, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) = %v, want error", s, ty)
		}
	}
}

func TestParseErrorMessage(t *testing.T) {
	_, err := Parse("Foo")
	if err == nil {
		t.Fatal("want error")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("want *ParseError, got %T", err)
	}
	if pe.Error() == "" {
		t.Fatal("empty error message")
	}
}

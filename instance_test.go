package pcore

import (
	"testing"
	"time"
)

func TestIsInstance(t *testing.T) {
	reAB, _ := NewRegexp("ab")
	ts := NewTimestamp(time.Unix(0, 0))
	tsp := NewTimespan(time.Second)
	bin := NewBinary([]byte{1, 2})
	sens := NewSensitive("secret")
	h := NewHash(HashEntry{"a", int64(1)}, HashEntry{"b", int64(2)})
	hInt := NewHash(HashEntry{int64(1), "x"})
	arr := []Value{int64(1), int64(2)}

	cases := []struct {
		typ  string
		val  Value
		want bool
	}{
		{"Any", nil, true},
		{"Any", int64(3), true},
		{"Undef", Undef, true},
		{"Undef", nil, true},
		{"Undef", int64(1), false},
		{"Default", Default, true},
		{"Default", int64(1), false},
		{"Boolean", true, true},
		{"Boolean", 1, false},
		{"Integer", int64(5), true},
		{"Integer", 5, true}, // canonicalized
		{"Integer", 1.5, false},
		{"Integer[1, 10]", int64(5), true},
		{"Integer[1, 10]", int64(20), false},
		{"Float", 1.5, true},
		{"Float[0.0, 2.0]", 1.5, true},
		{"Float[0.0, 2.0]", 3.0, false},
		{"Float", int64(1), false},
		{"Numeric", int64(1), true},
		{"Numeric", 1.5, true},
		{"Numeric", "x", false},
		{"String", "abc", true},
		{"String", int64(1), false},
		{"String[1, 2]", "ab", true},
		{"String[1, 2]", "abc", false},
		{"Enum['a', 'b']", "a", true},
		{"Enum['a', 'b']", "c", false},
		{"Enum['a', 'b']", int64(1), false},
		{"Enum['a', true]", "A", true}, // case-insensitive
		{"Pattern[/^a/]", "abc", true},
		{"Pattern[/^a/]", "xyz", false},
		{"Pattern[/^a/]", int64(1), false},
		{"Regexp", reAB, true},
		{"Regexp", "ab", false},
		{"Regexp[/ab/]", reAB, true},
		{"Regexp[/cd/]", reAB, false},
		{"ScalarData", int64(1), true},
		{"ScalarData", reAB, false},
		{"Scalar", "x", true},
		{"Scalar", ts, true},
		{"Scalar", tsp, true},
		{"Scalar", reAB, true},
		{"Scalar", arr, false},
		{"Data", int64(1), true},
		{"Data", Undef, true},
		{"Data", arr, true},
		{"Data", h, true},
		{"Data", hInt, false}, // non-string key
		{"Data", []Value{reAB}, false},
		{"Data", reAB, false},
		{"Collection", arr, true},
		{"Collection", h, true},
		{"Collection[3, 5]", arr, false},
		{"Collection", int64(1), false},
		{"Array[Integer]", arr, true},
		{"Array[Integer]", []Value{"x"}, false},
		{"Array[Integer, 3, 5]", arr, false},
		{"Array[Integer]", int64(1), false},
		{"Tuple[Integer, String]", []Value{int64(1), "x"}, true},
		{"Tuple[Integer, String]", []Value{int64(1), int64(2)}, false},
		{"Tuple[Integer, String]", []Value{int64(1)}, false},
		{"Tuple[Integer, 1, 3]", []Value{int64(1), int64(2)}, true},
		{"Tuple[Integer, String]", int64(1), false},
		{"Hash[String, Integer]", h, true},
		{"Hash[String, Integer]", hInt, false},
		{"Hash[String, Integer, 3, 5]", h, false},
		{"Hash[String, Integer]", int64(1), false},
		{"Struct[{'a' => Integer, 'b' => Integer}]", h, true},
		{"Struct[{'a' => Integer}]", h, false}, // extra key b
		{"Struct[{'a' => Integer, 'b' => Integer, Optional['c'] => Integer}]", h, true},
		{"Struct[{'a' => Integer, 'b' => Integer, 'c' => Integer}]", h, false}, // missing c
		{"Struct[{'a' => Integer, 'b' => String}]", h, false},                  // b wrong type
		{"Struct[{'a' => Integer}]", hInt, false},                              // non-string key
		{"Struct[{'a' => Integer}]", int64(1), false},
		{"Variant[Integer, String]", "x", true},
		{"Variant[Integer, String]", true, false},
		{"Optional[Integer]", Undef, true},
		{"Optional[Integer]", int64(1), true},
		{"Optional[Integer]", "x", false},
		{"NotUndef[Integer]", int64(1), true},
		{"NotUndef[Integer]", Undef, false},
		{"NotUndef", int64(1), true},
		{"Type[Integer]", AnyInteger(), true},
		{"Type[Integer]", int64(1), false},
		{"Type", NewInteger(1, 3), true},
		{"Sensitive[String]", sens, true},
		{"Sensitive[Integer]", sens, false},
		{"Sensitive", int64(1), false},
		{"Binary", bin, true},
		{"Binary", int64(1), false},
		{"Timestamp", ts, true},
		{"Timestamp", int64(1), false},
		{"Timespan", tsp, true},
		{"Timespan", int64(1), false},
	}
	for _, c := range cases {
		ty := mustParse(t, c.typ)
		if got := IsInstance(ty, c.val); got != c.want {
			t.Errorf("IsInstance(%s, %v) = %v, want %v", c.typ, c.val, got, c.want)
		}
	}
}

func TestStructOptionalByUndefValue(t *testing.T) {
	// A member whose type accepts Undef is optional to provide.
	ty := mustParse(t, "Struct[{'a' => Integer, 'b' => Optional[String]}]")
	if !IsInstance(ty, NewHash(HashEntry{"a", int64(1)})) {
		t.Error("member with Undef-accepting type should be optional")
	}
}

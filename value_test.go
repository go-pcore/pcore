package pcore

import (
	"testing"
	"time"
)

func TestCanon(t *testing.T) {
	cases := []struct {
		in   Value
		want Value
	}{
		{nil, Undef},
		{int(1), int64(1)},
		{int8(1), int64(1)},
		{int16(1), int64(1)},
		{int32(1), int64(1)},
		{int64(1), int64(1)},
		{float32(1.5), float64(1.5)},
		{float64(1.5), float64(1.5)},
	}
	for _, c := range cases {
		if got := canon(c.in); got != c.want {
			t.Errorf("canon(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	// map[string]Value canonicalizes to a sorted *Hash.
	h, ok := canon(map[string]Value{"b": 2, "a": 1}).(*Hash)
	if !ok || h.Len() != 2 {
		t.Fatalf("map canon: %v", h)
	}
	if h.entries[0].Key != "a" || h.entries[1].Key != "b" {
		t.Errorf("map canon not sorted: %v", h)
	}
	if h.entries[0].Value != int64(1) {
		t.Errorf("map canon value not canonicalized: %v", h.entries[0].Value)
	}
	// nested []Value canonicalizes elements.
	a := canon([]Value{1, 2}).([]Value)
	if a[0] != int64(1) {
		t.Errorf("slice canon: %v", a)
	}
}

func TestHashGet(t *testing.T) {
	h := NewHash(HashEntry{"a", int64(1)})
	if v, ok := h.Get("a"); !ok || v != int64(1) {
		t.Errorf("Get present: %v %v", v, ok)
	}
	if _, ok := h.Get("missing"); ok {
		t.Error("Get missing should be absent")
	}
	if h.Entries()[0].Key != "a" {
		t.Error("Entries")
	}
}

func TestEqualValue(t *testing.T) {
	reAB, _ := NewRegexp("ab")
	reCD, _ := NewRegexp("cd")
	ts1 := NewTimestamp(time.Unix(0, 0))
	ts2 := NewTimestamp(time.Unix(1, 0))
	equal := [][2]Value{
		{Undef, nil},
		{Default, Default},
		{true, true},
		{int64(1), 1},
		{1.5, 1.5},
		{"x", "x"},
		{[]Value{int64(1)}, []Value{1}},
		{NewHash(HashEntry{"a", int64(1)}), NewHash(HashEntry{"a", int64(1)})},
		{reAB, reAB},
		{NewBinary([]byte{1, 2}), NewBinary([]byte{1, 2})},
		{ts1, ts1},
		{NewTimespan(time.Second), NewTimespan(time.Second)},
		{NewSensitive("s"), NewSensitive("s")},
		{NewInteger(1, 3), NewInteger(1, 3)},
	}
	for _, p := range equal {
		if !equalValue(p[0], p[1]) {
			t.Errorf("equalValue(%v, %v) = false, want true", p[0], p[1])
		}
	}
	unequal := [][2]Value{
		{Undef, int64(1)},
		{Default, int64(1)},
		{true, false},
		{true, int64(1)},
		{int64(1), int64(2)},
		{int64(1), "x"},
		{1.5, 2.5},
		{1.5, "x"},
		{"x", "y"},
		{"x", int64(1)},
		{[]Value{int64(1)}, []Value{int64(2)}},
		{[]Value{int64(1)}, []Value{int64(1), int64(2)}},
		{[]Value{int64(1)}, int64(1)},
		{NewHash(HashEntry{"a", int64(1)}), NewHash(HashEntry{"a", int64(2)})},
		{NewHash(HashEntry{"a", int64(1)}), NewHash(HashEntry{"b", int64(1)})},
		{NewHash(HashEntry{"a", int64(1)}), NewHash()},
		{NewHash(HashEntry{"a", int64(1)}), int64(1)},
		{reAB, reCD},
		{reAB, int64(1)},
		{NewBinary([]byte{1}), NewBinary([]byte{2})},
		{NewBinary([]byte{1}), NewBinary([]byte{1, 2})},
		{NewBinary([]byte{1}), int64(1)},
		{ts1, ts2},
		{ts1, int64(1)},
		{NewTimespan(time.Second), NewTimespan(time.Minute)},
		{NewTimespan(time.Second), int64(1)},
		{NewSensitive("s"), NewSensitive("t")},
		{NewSensitive("s"), int64(1)},
		{NewInteger(1, 3), NewInteger(1, 4)},
		{NewInteger(1, 3), int64(1)},
		{make(chan int), int64(1)},
	}
	for _, p := range unequal {
		if equalValue(p[0], p[1]) {
			t.Errorf("equalValue(%v, %v) = true, want false", p[0], p[1])
		}
	}
}

func TestValueStrings(t *testing.T) {
	if Undef.(undefValue).String() != "undef" {
		t.Error("Undef.String")
	}
	if Default.(defaultValue).String() != "default" {
		t.Error("Default.String")
	}
	re, _ := NewRegexp("ab")
	if re.String() != "/ab/" || re.Source() != "ab" || !re.MatchString("xaby") {
		t.Error("Regexp methods")
	}
	b := NewBinary([]byte{1, 2, 3})
	if b.String() != "Binary(3 bytes)" || len(b.Bytes()) != 3 {
		t.Error("Binary methods")
	}
	ts := NewTimestamp(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if ts.String() != "2026-01-02T03:04:05Z" || ts.Time().Year() != 2026 {
		t.Errorf("Timestamp: %q", ts.String())
	}
	tsp := NewTimespan(90 * time.Second)
	if tsp.String() != "1m30s" || tsp.Duration() != 90*time.Second {
		t.Error("Timespan methods")
	}
	s := NewSensitive("secret")
	if s.String() != "Sensitive[value redacted]" || s.Unwrap() != "secret" {
		t.Error("Sensitive methods")
	}
	h := NewHash(HashEntry{"a", int64(1)}, HashEntry{"b", int64(2)})
	if h.String() != "{a => 1, b => 2}" {
		t.Errorf("Hash.String = %q", h.String())
	}
}

func TestBadRegexp(t *testing.T) {
	if _, err := NewRegexp("("); err == nil {
		t.Error("NewRegexp( should error")
	}
}

func TestNameAndConstructors(t *testing.T) {
	if AnyInteger().Name() != "Integer" || AnyFloat().Name() != "Float" ||
		AnyString().Name() != "String" || NewArray(AnyT(), 0, 1).Name() != "Array" {
		t.Error("Name()")
	}
	// exercise the remaining exported constructors / singletons
	_ = NewHashType(AnyString(), AnyInteger(), 0, 1)
	_ = NewVariant(AnyInteger(), AnyString())
	_ = NewOptional(AnyInteger())
	for _, ty := range []Type{ScalarT(), DataT(), BooleanT(), UndefT(), DefaultT(),
		NumericT(), BinaryT(), TimestampT(), TimespanT()} {
		if ty.Name() == "" {
			t.Errorf("empty Name for %T", ty)
		}
	}
}

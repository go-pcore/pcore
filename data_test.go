package pcore

import (
	"testing"
	"time"
)

func TestDataRoundTrip(t *testing.T) {
	reAB, _ := NewRegexp("ab")
	values := []Value{
		Undef,
		true,
		int64(42),
		3.14,
		"hello",
		[]Value{int64(1), "two", true},
		NewHash(HashEntry{"a", int64(1)}, HashEntry{"b", []Value{int64(2)}}),
		NewHash(HashEntry{int64(1), "x"}, HashEntry{int64(2), "y"}), // non-string keys
		Default,
		reAB,
		NewBinary([]byte("binary\x00data")),
		NewTimestamp(time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)),
		NewTimespan(90 * time.Minute),
		NewInteger(1, 3),
	}
	for _, v := range values {
		data, err := ToData(v)
		if err != nil {
			t.Fatalf("ToData(%v): %v", v, err)
		}
		back, err := FromData(data)
		if err != nil {
			t.Fatalf("FromData for %v: %v", v, err)
		}
		if !equalValue(v, back) {
			t.Errorf("round-trip mismatch: %v -> %v", v, back)
		}
	}
}

func TestDataSensitiveRedacted(t *testing.T) {
	data, err := ToData(NewSensitive("secret"))
	if err != nil {
		t.Fatal(err)
	}
	h, ok := data.(*Hash)
	if !ok {
		t.Fatalf("want tagged hash, got %T", data)
	}
	if pt, _ := h.Get("__ptype"); pt != "Sensitive" {
		t.Errorf("ptype = %v", pt)
	}
	if _, present := h.Get("__pvalue"); present {
		t.Error("Sensitive payload must be redacted (no __pvalue)")
	}
	back, err := FromData(data)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := back.(*Sensitive)
	if !ok || s.Unwrap() != Undef {
		t.Errorf("redacted Sensitive should reconstruct wrapping Undef, got %v", back)
	}
}

func TestDataReservedKeyHash(t *testing.T) {
	// A plain hash that happens to contain a reserved key must be encoded via
	// the tagged array form so it survives the round-trip unambiguously.
	orig := NewHash(HashEntry{"__ptype", "NotAType"}, HashEntry{"x", int64(1)})
	data, err := ToData(orig)
	if err != nil {
		t.Fatal(err)
	}
	h := data.(*Hash)
	if pt, _ := h.Get("__ptype"); pt != "Hash" {
		t.Fatalf("reserved-key hash should be tagged Hash, got %v", pt)
	}
	back, err := FromData(data)
	if err != nil {
		t.Fatal(err)
	}
	if !equalValue(orig, back) {
		t.Errorf("reserved-key hash round-trip failed: %v", back)
	}
}

func TestToDataErrors(t *testing.T) {
	if _, err := ToData(make(chan int)); err == nil {
		t.Error("ToData(chan) should error")
	}
	if _, err := ToData([]Value{make(chan int)}); err == nil {
		t.Error("ToData(array with bad element) should error")
	}
	if _, err := ToData(NewHash(HashEntry{"a", make(chan int)})); err == nil {
		t.Error("ToData(hash with bad value) should error")
	}
	if _, err := ToData(NewHash(HashEntry{make(chan int), "a"})); err == nil {
		t.Error("ToData(hash with bad key) should error")
	}
}

func TestFromDataErrors(t *testing.T) {
	bad := []Value{
		make(chan int),                                                           // unsupported
		[]Value{make(chan int)},                                                  // bad element
		NewHash(HashEntry{"k", make(chan int)}),                                  // bad value
		NewHash(HashEntry{"__ptype", int64(1)}),                                  // ptype not string
		NewHash(HashEntry{"__ptype", "Bogus"}),                                   // unknown ptype
		NewHash(HashEntry{"__ptype", "Regexp"}),                                  // missing payload
		NewHash(HashEntry{"__ptype", "Regexp"}, HashEntry{"__pvalue", int64(1)}), // payload not string
		NewHash(HashEntry{"__ptype", "Regexp"}, HashEntry{"__pvalue", "("}),      // invalid regexp
		NewHash(HashEntry{"__ptype", "Binary"}, HashEntry{"__pvalue", "!!!"}),    // invalid base64
		NewHash(HashEntry{"__ptype", "Timestamp"}, HashEntry{"__pvalue", "nope"}),
		NewHash(HashEntry{"__ptype", "Timespan"}, HashEntry{"__pvalue", "nope"}),
		NewHash(HashEntry{"__ptype", "Type"}, HashEntry{"__pvalue", "Bogus"}),
		NewHash(HashEntry{"__ptype", "Hash"}, HashEntry{"__pvalue", int64(1)}),                          // payload not array
		NewHash(HashEntry{"__ptype", "Hash"}, HashEntry{"__pvalue", []Value{int64(1)}}),                 // odd length
		NewHash(HashEntry{"__ptype", "Hash"}, HashEntry{"__pvalue", []Value{make(chan int), int64(1)}}), // bad key
		NewHash(HashEntry{"__ptype", "Hash"}, HashEntry{"__pvalue", []Value{int64(1), make(chan int)}}), // bad value
	}
	for i, v := range bad {
		if _, err := FromData(v); err == nil {
			t.Errorf("FromData case %d should error", i)
		}
	}
}

func TestFromDataDefaultTag(t *testing.T) {
	d, err := FromData(NewHash(HashEntry{"__ptype", "Default"}))
	if err != nil || d != Default {
		t.Errorf("Default tag: %v %v", d, err)
	}
}

// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "testing"

func TestObjectParseAndInstance(t *testing.T) {
	ty := mustParse(t, "Object[{name => 'Point', attributes => {'x' => Integer, 'y' => Integer}}]")
	if ty.Name() != "Object" {
		t.Error("Object name")
	}
	ov, err := NewObjectValue(ty, NewHash(HashEntry{"x", int64(1)}, HashEntry{"y", int64(2)}))
	if err != nil {
		t.Fatal(err)
	}
	if !IsInstance(ty, ov) {
		t.Error("point value is a Point")
	}
	if v, _ := ov.Get("x"); v != int64(1) {
		t.Errorf("Get x = %v", v)
	}
	if ov.String() == "" {
		t.Error("object value string")
	}
	// A non-object value.
	if IsInstance(ty, int64(1)) {
		t.Error("int is not a Point")
	}
	// The base Object matches any object value.
	if !IsInstance(mustParse(t, "Object"), ov) {
		t.Error("base Object matches any object")
	}
	if IsInstance(mustParse(t, "Object"), int64(1)) {
		t.Error("base Object does not match a scalar")
	}
}

func TestObjectConstructorValidation(t *testing.T) {
	ty := mustParse(t, "Object[{name => 'P', attributes => {'x' => Integer}}]")
	if _, err := NewObjectValue(ty, NewHash()); err == nil {
		t.Error("missing required attribute should error")
	}
	if _, err := NewObjectValue(ty, NewHash(HashEntry{"x", "notint"})); err == nil {
		t.Error("wrong attribute type should error")
	}
	if _, err := NewObjectValue(mustParse(t, "Integer"), nil); err == nil {
		t.Error("NewObjectValue on a non-Object type should error")
	}
	// nil attrs with a default-only object.
	def := mustParse(t, "Object[{name => 'D', attributes => {'n' => {'type' => Integer, 'value' => 7}}}]")
	ov, err := NewObjectValue(def, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := ov.Get("n"); v != int64(7) {
		t.Errorf("default value = %v", v)
	}
}

func TestObjectInheritance(t *testing.T) {
	l := NewLoader()
	if err := l.Declare("type Animal = Object[{name => 'Animal', attributes => {'legs' => Integer}}]"); err != nil {
		t.Fatal(err)
	}
	if err := l.Declare("type Dog = Object[{name => 'Dog', parent => Animal, attributes => {'good' => Boolean}}]"); err != nil {
		t.Fatal(err)
	}
	animal, err := l.Parse("Animal")
	if err != nil {
		t.Fatal(err)
	}
	dog, err := l.Parse("Dog")
	if err != nil {
		t.Fatal(err)
	}
	// A Dog value carries inherited + own attributes.
	dv, err := NewObjectValue(dog, NewHash(HashEntry{"legs", int64(4)}, HashEntry{"good", true}))
	if err != nil {
		t.Fatal(err)
	}
	if !IsInstance(dog, dv) {
		t.Error("dog value is a Dog")
	}
	if !IsInstance(animal, dv) {
		t.Error("a Dog is an Animal (nominal subtype)")
	}
	// Assignability: Dog <: Animal, not the reverse.
	if !IsAssignable(animal, dog) {
		t.Error("Dog <: Animal")
	}
	if IsAssignable(dog, animal) {
		t.Error("Animal not <: Dog")
	}
	if IsAssignable(dog, mustParse(t, "Integer")) {
		t.Error("Integer not <: Dog")
	}
	// String round-trips through the loader.
	if _, err := l.Parse(asObjectType(dog).String()); err != nil {
		t.Errorf("object string round-trip: %v", err)
	}
}

func TestObjectParseErrors(t *testing.T) {
	bad := []string{
		"Object[Integer]",                                              // not a hash
		"Object[{attributes => {'x' => Integer}}]",                     // missing name
		"Object[{name => 5}]",                                          // name not a string
		"Object[{name => 'X', parent => 'nope'}]",                      // parent not a type
		"Object[{name => 'X', attributes => Integer}]",                 // attributes not a hash
		"Object[{name => 'X', attributes => {'x' => 5}}]",              // attribute not a type/hash
		"Object[{name => 'X', attributes => {'x' => {'value' => 1}}}]", // attr hash missing type
		"Object[{name => 'X', attributes => {'x' => {'type' => 5}}}]",  // attr type not a type
		"Object[{name => 'X'",                                          // truncated
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

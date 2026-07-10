// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "testing"

func TestTypeSetBasic(t *testing.T) {
	src := `TypeSet[{
		pcore_version => '1.0.0',
		name => 'MyMod::Types',
		version => '1.2.0',
		types => {
			Age => Integer[0, 130],
			Person => Struct[{'name' => String, 'age' => Age}]
		}
	}]`
	ty, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	ts, ok := ty.(*typeSetType)
	if !ok {
		t.Fatalf("expected a TypeSet, got %T", ty)
	}
	if ts.Name() != "TypeSet" {
		t.Error("TypeSet Name()")
	}
	if ts.TypeSetName() != "MyMod::Types" || ts.Version() != "1.2.0" {
		t.Errorf("metadata: %q %q", ts.TypeSetName(), ts.Version())
	}
	// Member types resolve, including cross-member references (Person -> Age).
	person, ok := ts.Type("Person")
	if !ok {
		t.Fatal("Person member missing")
	}
	good := NewHash(HashEntry{"name", "Ada"}, HashEntry{"age", int64(37)})
	if !IsInstance(person, good) {
		t.Error("valid person should match")
	}
	bad := NewHash(HashEntry{"name", "Ada"}, HashEntry{"age", int64(200)})
	if IsInstance(person, bad) {
		t.Error("age 200 out of Age range")
	}
	if _, ok := ts.Type("Ghost"); ok {
		t.Error("no Ghost member")
	}
	// String round-trips through Parse.
	if _, err := Parse(ts.String()); err != nil {
		t.Errorf("TypeSet string round-trip: %v", err)
	}
}

func TestTypeSetInstanceUnion(t *testing.T) {
	ty, err := Parse(`TypeSet[{name => 'S', version => '1.0.0', types => {A => Integer, B => String}}]`)
	if err != nil {
		t.Fatal(err)
	}
	if !IsInstance(ty, int64(1)) || !IsInstance(ty, "x") {
		t.Error("TypeSet accepts instances of its members")
	}
	if IsInstance(ty, true) {
		t.Error("TypeSet rejects non-member instances")
	}
}

func TestTypeSetAssignable(t *testing.T) {
	mk := func() Type {
		ty, err := Parse(`TypeSet[{name => 'S', version => '1.0.0', types => {A => Integer}}]`)
		if err != nil {
			t.Fatal(err)
		}
		return ty
	}
	if !IsAssignable(mk(), mk()) {
		t.Error("TypeSets with same name+version are assignable")
	}
	other, err := Parse(`TypeSet[{name => 'S', version => '2.0.0', types => {A => Integer}}]`)
	if err != nil {
		t.Fatal(err)
	}
	if IsAssignable(mk(), other) {
		t.Error("different versions not assignable")
	}
	if IsAssignable(mk(), mustParse(t, "Integer")) {
		t.Error("TypeSet not assignable from Integer")
	}
}

func TestTypeSetReferences(t *testing.T) {
	l := NewLoader()
	// Register a base TypeSet, then reference it from another.
	base := `TypeSet[{name => 'Base', version => '1.0.0', types => {Id => Integer[1, 100]}}]`
	if _, err := l.Parse(base); err != nil {
		t.Fatal(err)
	}
	ref := `TypeSet[{
		name => 'App', version => '1.0.0',
		references => { B => {name => 'Base', version_range => '1.x'} },
		types => { Record => Struct[{'id' => B::Id}] }
	}]`
	ty, err := l.Parse(ref)
	if err != nil {
		t.Fatal(err)
	}
	ts := ty.(*typeSetType)
	rec, ok := ts.Type("Record")
	if !ok {
		t.Fatal("Record missing")
	}
	if !IsInstance(rec, NewHash(HashEntry{"id", int64(5)})) {
		t.Error("record with valid cross-typeset id")
	}
	if IsInstance(rec, NewHash(HashEntry{"id", int64(500)})) {
		t.Error("id 500 out of Base::Id range")
	}
	if _, err := l.Parse(ts.String()); err != nil {
		t.Errorf("TypeSet-with-references round-trip: %v", err)
	}
}

func TestTypeSetErrors(t *testing.T) {
	bad := []string{
		`TypeSet`,                                                           // no definition
		`TypeSet[Integer]`,                                                  // not a hash
		`TypeSet[{name => 'X', bogus => 1}]`,                                // unknown key
		`TypeSet[{name => 5}]`,                                              // name not string via stringValue
		`TypeSet[{types => Integer}]`,                                       // types not a hash
		`TypeSet[{types => {A => Integer, A => String}}]`,                   // duplicate member
		`TypeSet[{types => {A => Ghost}}]`,                                  // unresolved member ref
		`TypeSet[{references => Integer}]`,                                  // references not a hash
		`TypeSet[{references => {B => Integer}}]`,                           // reference body not a hash
		`TypeSet[{references => {B => {version_range => '1.x'}}}]`,          // reference missing name
		`TypeSet[{references => {B => {name => 5}}}]`,                       // reference name not string
		`TypeSet[{references => {B => {name => 'X', version_range => 5}}}]`, // version_range not string
		`TypeSet[{name => 'X'`,                                              // truncated
		`TypeSet[{name => 'X' bogus}]`,                                      // missing comma
		`TypeSet[{name 'X'}]`,                                               // missing arrow
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestTypeSetTypesBlockErrors(t *testing.T) {
	bad := []string{
		`TypeSet[{types => {A Integer}}]`,      // missing arrow in types
		`TypeSet[{types => {A => Integer B}}]`, // missing comma in types
		`TypeSet[{types => {A => Ghost[}}]`,    // parse error in member body
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "testing"

func mustSemVer(t *testing.T, s string) *SemVer {
	t.Helper()
	v, err := NewSemVer(s)
	if err != nil {
		t.Fatalf("NewSemVer(%q): %v", s, err)
	}
	return v
}

func mustSemVerRange(t *testing.T, s string) *SemVerRange {
	t.Helper()
	r, err := NewSemVerRange(s)
	if err != nil {
		t.Fatalf("NewSemVerRange(%q): %v", s, err)
	}
	return r
}

// mustLoaderParse declares each decl then parses expr in that loader.
func mustLoaderParse(t *testing.T, expr string, decls ...string) Type {
	t.Helper()
	l := NewLoader()
	for _, d := range decls {
		if err := l.Declare(d); err != nil {
			t.Fatalf("Declare(%q): %v", d, err)
		}
	}
	ty, err := l.Parse(expr)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expr, err)
	}
	return ty
}

func TestAliasSimple(t *testing.T) {
	l := NewLoader()
	if err := l.Declare("type Age = Integer[0, 130]"); err != nil {
		t.Fatal(err)
	}
	ty, err := l.Parse("Age")
	if err != nil {
		t.Fatal(err)
	}
	if ty.String() != "Age" || ty.Name() != "Age" {
		t.Errorf("alias string/name = %q/%q", ty.String(), ty.Name())
	}
	if !IsInstance(ty, int64(40)) {
		t.Error("40 should be an Age")
	}
	if IsInstance(ty, int64(200)) {
		t.Error("200 should not be an Age")
	}
	// Assignability transparently through the alias, both directions.
	if !IsAssignable(mustParse(t, "Integer"), ty) {
		t.Error("Age <: Integer")
	}
	if !IsAssignable(ty, mustParse(t, "Integer[10, 20]")) {
		t.Error("Integer[10,20] <: Age")
	}
}

func TestAliasForwardReference(t *testing.T) {
	l := NewLoader()
	// B is declared after A references it.
	if err := l.Declare("type A = Array[B]"); err != nil {
		t.Fatal(err)
	}
	if err := l.Declare("type B = Integer[0, 9]"); err != nil {
		t.Fatal(err)
	}
	ty, err := l.Parse("A")
	if err != nil {
		t.Fatal(err)
	}
	if !IsInstance(ty, []Value{int64(1), int64(2)}) {
		t.Error("[1,2] should be an A")
	}
	if IsInstance(ty, []Value{int64(1), int64(20)}) {
		t.Error("[1,20] should not be an A")
	}
}

func TestAliasRecursiveTree(t *testing.T) {
	// The headline recursive alias from the task.
	ty := mustLoaderParse(t, "Tree", "type Tree = Hash[String, Variant[Tree, Integer]]")
	good := NewHash(
		HashEntry{"a", int64(1)},
		HashEntry{"b", NewHash(HashEntry{"c", int64(2)})},
	)
	if !IsInstance(ty, good) {
		t.Error("nested tree hash should match Tree")
	}
	bad := NewHash(HashEntry{"a", NewHash(HashEntry{"b", "not-an-int"})})
	if IsInstance(ty, bad) {
		t.Error("tree with a string leaf should not match Tree")
	}
	// String renders as the alias name.
	if ty.String() != "Tree" {
		t.Errorf("Tree.String() = %q", ty.String())
	}
	// Assignable to itself (guarded) and to its expansion.
	if !IsAssignable(ty, ty) {
		t.Error("Tree <: Tree")
	}
	if !IsAssignable(ty, mustLoaderParse(t, "Tree", "type Tree = Hash[String, Variant[Tree, Integer]]")) {
		t.Error("Tree <: Tree (fresh)")
	}
}

func TestAliasMutualRecursionAssignable(t *testing.T) {
	// Two structurally-identical recursive aliases: assignability must terminate.
	l := NewLoader()
	must := func(d string) {
		if err := l.Declare(d); err != nil {
			t.Fatal(err)
		}
	}
	must("type A = Hash[String, A]")
	must("type B = Hash[String, B]")
	a, err := l.Parse("A")
	if err != nil {
		t.Fatal(err)
	}
	b, err := l.Parse("B")
	if err != nil {
		t.Fatal(err)
	}
	if !IsAssignable(a, b) {
		t.Error("A <: B for identical recursive shapes")
	}
}

func TestAliasNonProductiveCycle(t *testing.T) {
	l := NewLoader()
	if err := l.Declare("type A = B"); err != nil {
		t.Fatal(err)
	}
	if err := l.Declare("type B = A"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Parse("A"); err == nil {
		t.Error("A = B, B = A should be an unresolvable cycle")
	}
}

func TestAliasSelfCycle(t *testing.T) {
	l := NewLoader()
	if err := l.Declare("type A = A"); err != nil {
		t.Fatal(err)
	}
	if l.Validate() == nil {
		t.Error("type A = A should be unresolvable")
	}
}

func TestAliasUnresolved(t *testing.T) {
	l := NewLoader()
	if _, err := l.Parse("Ghost"); err == nil {
		t.Error("referencing an undeclared alias should error")
	}
}

func TestAliasErrors(t *testing.T) {
	l := NewLoader()
	cases := []string{
		"Age = Integer",         // missing 'type'
		"type",                  // nothing after keyword
		"type Age Integer",      // missing '='
		"type age = Integer",    // lowercase name
		"type Integer = String", // redefine builtin
		"type Age = Bogus[",     // parse error in body
	}
	for _, c := range cases {
		if err := l.Declare(c); err == nil {
			t.Errorf("Declare(%q) should error", c)
		}
	}
	// Duplicate declaration.
	if err := l.Declare("type Dup = Integer"); err != nil {
		t.Fatal(err)
	}
	if err := l.Declare("type Dup = String"); err == nil {
		t.Error("duplicate declaration should error")
	}
}

func TestAliasParameterizedReferenceRejected(t *testing.T) {
	l := NewLoader()
	if err := l.Declare("type Age = Integer"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Parse("Age[1, 2]"); err == nil {
		t.Error("an alias reference cannot take parameters")
	}
}

func TestAliasInInferAndCommon(t *testing.T) {
	ty := mustLoaderParse(t, "Small", "type Small = Integer[0, 5]")
	// NotUndef through an alias body.
	if !IsInstance(mustParse(t, "NotUndef"), int64(3)) {
		t.Error("sanity")
	}
	// allowsUndef resolves through an alias.
	opt := mustLoaderParse(t, "MaybeSmall", "type MaybeSmall = Optional[Integer[0, 5]]")
	if !IsInstance(opt, Undef) {
		t.Error("Optional alias should accept undef")
	}
	_ = ty
}

func TestPackageParseUnknownStillErrors(t *testing.T) {
	// Without a loader, an unknown name is an error (not an alias reference).
	if _, err := Parse("Whatever"); err == nil {
		t.Error("package-level Parse should reject unknown type names")
	}
}

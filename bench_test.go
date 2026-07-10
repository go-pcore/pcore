// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "testing"

// Representative expressions spanning scalar, collection, abstract and rich
// constructs, plus a recursive alias resolved once for the instance/assign
// benchmarks.

var benchExprs = []string{
	"Integer[0, 10]",
	"String[1, 20]",
	"Enum['red', 'green', 'blue']",
	"Array[Integer[0, 100], 1, 10]",
	"Hash[String, Variant[Integer, String]]",
	"Struct[{'name' => String, 'age' => Integer[0, 130]}]",
	"Variant[Integer, Enum['a', 'b'], Array[Float]]",
	"Optional[Pattern[/^[a-z]+$/]]",
	"SemVer['>=1.0.0 <2.0.0']",
	"Timestamp['2020-01-01T00:00:00Z', '2030-01-01T00:00:00Z']",
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, e := range benchExprs {
			if _, err := Parse(e); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkParseAlias(b *testing.B) {
	decl := "type Tree = Hash[String, Variant[Tree, Integer]]"
	for i := 0; i < b.N; i++ {
		l := NewLoader()
		if err := l.Declare(decl); err != nil {
			b.Fatal(err)
		}
		if _, err := l.Parse("Tree"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIsInstance(b *testing.B) {
	ty := mustParseB(b, "Struct[{'name' => String, 'age' => Integer[0, 130]}]")
	v := NewHash(HashEntry{"name", "Ada"}, HashEntry{"age", int64(37)})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !IsInstance(ty, v) {
			b.Fatal("expected instance")
		}
	}
}

func BenchmarkIsInstanceRecursiveAlias(b *testing.B) {
	l := NewLoader()
	if err := l.Declare("type Tree = Hash[String, Variant[Tree, Integer]]"); err != nil {
		b.Fatal(err)
	}
	ty, err := l.Parse("Tree")
	if err != nil {
		b.Fatal(err)
	}
	v := NewHash(
		HashEntry{"a", int64(1)},
		HashEntry{"b", NewHash(HashEntry{"c", int64(2)}, HashEntry{"d", int64(3)})},
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !IsInstance(ty, v) {
			b.Fatal("expected instance")
		}
	}
}

func BenchmarkIsAssignable(b *testing.B) {
	a := mustParseB(b, "Hash[String, Variant[Integer, String]]")
	c := mustParseB(b, "Struct[{'x' => Integer[1, 2], 'y' => String[1, 3]}]")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !IsAssignable(a, c) {
			b.Fatal("expected assignable")
		}
	}
}

func BenchmarkInfer(b *testing.B) {
	v := []Value{int64(1), int64(2), int64(3), "x", 1.5}
	for i := 0; i < b.N; i++ {
		_ = Infer(v)
	}
}

func mustParseB(b *testing.B, s string) Type {
	b.Helper()
	t, err := Parse(s)
	if err != nil {
		b.Fatalf("Parse(%q): %v", s, err)
	}
	return t
}

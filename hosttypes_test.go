// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "testing"

func TestRuntimeType(t *testing.T) {
	if mustParse(t, "Runtime").String() != "Runtime" {
		t.Error("generic Runtime string")
	}
	rv := NewRuntimeValue("go", "bytes.Buffer", nil)
	if rv.Unwrap() != nil || rv.String() == "" {
		t.Error("runtime value accessors")
	}
	if !IsInstance(mustParse(t, "Runtime"), rv) {
		t.Error("any runtime value is Runtime")
	}
	if IsInstance(mustParse(t, "Runtime"), int64(1)) {
		t.Error("int is not Runtime")
	}
	goRun := mustParse(t, "Runtime['go']")
	if goRun.String() != "Runtime['go']" || goRun.Name() != "Runtime" {
		t.Error("Runtime['go'] string/name")
	}
	if !IsInstance(goRun, rv) {
		t.Error("go runtime matches Runtime['go']")
	}
	if IsInstance(goRun, NewRuntimeValue("ruby", "String", nil)) {
		t.Error("ruby value does not match Runtime['go']")
	}
	named := mustParse(t, "Runtime['go', 'bytes.Buffer']")
	if !IsInstance(named, rv) {
		t.Error("named runtime matches")
	}
	if IsInstance(named, NewRuntimeValue("go", "other", nil)) {
		t.Error("different name does not match")
	}
	pat := mustParse(t, "Runtime['go', /Buffer$/]")
	if pat.String() != "Runtime['go', /Buffer$/]" {
		t.Errorf("pattern runtime string = %q", pat.String())
	}
	if !IsInstance(pat, rv) {
		t.Error("pattern matches bytes.Buffer")
	}
	if IsInstance(pat, NewRuntimeValue("go", "Reader", nil)) {
		t.Error("pattern does not match Reader")
	}
}

func TestRuntimeAssignable(t *testing.T) {
	if !IsAssignable(mustParse(t, "Runtime"), mustParse(t, "Runtime['go']")) {
		t.Error("Runtime accepts Runtime['go']")
	}
	if !IsAssignable(mustParse(t, "Runtime['go']"), mustParse(t, "Runtime['go', 'x']")) {
		t.Error("Runtime['go'] accepts a more specific go runtime")
	}
	if IsAssignable(mustParse(t, "Runtime['go']"), mustParse(t, "Runtime['ruby']")) {
		t.Error("different runtimes not assignable")
	}
	if !IsAssignable(mustParse(t, "Runtime['go', 'x']"), mustParse(t, "Runtime['go', 'x']")) {
		t.Error("equal named runtimes assignable")
	}
	if IsAssignable(mustParse(t, "Runtime['go', 'x']"), mustParse(t, "Runtime['go']")) {
		t.Error("named not assignable from unnamed")
	}
	if !IsAssignable(mustParse(t, "Runtime['go', /B/]"), mustParse(t, "Runtime['go', 'aBc']")) {
		t.Error("pattern runtime accepts matching name")
	}
	if IsAssignable(mustParse(t, "Runtime['go', /B/]"), mustParse(t, "Runtime['go']")) {
		t.Error("pattern runtime rejects unnamed")
	}
	if IsAssignable(mustParse(t, "Runtime"), mustParse(t, "Integer")) {
		t.Error("Runtime not assignable from Integer")
	}
}

func TestRuntimeBuildErrors(t *testing.T) {
	for _, s := range []string{"Runtime[1]", "Runtime[1, 'x']", "Runtime['go', 5]", "Runtime['a', 'b', 'c']"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestURIType(t *testing.T) {
	if mustParse(t, "URI").String() != "URI" || mustParse(t, "URI").Name() != "URI" {
		t.Error("URI string/name")
	}
	u := NewURI("https://example.com")
	if u.Value() != "https://example.com" || u.String() != "https://example.com" {
		t.Error("URI value accessors")
	}
	if !IsInstance(mustParse(t, "URI"), u) {
		t.Error("any URI value is URI")
	}
	if IsInstance(mustParse(t, "URI"), int64(1)) {
		t.Error("int is not URI")
	}
	https := mustParse(t, "URI['https']")
	if https.String() != "URI['https']" {
		t.Errorf("URI['https'] string = %q", https.String())
	}
	if !IsInstance(https, u) {
		t.Error("https URI matches URI['https']")
	}
	if IsInstance(https, NewURI("ftp://x")) {
		t.Error("ftp URI does not match URI['https']")
	}
	if !IsAssignable(mustParse(t, "URI"), https) {
		t.Error("URI accepts URI['https']")
	}
	if !IsAssignable(https, https) {
		t.Error("URI['https'] <: itself")
	}
	if IsAssignable(https, mustParse(t, "URI['ftp']")) {
		t.Error("URI['https'] not from URI['ftp']")
	}
	if IsAssignable(mustParse(t, "URI"), mustParse(t, "Integer")) {
		t.Error("URI not from Integer")
	}
}

func TestURIBuildError(t *testing.T) {
	for _, s := range []string{"URI[5]", "URI['a', 'b']"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestIterableIterator(t *testing.T) {
	if mustParse(t, "Iterable").String() != "Iterable" {
		t.Error("generic Iterable string")
	}
	if mustParse(t, "Iterator").String() != "Iterator" {
		t.Error("generic Iterator string")
	}
	iti := mustParse(t, "Iterable[Integer]")
	if iti.String() != "Iterable[Integer]" || iti.Name() != "Iterable" {
		t.Error("Iterable[Integer] string/name")
	}
	if !IsInstance(iti, []Value{int64(1), int64(2)}) {
		t.Error("[1,2] is Iterable[Integer]")
	}
	if IsInstance(iti, []Value{int64(1), "x"}) {
		t.Error("[1,x] is not Iterable[Integer]")
	}
	if IsInstance(iti, int64(1)) {
		t.Error("int is not Iterable")
	}
	// Strings are iterable as characters.
	if !IsInstance(mustParse(t, "Iterable[String]"), "abc") {
		t.Error("string is Iterable[String]")
	}
	if !IsInstance(mustParse(t, "Iterable"), "abc") {
		t.Error("string is Iterable (Any)")
	}
	if IsInstance(mustParse(t, "Iterable[Integer]"), "abc") {
		t.Error("string is not Iterable[Integer]")
	}
	// Iterator values.
	it := NewIterator(mustParse(t, "Integer"), int64(1), int64(2))
	if it.String() != "Iterator[Integer]" || len(it.Items()) != 2 {
		t.Error("iterator value")
	}
	if !IsInstance(mustParse(t, "Iterator[Integer]"), it) {
		t.Error("integer iterator is Iterator[Integer]")
	}
	if IsInstance(mustParse(t, "Iterator[String]"), it) {
		t.Error("integer iterator is not Iterator[String]")
	}
	if IsInstance(mustParse(t, "Iterator[Integer]"), int64(1)) {
		t.Error("int is not an Iterator")
	}
	// An Iterator is Iterable.
	if !IsInstance(mustParse(t, "Iterable[Integer]"), it) {
		t.Error("Iterator[Integer] value is Iterable[Integer]")
	}
}

func TestIterableIteratorAssignable(t *testing.T) {
	if !IsAssignable(mustParse(t, "Iterable[Integer]"), mustParse(t, "Iterable[Integer[0, 5]]")) {
		t.Error("Iterable covariance")
	}
	if !IsAssignable(mustParse(t, "Iterable[Integer]"), mustParse(t, "Iterator[Integer]")) {
		t.Error("Iterator <: Iterable")
	}
	if !IsAssignable(mustParse(t, "Iterable[Integer]"), mustParse(t, "Array[Integer]")) {
		t.Error("Array <: Iterable")
	}
	if IsAssignable(mustParse(t, "Iterable[Integer]"), mustParse(t, "String")) {
		t.Error("String type not <: Iterable[Integer]")
	}
	if !IsAssignable(mustParse(t, "Iterator[Integer]"), mustParse(t, "Iterator[Integer[0, 5]]")) {
		t.Error("Iterator covariance")
	}
	if IsAssignable(mustParse(t, "Iterator[Integer]"), mustParse(t, "Array[Integer]")) {
		t.Error("Array not <: Iterator")
	}
}

func TestIterBuildErrors(t *testing.T) {
	for _, s := range []string{"Iterable[1]", "Iterator[1]", "Iterable[Integer, String]"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestErrorType(t *testing.T) {
	if mustParse(t, "Error").String() != "Error" || mustParse(t, "Error").Name() != "Error" {
		t.Error("Error string/name")
	}
	ev := NewError("boom", "puppet.error", "PUP123")
	if ev.Message() != "boom" || ev.Kind() != "puppet.error" || ev.IssueCode() != "PUP123" || ev.String() == "" {
		t.Error("error value accessors")
	}
	if !IsInstance(mustParse(t, "Error"), ev) {
		t.Error("any error value is Error")
	}
	if IsInstance(mustParse(t, "Error"), int64(1)) {
		t.Error("int is not Error")
	}
	kinded := mustParse(t, "Error[Enum['puppet.error']]")
	if kinded.String() != "Error[Enum['puppet.error']]" {
		t.Errorf("kinded error string = %q", kinded.String())
	}
	if !IsInstance(kinded, ev) {
		t.Error("matching kind")
	}
	if IsInstance(kinded, NewError("x", "other.kind", "")) {
		t.Error("non-matching kind")
	}
	both := mustParse(t, "Error[String, String]")
	if both.String() != "Error[String, String]" {
		t.Errorf("two-arg error string = %q", both.String())
	}
	if IsInstance(both, NewError("x", "k", "")) {
		t.Error("empty issue code fails String issue constraint")
	}
	if !IsInstance(both, NewError("x", "k", "code")) {
		t.Error("full error matches Error[String,String]")
	}
}

func TestErrorAssignable(t *testing.T) {
	if !IsAssignable(mustParse(t, "Error"), mustParse(t, "Error[String]")) {
		t.Error("Error accepts Error[String]")
	}
	if IsAssignable(mustParse(t, "Error[String]"), mustParse(t, "Error")) {
		t.Error("Error[String] not from Error (Any kind)")
	}
	if IsAssignable(mustParse(t, "Error"), mustParse(t, "Integer")) {
		t.Error("Error not from Integer")
	}
}

func TestErrorBuildErrors(t *testing.T) {
	for _, s := range []string{"Error[1]", "Error[String, 1]", "Error[String, String, String]"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestCallableType(t *testing.T) {
	if mustParse(t, "Callable").String() != "Callable" || mustParse(t, "Callable").Name() != "Callable" {
		t.Error("generic Callable string/name")
	}
	ct := mustParse(t, "Callable[Integer, String]")
	if ct.String() != "Callable[Integer, String]" {
		t.Errorf("callable string = %q", ct.String())
	}
	c, err := NewCallable(ct)
	if err != nil {
		t.Fatal(err)
	}
	if c.String() != "Callable[Integer, String]" {
		t.Error("callable value string")
	}
	if !IsInstance(ct, c) {
		t.Error("callable value is instance of its own signature")
	}
	if IsInstance(ct, int64(1)) {
		t.Error("int is not Callable")
	}
	if _, err := NewCallable(mustParse(t, "Integer")); err == nil {
		t.Error("NewCallable on non-callable should error")
	}
	// Generic Callable matches any callable value.
	if !IsInstance(mustParse(t, "Callable"), c) {
		t.Error("generic Callable matches any callable")
	}
	// Arity form.
	sz := mustParse(t, "Callable[String, 0, 3]")
	if sz.String() != "Callable[String, 0, 3]" {
		t.Errorf("callable arity string = %q", sz.String())
	}
	// Single min with open max.
	one := mustParse(t, "Callable[String, 1]")
	if one.String() != "Callable[String, 1, default]" {
		t.Errorf("callable single-arity string = %q", one.String())
	}
	// Block parameter.
	blk := mustParse(t, "Callable[Integer, Callable[String]]")
	if blk.(*callableType).block == nil {
		t.Error("trailing Callable should be treated as a block")
	}
	// Block after size.
	blk2 := mustParse(t, "Callable[Integer, 1, 1, Callable[String]]")
	if blk2.(*callableType).block == nil {
		t.Error("block after arity")
	}
	// Optional[Callable] block.
	blk3 := mustParse(t, "Callable[Integer, Optional[Callable[String]]]")
	if blk3.(*callableType).block == nil {
		t.Error("Optional[Callable] block")
	}
}

func TestCallableAssignable(t *testing.T) {
	// Contravariant parameters: a supertype with a wider param accepts a subtype
	// with a narrower one? No — Callable[Integer] (super) needs the sub to accept
	// at least Integers.
	if !IsAssignable(mustParse(t, "Callable[Integer]"), mustParse(t, "Callable[Numeric]")) {
		t.Error("Callable[Numeric] can be called with an Integer, so <: Callable[Integer]")
	}
	if IsAssignable(mustParse(t, "Callable[Numeric]"), mustParse(t, "Callable[Integer]")) {
		t.Error("Callable[Integer] cannot serve where Callable[Numeric] is needed")
	}
	if !IsAssignable(mustParse(t, "Callable"), mustParse(t, "Callable[Integer]")) {
		t.Error("generic Callable accepts any callable")
	}
	if IsAssignable(mustParse(t, "Callable[Integer]"), mustParse(t, "Callable")) {
		t.Error("specific Callable not from generic Callable")
	}
	if IsAssignable(mustParse(t, "Callable[Integer]"), mustParse(t, "Integer")) {
		t.Error("Callable not from Integer")
	}
	// Arity mismatch.
	if IsAssignable(mustParse(t, "Callable[String, 2, 2]"), mustParse(t, "Callable[String, 0, 1]")) {
		t.Error("arity not covered")
	}
}

func TestCallableBuildError(t *testing.T) {
	// An integer before the parameter types is malformed.
	if _, err := Parse("Callable[1, String]"); err == nil {
		t.Error("Callable[1, String] should error")
	}
}

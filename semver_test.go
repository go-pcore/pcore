// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "testing"

func TestSemVerParseAndString(t *testing.T) {
	cases := map[string]string{
		"1.2.3":            "1.2.3",
		"1.0.0-rc.1":       "1.0.0-rc.1",
		"2.3.4+build.5":    "2.3.4+build.5",
		"1.0.0-alpha+meta": "1.0.0-alpha+meta",
	}
	for in, want := range cases {
		v := mustSemVer(t, in)
		if v.String() != want {
			t.Errorf("SemVer(%q).String() = %q", in, v.String())
		}
	}
}

func TestSemVerParseErrors(t *testing.T) {
	bad := []string{"1.2", "1.2.3.4", "x.2.3", "1.2.3-", "1.2.3+", "1.2.3-a..b", "1.2.-3"}
	for _, s := range bad {
		if _, err := NewSemVer(s); err == nil {
			t.Errorf("NewSemVer(%q) should error", s)
		}
	}
}

func TestSemVerCompare(t *testing.T) {
	lt := func(a, b string) {
		t.Helper()
		if mustSemVer(t, a).Compare(mustSemVer(t, b)) != -1 {
			t.Errorf("%s should be < %s", a, b)
		}
		if mustSemVer(t, b).Compare(mustSemVer(t, a)) != 1 {
			t.Errorf("%s should be > %s", b, a)
		}
	}
	lt("1.0.0", "2.0.0")
	lt("1.1.0", "1.2.0")
	lt("1.0.1", "1.0.2")
	lt("1.0.0-alpha", "1.0.0")           // pre-release < release
	lt("1.0.0-alpha", "1.0.0-alpha.1")   // fewer identifiers first
	lt("1.0.0-alpha.1", "1.0.0-alpha.2") // numeric compare
	lt("1.0.0-alpha", "1.0.0-beta")      // lexical
	lt("1.0.0-1", "1.0.0-alpha")         // numeric < alnum
	if mustSemVer(t, "1.0.0+a").Compare(mustSemVer(t, "1.0.0+b")) != 0 {
		t.Error("build metadata must not affect precedence")
	}
	if mustSemVer(t, "1.2.3").Compare(mustSemVer(t, "1.2.3")) != 0 {
		t.Error("equal versions")
	}
}

func TestSemVerRangeOperators(t *testing.T) {
	in := func(rng, ver string, want bool) {
		t.Helper()
		if got := mustSemVerRange(t, rng).Includes(mustSemVer(t, ver)); got != want {
			t.Errorf("%q.Includes(%q) = %v, want %v", rng, ver, got, want)
		}
	}
	in(">=1.0.0", "1.0.0", true)
	in(">=1.0.0", "0.9.0", false)
	in(">1.0.0", "1.0.0", false)
	in("<=2.0.0", "2.0.0", true)
	in("<2.0.0", "2.0.0", false)
	in("=1.2.3", "1.2.3", true)
	in("1.2.3", "1.2.4", false)
	in(">=1.0.0 <2.0.0", "1.5.0", true)
	in(">=1.0.0 <2.0.0", "2.0.0", false)
	in("<1.0.0 || >=2.0.0", "2.1.0", true)
	in("<1.0.0 || >=2.0.0", "1.5.0", false)
	in("*", "9.9.9", true)
	in("", "1.0.0", true)
	// Exact pre-release version as a bare comparator.
	in("1.0.0-rc.1", "1.0.0-rc.1", true)
	in("1.0.0-rc.1", "1.0.0", false)
	in(">=1.0.0-rc.1", "1.0.0-rc.2", true)
}

func TestSemVerRangeShorthands(t *testing.T) {
	in := func(rng, ver string, want bool) {
		t.Helper()
		if got := mustSemVerRange(t, rng).Includes(mustSemVer(t, ver)); got != want {
			t.Errorf("%q.Includes(%q) = %v, want %v", rng, ver, got, want)
		}
	}
	// x-ranges
	in("1.2.x", "1.2.9", true)
	in("1.2.x", "1.3.0", false)
	in("1.x", "1.9.9", true)
	in("1.x", "2.0.0", false)
	in("1.2.*", "1.2.0", true)
	// tilde
	in("~1.2.3", "1.2.9", true)
	in("~1.2.3", "1.3.0", false)
	in("~1.2", "1.2.5", true)
	in("~1", "1.9.9", true)
	in("~1", "2.0.0", false)
	// caret
	in("^1.2.3", "1.9.9", true)
	in("^1.2.3", "2.0.0", false)
	in("^0.2.3", "0.2.9", true)
	in("^0.2.3", "0.3.0", false)
	in("^0.0.3", "0.0.3", true)
	in("^0.0.3", "0.0.4", false)
	// hyphen
	in("1.0.0 - 2.0.0", "1.5.0", true)
	in("1.0.0 - 2.0.0", "2.0.1", false)
	in("1.0.0 - 2.x", "2.9.9", true)
}

func TestSemVerRangeParseErrors(t *testing.T) {
	bad := []string{">=notaversion", "1.2.3 - bad", "bad - 1.2.3", "~notaver", ">= "}
	for _, s := range bad {
		if _, err := NewSemVerRange(s); err == nil {
			t.Errorf("NewSemVerRange(%q) should error", s)
		}
	}
}

func TestSemVerRangeString(t *testing.T) {
	if mustSemVerRange(t, ">=1.0.0").String() != ">=1.0.0" {
		t.Error("range String should echo source")
	}
	if mustSemVerRange(t, "*").String() != "*" {
		t.Error("match-all range String")
	}
}

// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "fmt"

// guard is the co-inductive recursion guard threaded through assignability and
// instance-of checks. Recursive type aliases (e.g. Tree = Hash[String,
// Variant[Tree, Integer]], or the degenerate Variant[Self, Integer]) make those
// relations potentially non-terminating; when the same (type, subject) pair is
// re-entered we stop and let the caller assume a fixed result. This mirrors the
// RecursionGuard Puppet's Pops type calculus uses.
type guard struct{ seen map[[2]string]bool }

func newGuard() *guard { return &guard{} }

// enter marks the pair (a, b) as being evaluated. It returns already=true if the
// pair is already on the evaluation stack, in which case the caller must return
// its co-inductive assumption without recursing further.
func (g *guard) enter(a, b string) (already bool) {
	key := [2]string{a, b}
	if g.seen[key] {
		return true
	}
	if g.seen == nil {
		g.seen = make(map[[2]string]bool)
	}
	g.seen[key] = true
	return false
}

// valueRepr is a deterministic textual fingerprint of a canonical value, used as
// the guard key for instance-of recursion. Distinct finite values yield distinct
// reprs, so the guard only trips on genuine re-entry with the same value.
func valueRepr(v Value) string {
	switch x := v.(type) {
	case []Value:
		s := "["
		for i, e := range x {
			if i > 0 {
				s += ","
			}
			s += valueRepr(e)
		}
		return s + "]"
	case *Hash:
		s := "{"
		for i, e := range x.entries {
			if i > 0 {
				s += ","
			}
			s += valueRepr(e.Key) + "=>" + valueRepr(e.Value)
		}
		return s + "}"
	default:
		// %T distinguishes scalar kinds (int64 vs float64 vs string), %v renders
		// the value (wrapper types via their String method).
		return fmt.Sprintf("%T:%v", v, v)
	}
}

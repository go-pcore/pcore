// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer is a Pcore SemVer value: a Semantic Versioning 2.0.0 version.
type SemVer struct {
	major, minor, patch uint64
	pre                 []string // dot-separated pre-release identifiers
	meta                string   // build metadata (ignored for precedence)
}

// NewSemVer parses s (e.g. "1.2.3-rc.1+build.5") into a SemVer.
func NewSemVer(s string) (*SemVer, error) {
	core := s
	meta := ""
	if i := strings.IndexByte(core, '+'); i >= 0 {
		meta = core[i+1:]
		core = core[:i]
		if meta == "" {
			return nil, fmt.Errorf("pcore: empty build metadata in SemVer %q", s)
		}
	}
	var pre []string
	if i := strings.IndexByte(core, '-'); i >= 0 {
		p := core[i+1:]
		core = core[:i]
		if p == "" {
			return nil, fmt.Errorf("pcore: empty pre-release in SemVer %q", s)
		}
		pre = strings.Split(p, ".")
		for _, id := range pre {
			if id == "" {
				return nil, fmt.Errorf("pcore: empty pre-release identifier in SemVer %q", s)
			}
		}
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("pcore: SemVer %q must be MAJOR.MINOR.PATCH", s)
	}
	nums := make([]uint64, 3)
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("pcore: invalid SemVer component %q in %q", p, s)
		}
		nums[i] = n
	}
	return &SemVer{major: nums[0], minor: nums[1], patch: nums[2], pre: pre, meta: meta}, nil
}

// String renders the version in canonical SemVer form.
func (v *SemVer) String() string {
	b := strconv.FormatUint(v.major, 10) + "." + strconv.FormatUint(v.minor, 10) + "." + strconv.FormatUint(v.patch, 10)
	if len(v.pre) > 0 {
		b += "-" + strings.Join(v.pre, ".")
	}
	if v.meta != "" {
		b += "+" + v.meta
	}
	return b
}

// Compare returns -1, 0 or +1 as v orders before, equal to, or after o under
// Semantic Versioning precedence (build metadata is ignored).
func (v *SemVer) Compare(o *SemVer) int {
	if c := cmpUint(v.major, o.major); c != 0 {
		return c
	}
	if c := cmpUint(v.minor, o.minor); c != 0 {
		return c
	}
	if c := cmpUint(v.patch, o.patch); c != 0 {
		return c
	}
	return cmpPre(v.pre, o.pre)
}

func cmpUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// cmpPre compares pre-release identifier lists per SemVer §11.4. A version with
// pre-release identifiers has lower precedence than one without.
func cmpPre(a, b []string) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return 1 // no pre-release > has pre-release
	}
	if len(b) == 0 {
		return -1
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if c := cmpPreIdent(a[i], b[i]); c != 0 {
			return c
		}
	}
	return cmpInt(len(a), len(b))
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpPreIdent(a, b string) int {
	an, aerr := strconv.ParseUint(a, 10, 64)
	bn, berr := strconv.ParseUint(b, 10, 64)
	aNum, bNum := aerr == nil, berr == nil
	switch {
	case aNum && bNum:
		return cmpUint(an, bn)
	case aNum: // numeric identifiers are lower than alphanumeric
		return -1
	case bNum:
		return 1
	default:
		return strings.Compare(a, b)
	}
}

// SemVerRange is a Pcore SemVerRange value: a set of version constraints
// (npm/semantic_puppet grammar) matched against a SemVer.
type SemVerRange struct {
	orig     string
	orSets   [][]verComparator // OR of AND-sets of comparators
	matchAll bool
}

type verComparator struct {
	op  string // one of "<", "<=", ">", ">=", "="
	ver *SemVer
}

// NewSemVerRange parses a version range expression.
func NewSemVerRange(s string) (*SemVerRange, error) {
	r := &SemVerRange{orig: s}
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == "*" {
		r.matchAll = true
		return r, nil
	}
	for _, part := range strings.Split(trimmed, "||") {
		set, err := parseComparatorSet(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		r.orSets = append(r.orSets, set)
	}
	return r, nil
}

// parseComparatorSet parses a whitespace-separated AND-set, expanding the
// hyphen range, x-ranges, tilde and caret shorthands into simple comparators.
func parseComparatorSet(s string) ([]verComparator, error) {
	fields := strings.Fields(s)
	// Hyphen range: "A - B".
	if len(fields) == 3 && fields[1] == "-" {
		lo, err := lowerBound(fields[0])
		if err != nil {
			return nil, err
		}
		hi, err := hyphenUpper(fields[2])
		if err != nil {
			return nil, err
		}
		return append(lo, hi...), nil
	}
	var out []verComparator
	for _, f := range fields {
		cs, err := parseComparator(f)
		if err != nil {
			return nil, err
		}
		out = append(out, cs...)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("pcore: empty comparator set in SemVerRange %q", s)
	}
	return out, nil
}

func parseComparator(f string) ([]verComparator, error) {
	switch {
	case strings.HasPrefix(f, "~"):
		return tildeRange(f[1:])
	case strings.HasPrefix(f, "^"):
		return caretRange(f[1:])
	case strings.HasPrefix(f, ">="):
		return single(">=", f[2:])
	case strings.HasPrefix(f, "<="):
		return single("<=", f[2:])
	case strings.HasPrefix(f, ">"):
		return single(">", f[1:])
	case strings.HasPrefix(f, "<"):
		return single("<", f[1:])
	case strings.HasPrefix(f, "="):
		return exact(f[1:])
	default:
		return exact(f)
	}
}

func single(op, v string) ([]verComparator, error) {
	sv, err := NewSemVer(fullVersion(v))
	if err != nil {
		return nil, err
	}
	return []verComparator{{op: op, ver: sv}}, nil
}

// exact turns a bare (possibly partial) version into comparators. A full x.y.z
// is an equality; a partial version becomes a range over the missing segments.
func exact(v string) ([]verComparator, error) {
	if strings.ContainsAny(v, "-+") { // a pre-release / build version is always exact
		sv, err := NewSemVer(v)
		if err != nil {
			return nil, err
		}
		return []verComparator{{op: "=", ver: sv}}, nil
	}
	seg, wild, err := splitVersion(v)
	if err != nil {
		return nil, err
	}
	if wild == 0 {
		return []verComparator{{op: "=", ver: segVer(seg)}}, nil
	}
	return lowerBound(v)
}

// lowerBound produces the comparators covering a partial version's lower end:
// ">=lo" plus, for a partial (wildcarded) version, an exclusive "<upper". It
// serves both x-ranges and the low end of a hyphen range.
func lowerBound(v string) ([]verComparator, error) {
	seg, wild, err := splitVersion(v)
	if err != nil {
		return nil, err
	}
	if wild == 3 { // "*" — no constraint
		return nil, nil
	}
	lo := []verComparator{{op: ">=", ver: segVer(seg)}}
	if wild == 0 {
		return lo, nil
	}
	return append(lo, verComparator{op: "<", ver: upperFor(seg, wild)}), nil
}

// hyphenUpper produces the upper comparator for the high end of a hyphen range.
func hyphenUpper(v string) ([]verComparator, error) {
	seg, wild, err := splitVersion(v)
	if err != nil {
		return nil, err
	}
	if wild == 0 {
		return []verComparator{{op: "<=", ver: segVer(seg)}}, nil
	}
	return []verComparator{{op: "<", ver: upperFor(seg, wild)}}, nil
}

func segVer(seg [3]uint64) *SemVer {
	return &SemVer{major: seg[0], minor: seg[1], patch: seg[2]}
}

// upperFor returns the exclusive upper bound for a partial version whose first
// wild-th segments were wildcards (wild counts trailing wildcard segments).
func upperFor(seg [3]uint64, wild int) *SemVer {
	switch wild {
	case 1: // x.y.* -> < x.(y+1).0
		return &SemVer{major: seg[0], minor: seg[1] + 1, patch: 0}
	default: // x.* -> < (x+1).0.0
		return &SemVer{major: seg[0] + 1, minor: 0, patch: 0}
	}
}

func tildeRange(v string) ([]verComparator, error) {
	seg, wild, err := splitVersion(v)
	if err != nil {
		return nil, err
	}
	lo := segVer(seg)
	var hi *SemVer
	if wild >= 2 { // ~1 -> >=1.0.0 <2.0.0
		hi = &SemVer{major: seg[0] + 1}
	} else { // ~1.2 or ~1.2.3 -> >=lo <1.3.0
		hi = &SemVer{major: seg[0], minor: seg[1] + 1}
	}
	return []verComparator{{op: ">=", ver: lo}, {op: "<", ver: hi}}, nil
}

func caretRange(v string) ([]verComparator, error) {
	seg, _, err := splitVersion(v)
	if err != nil {
		return nil, err
	}
	lo := segVer(seg)
	var hi *SemVer
	switch {
	case seg[0] > 0:
		hi = &SemVer{major: seg[0] + 1}
	case seg[1] > 0:
		hi = &SemVer{major: 0, minor: seg[1] + 1}
	default:
		hi = &SemVer{major: 0, minor: 0, patch: seg[2] + 1}
	}
	return []verComparator{{op: ">=", ver: lo}, {op: "<", ver: hi}}, nil
}

// splitVersion parses up to three numeric segments; "x", "X" and "*" segments
// (and absent segments) are wildcards. It returns the segment values (wildcards
// as 0) and the count of trailing wildcard segments. A non-numeric,
// non-wildcard segment is an error.
func splitVersion(v string) (seg [3]uint64, wild int, err error) {
	parts := strings.Split(v, ".")
	if len(parts) > 3 {
		return seg, 0, fmt.Errorf("pcore: invalid version %q in SemVerRange", v)
	}
	for i := 0; i < 3; i++ {
		if i >= len(parts) || isWildcard(parts[i]) {
			return seg, 3 - i, nil
		}
		n, perr := strconv.ParseUint(parts[i], 10, 64)
		if perr != nil {
			return seg, 0, fmt.Errorf("pcore: invalid version segment %q in SemVerRange", parts[i])
		}
		seg[i] = n
	}
	return seg, 0, nil
}

func isWildcard(s string) bool { return s == "x" || s == "X" || s == "*" }

// fullVersion pads a partial version with zeros so NewSemVer accepts it.
func fullVersion(v string) string {
	n := strings.Count(v, ".")
	for ; n < 2; n++ {
		v += ".0"
	}
	return v
}

// Includes reports whether v satisfies the range.
func (r *SemVerRange) Includes(v *SemVer) bool {
	if r.matchAll {
		return true
	}
	for _, set := range r.orSets {
		if satisfiesAll(v, set) {
			return true
		}
	}
	return false
}

func satisfiesAll(v *SemVer, set []verComparator) bool {
	for _, c := range set {
		if !c.satisfied(v) {
			return false
		}
	}
	return true
}

func (c verComparator) satisfied(v *SemVer) bool {
	cmp := v.Compare(c.ver)
	switch c.op {
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	default: // "="
		return cmp == 0
	}
}

// String renders the range using its original source expression.
func (r *SemVerRange) String() string {
	if r.matchAll {
		return "*"
	}
	return r.orig
}

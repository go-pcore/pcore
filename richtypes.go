// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import (
	"strings"
	"time"
)

// --- rich scalar and data type structs -------------------------------------

// semVerType is SemVer[range, ...]. With no ranges it admits any SemVer value.
type semVerType struct{ ranges []*SemVerRange }

// semVerRangeType is the SemVerRange type. Its instances are SemVerRange values.
type semVerRangeType struct{}

// richDataType is RichData: Data widened with the rich scalar/wrapper types.
type richDataType struct{}

// richDataKeyType is RichDataKey: Variant[String, Numeric], the admissible key
// type of a RichData hash.
type richDataKeyType struct{}

// initType is Init[T, args...]: a value from which a T can be created.
type initType struct {
	typ  Type
	args []Type
}

// --- Name methods ----------------------------------------------------------

func (*semVerType) Name() string     { return "SemVer" }
func (semVerRangeType) Name() string { return "SemVerRange" }
func (richDataType) Name() string    { return "RichData" }
func (richDataKeyType) Name() string { return "RichDataKey" }
func (*initType) Name() string       { return "Init" }

// --- String methods --------------------------------------------------------

func (t *semVerType) String() string {
	if len(t.ranges) == 0 {
		return "SemVer"
	}
	parts := make([]string, len(t.ranges))
	for i, r := range t.ranges {
		parts[i] = quote(r.String())
	}
	return "SemVer[" + strings.Join(parts, ", ") + "]"
}

func (semVerRangeType) String() string { return "SemVerRange" }
func (richDataType) String() string    { return "RichData" }
func (richDataKeyType) String() string { return "RichDataKey" }

func (t *initType) String() string {
	if t.typ == nil {
		return "Init"
	}
	parts := []string{t.typ.String()}
	for _, a := range t.args {
		parts = append(parts, a.String())
	}
	return "Init[" + strings.Join(parts, ", ") + "]"
}

func (t *timestampType) String() string {
	if t.min == minInt && t.max == maxInt {
		return "Timestamp"
	}
	return "Timestamp[" + tsBound(t.min) + ", " + tsBound(t.max) + "]"
}

func (t *timespanType) String() string {
	if t.min == minInt && t.max == maxInt {
		return "Timespan"
	}
	return "Timespan[" + spanBound(t.min) + ", " + spanBound(t.max) + "]"
}

// tsBound renders one Timestamp bound: 'default' for the open ends, else a
// single-quoted RFC 3339 (nanosecond) instant.
func tsBound(n int64) string {
	if n == minInt || n == maxInt {
		return "default"
	}
	return quote(time.Unix(0, n).UTC().Format(time.RFC3339Nano))
}

// spanBound renders one Timespan bound.
func spanBound(n int64) string {
	if n == minInt || n == maxInt {
		return "default"
	}
	return quote(time.Duration(n).String())
}

// --- isInstance methods (RichData / RichDataKey live in instance.go) --------

func (t *semVerType) isInstance(v Value, _ *guard) bool {
	sv, ok := v.(*SemVer)
	if !ok {
		return false
	}
	if len(t.ranges) == 0 {
		return true
	}
	for _, r := range t.ranges {
		if r.Includes(sv) {
			return true
		}
	}
	return false
}

func (semVerRangeType) isInstance(v Value, _ *guard) bool {
	_, ok := v.(*SemVerRange)
	return ok
}

func (t *initType) isInstance(v Value, g *guard) bool {
	if t.typ == nil {
		return true
	}
	return t.typ.isInstance(v, g)
}

// --- isAssignable methods ---------------------------------------------------

func (t *semVerType) isAssignable(other Type, _ *guard) bool {
	o, ok := other.(*semVerType)
	if !ok {
		return false
	}
	if len(t.ranges) == 0 {
		return true
	}
	return t.String() == o.String()
}

func (semVerRangeType) isAssignable(other Type, _ *guard) bool {
	_, ok := other.(*semVerRangeType)
	return ok
}

func (t *initType) isAssignable(other Type, g *guard) bool {
	if t.typ == nil {
		return true
	}
	return asg(t.typ, other, g)
}

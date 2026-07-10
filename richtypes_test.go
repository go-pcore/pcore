// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import (
	"testing"
	"time"
)

func TestSemVerType(t *testing.T) {
	if mustParse(t, "SemVer").String() != "SemVer" {
		t.Error("generic SemVer string")
	}
	v := mustSemVer(t, "1.5.0")
	if !IsInstance(mustParse(t, "SemVer"), v) {
		t.Error("any SemVer is a SemVer")
	}
	if IsInstance(mustParse(t, "SemVer"), int64(1)) {
		t.Error("int is not a SemVer")
	}
	ranged := mustParse(t, "SemVer['>=1.0.0 <2.0.0']")
	if ranged.String() != "SemVer['>=1.0.0 <2.0.0']" {
		t.Errorf("ranged SemVer string = %q", ranged.String())
	}
	if !IsInstance(ranged, v) {
		t.Error("1.5.0 in >=1.0.0 <2.0.0")
	}
	if IsInstance(ranged, mustSemVer(t, "2.1.0")) {
		t.Error("2.1.0 not in range")
	}
	// Multiple ranges.
	multi := mustParse(t, "SemVer['1.x', '3.x']")
	if !IsInstance(multi, mustSemVer(t, "3.4.5")) {
		t.Error("3.4.5 in 3.x")
	}
	if IsInstance(multi, mustSemVer(t, "2.0.0")) {
		t.Error("2.0.0 not in 1.x|3.x")
	}
}

func TestSemVerTypeAssignable(t *testing.T) {
	if !IsAssignable(mustParse(t, "SemVer"), mustParse(t, "SemVer['1.x']")) {
		t.Error("SemVer accepts any ranged SemVer")
	}
	if !IsAssignable(mustParse(t, "SemVer['1.x']"), mustParse(t, "SemVer['1.x']")) {
		t.Error("equal ranged SemVers assignable")
	}
	if IsAssignable(mustParse(t, "SemVer['1.x']"), mustParse(t, "SemVer['2.x']")) {
		t.Error("different ranged SemVers not assignable")
	}
	if IsAssignable(mustParse(t, "SemVer"), mustParse(t, "Integer")) {
		t.Error("SemVer not assignable from Integer")
	}
	// SemVer is a Scalar.
	if !IsAssignable(mustParse(t, "Scalar"), mustParse(t, "SemVer")) {
		t.Error("SemVer <: Scalar")
	}
}

func TestSemVerRangeType(t *testing.T) {
	if !IsInstance(mustParse(t, "SemVerRange"), mustSemVerRange(t, ">=1.0.0")) {
		t.Error("SemVerRange value is a SemVerRange")
	}
	if IsInstance(mustParse(t, "SemVerRange"), int64(1)) {
		t.Error("int is not a SemVerRange")
	}
	if !IsAssignable(mustParse(t, "SemVerRange"), mustParse(t, "SemVerRange")) {
		t.Error("SemVerRange <: SemVerRange")
	}
	if IsAssignable(mustParse(t, "SemVerRange"), mustParse(t, "Integer")) {
		t.Error("SemVerRange not from Integer")
	}
	if mustParse(t, "SemVerRange").Name() != "SemVerRange" {
		t.Error("SemVerRange name")
	}
}

func TestSemVerBuildErrors(t *testing.T) {
	for _, s := range []string{"SemVer[1]", "SemVer['badrange (']"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestInitType(t *testing.T) {
	if mustParse(t, "Init").String() != "Init" {
		t.Error("generic Init string")
	}
	if !IsInstance(mustParse(t, "Init"), int64(1)) {
		t.Error("generic Init accepts anything")
	}
	it := mustParse(t, "Init[Integer[0, 10]]")
	if it.String() != "Init[Integer[0, 10]]" {
		t.Errorf("Init string = %q", it.String())
	}
	if it.Name() != "Init" {
		t.Error("Init name")
	}
	if !IsInstance(it, int64(5)) {
		t.Error("5 can init Integer[0,10]")
	}
	if IsInstance(it, int64(50)) {
		t.Error("50 cannot init Integer[0,10]")
	}
	// Assignability: Init[T] accepts anything that is a T.
	if !IsAssignable(it, mustParse(t, "Integer[2, 4]")) {
		t.Error("Integer[2,4] <: Init[Integer[0,10]]")
	}
	if !IsAssignable(mustParse(t, "Init"), mustParse(t, "String")) {
		t.Error("generic Init accepts anything")
	}
	// With extra construction arg types.
	withArgs := mustParse(t, "Init[String, Integer]")
	if withArgs.String() != "Init[String, Integer]" {
		t.Errorf("Init args string = %q", withArgs.String())
	}
}

func TestInitBuildErrors(t *testing.T) {
	for _, s := range []string{"Init[1]", "Init[String, 2]"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestTimestampRange(t *testing.T) {
	if mustParse(t, "Timestamp").String() != "Timestamp" {
		t.Error("generic Timestamp string")
	}
	ty := mustParse(t, "Timestamp['2020-01-01T00:00:00Z', '2020-12-31T23:59:59Z']")
	mid := NewTimestamp(time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC))
	if !IsInstance(ty, mid) {
		t.Error("mid-2020 timestamp in range")
	}
	out := NewTimestamp(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
	if IsInstance(ty, out) {
		t.Error("2021 timestamp out of range")
	}
	if IsInstance(ty, int64(1)) {
		t.Error("int is not a Timestamp")
	}
	// Round-trips through Parse.
	if _, err := Parse(ty.String()); err != nil {
		t.Errorf("Timestamp range round-trip: %v", err)
	}
	// Epoch-seconds and default bounds.
	lo := mustParse(t, "Timestamp[1577836800]")
	if !IsInstance(lo, NewTimestamp(time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC))) {
		t.Error("epoch-second lower bound")
	}
	if mustParse(t, "Timestamp[default, 0]").String() != "Timestamp[default, '1970-01-01T00:00:00Z']" {
		t.Errorf("default bound string = %q", mustParse(t, "Timestamp[default, 0]").String())
	}
	// Float bound.
	mustParse(t, "Timestamp[1577836800.5]")
	// Assignability: narrower within wider.
	if !IsAssignable(mustParse(t, "Timestamp"), ty) {
		t.Error("ranged Timestamp <: Timestamp")
	}
	if IsAssignable(ty, mustParse(t, "Timestamp")) {
		t.Error("Timestamp not <: ranged")
	}
}

func TestTimespanRange(t *testing.T) {
	ty := mustParse(t, "Timespan['1s', '1m0s']")
	if !IsInstance(ty, NewTimespan(30*time.Second)) {
		t.Error("30s in [1s,1m]")
	}
	if IsInstance(ty, NewTimespan(2*time.Minute)) {
		t.Error("2m out of [1s,1m]")
	}
	if IsInstance(ty, int64(1)) {
		t.Error("int is not a Timespan")
	}
	if _, err := Parse(ty.String()); err != nil {
		t.Errorf("Timespan round-trip: %v", err)
	}
	mustParse(t, "Timespan[5]")   // seconds int
	mustParse(t, "Timespan[1.5]") // seconds float
	mustParse(t, "Timespan[default, 10]")
	if !IsAssignable(mustParse(t, "Timespan"), ty) {
		t.Error("ranged Timespan <: Timespan")
	}
}

func TestTimeRangeBuildErrors(t *testing.T) {
	bad := []string{
		"Timestamp['not-a-date']",
		"Timestamp[true]",
		"Timestamp[1, 2, 3]",
		"Timespan['not-a-dur']",
		"Timespan[true]",
		"Timespan[1, 2, 3]",
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestRichData(t *testing.T) {
	rd := mustParse(t, "RichData")
	if rd.String() != "RichData" || rd.Name() != "RichData" {
		t.Error("RichData string/name")
	}
	// Rich leaves.
	for _, v := range []Value{
		int64(1), 1.5, "x", true, Undef, Default,
		NewBinary([]byte{1}), NewTimestamp(time.Unix(0, 0)),
		NewTimespan(time.Second), mustSemVer(t, "1.0.0"),
		mustSemVerRange(t, ">=1.0.0"), NewSensitive("s"), mustParse(t, "Integer"),
	} {
		if !IsInstance(rd, v) {
			t.Errorf("%v (%T) should be RichData", v, v)
		}
	}
	// Collections of rich data, with numeric keys allowed.
	if !IsInstance(rd, NewHash(HashEntry{int64(1), NewBinary([]byte{2})})) {
		t.Error("hash with numeric key and binary value is RichData")
	}
	if !IsInstance(rd, []Value{mustSemVer(t, "1.0.0"), int64(2)}) {
		t.Error("array of rich data")
	}
	// A channel is not rich data.
	if IsInstance(rd, make(chan int)) {
		t.Error("chan is not RichData")
	}
	// Nested non-rich value fails.
	if IsInstance(rd, []Value{make(chan int)}) {
		t.Error("array with non-rich element")
	}
	if IsInstance(rd, NewHash(HashEntry{int64(1), make(chan int)})) {
		t.Error("hash with non-rich value")
	}
	// Hash with a non-key type key (bool) fails RichDataKey.
	if IsInstance(rd, NewHash(HashEntry{true, int64(1)})) {
		t.Error("hash with bool key is not RichData")
	}
}

func TestRichDataAssignable(t *testing.T) {
	rd := mustParse(t, "RichData")
	if !IsAssignable(rd, mustParse(t, "Data")) {
		t.Error("Data <: RichData")
	}
	if !IsAssignable(rd, mustParse(t, "Array[Binary]")) {
		t.Error("Array[Binary] <: RichData")
	}
	if !IsAssignable(rd, mustParse(t, "Hash[String, Timestamp]")) {
		t.Error("Hash[String,Timestamp] <: RichData")
	}
	if !IsAssignable(rd, mustParse(t, "Tuple[Binary, Integer]")) {
		t.Error("Tuple <: RichData")
	}
	if !IsAssignable(rd, mustParse(t, "Struct[{'a' => Binary}]")) {
		t.Error("Struct <: RichData")
	}
	if IsAssignable(rd, mustParse(t, "Hash[Boolean, Integer]")) {
		t.Error("Hash with Boolean key not <: RichData")
	}
	// allowsUndef through RichData (NotUndef rule).
	if IsAssignable(mustParse(t, "NotUndef"), rd) {
		t.Error("RichData allows undef so NotUndef must reject it")
	}
	// RichDataKey type.
	rdk := mustParse(t, "RichDataKey")
	if rdk.String() != "RichDataKey" || rdk.Name() != "RichDataKey" {
		t.Error("RichDataKey string/name")
	}
	if !IsInstance(rdk, "s") || !IsInstance(rdk, int64(1)) || !IsInstance(rdk, 1.5) {
		t.Error("RichDataKey accepts strings and numbers")
	}
	if IsInstance(rdk, true) {
		t.Error("RichDataKey rejects bool")
	}
	if !IsAssignable(rdk, mustParse(t, "String")) || !IsAssignable(rdk, mustParse(t, "Integer")) {
		t.Error("RichDataKey assignable from String/Integer")
	}
	if IsAssignable(rdk, mustParse(t, "Boolean")) {
		t.Error("RichDataKey not from Boolean")
	}
}

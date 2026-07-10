// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "testing"

// TestV2NamesAndConstructors exercises Name on every new type plus the exported
// singleton constructors.
func TestV2NamesAndConstructors(t *testing.T) {
	names := map[string]string{
		"SemVer": "SemVer", "SemVerRange": "SemVerRange", "RichData": "RichData",
		"RichDataKey": "RichDataKey", "Init": "Init", "Object": "Object",
		"Runtime": "Runtime", "URI": "URI", "Iterable": "Iterable",
		"Iterator": "Iterator", "Error": "Error", "Callable": "Callable",
	}
	for expr, want := range names {
		if got := mustParse(t, expr).Name(); got != want {
			t.Errorf("Name(%s) = %q, want %q", expr, got, want)
		}
	}
	if RichDataT().String() != "RichData" {
		t.Error("RichDataT")
	}
	if SemVerT().String() != "SemVer" {
		t.Error("SemVerT")
	}
	if TimestampT().String() != "Timestamp" || TimespanT().String() != "Timespan" {
		t.Error("Timestamp/Timespan constructors")
	}
}

// TestV2ObjectDefaultsString parses an Object whose attributes carry defaults of
// every scalar kind and renders it, covering parseValue, parseDataArray and
// valueLiteral branches.
func TestV2ObjectDefaultsString(t *testing.T) {
	src := `Object[{name => 'K', attributes => {
		'a' => {'type' => Float, 'value' => 1.5},
		'b' => {'type' => Regexp, 'value' => /x/},
		'c' => {'type' => Array[Integer], 'value' => [1, 2]},
		'd' => {'type' => Boolean, 'value' => true},
		'e' => {'type' => Boolean, 'value' => false},
		'f' => {'type' => Optional[Integer], 'value' => undef},
		'g' => {'type' => Integer, 'value' => 5},
		'h' => {'type' => String, 'value' => 'x'},
		'i' => {'type' => Default, 'value' => default}
	}}]`
	ty, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if ty.String() == "" {
		t.Error("object string")
	}
	// Building fills in every declared default.
	ov, err := NewObjectValue(ty, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := ov.Get("c"); len(v.([]Value)) != 2 {
		t.Error("array default")
	}
}

func TestV2ObjectInlineParentString(t *testing.T) {
	ty := mustParse(t, "Object[{name => 'C', parent => Object[{name => 'P'}], attributes => {'x' => Integer}}]")
	// Non-alias (inline) parent renders via its full form.
	if s := ty.String(); s == "" {
		t.Error("inline-parent object string")
	}
	if _, err := Parse(ty.String()); err != nil {
		t.Errorf("inline-parent round-trip: %v", err)
	}
}

func TestV2BaseObjectAssignable(t *testing.T) {
	dog := mustParse(t, "Object[{name => 'Dog'}]")
	if !IsAssignable(mustParse(t, "Object"), dog) {
		t.Error("base Object accepts any object type")
	}
}

func TestV2ParseValueAndHashErrors(t *testing.T) {
	bad := []string{
		"Object[{name => ]}]",      // parseValue unexpected token
		"Object[{5 => 1}]",         // parseHashKey non-key
		"Object[{name 'X'}]",       // missing '=>' in hash
		"Object[{name => Bogus[}]", // value parse error
		"Object[{name => 'X'}}]",   // missing ']' after Object hash
		`Object[{name=>'X', attributes=>{'a'=>{'type'=>Integer,'value'=>99999999999999999999}}}]`, // int overflow value
		"Object[{name=>'X', attributes=>{'a'=>{'type'=>Array[Integer],'value'=>[1 2]}}}]",         // array missing comma
		"Object[{name=>'X', attributes=>{'a'=>{'type'=>Array[Integer],'value'=>[Bogus[]}}}]",      // array value parse error
		"Object[{name=>'X', attributes=>{'a'=>{'type'=>Regexp,'value'=>/(/}}}]",                   // invalid regexp value
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestV2AliasRecursionReentry(t *testing.T) {
	// isInstance co-inductive re-entry: a self-referential Variant.
	loopy := mustLoaderParse(t, "Loopy", "type Loopy = Variant[Loopy, Integer]")
	if !IsInstance(loopy, int64(3)) {
		t.Error("Loopy accepts an Integer despite self-reference")
	}
	if IsInstance(loopy, "x") {
		t.Error("Loopy rejects a String")
	}
	// isAssignable co-inductive re-entry via a recursive alias (Integer <: Loopy).
	if !IsAssignable(loopy, mustParse(t, "Integer")) {
		t.Error("Integer <: Loopy")
	}
}

func TestV2AliasMoreErrors(t *testing.T) {
	l := NewLoader()
	bad := []string{
		"type Foo => Bar",    // '=>' before any '=' => no assignment
		"type = Integer",     // empty name
		"type A-B = Integer", // invalid character in name
	}
	for _, d := range bad {
		if err := l.Declare(d); err == nil {
			t.Errorf("Declare(%q) should error", d)
		}
	}
	// parseRaw errors: tokenize failure, bad body, and trailing input.
	if _, err := l.Parse("@"); err == nil {
		t.Error("un-tokenizable input should error")
	}
	if _, err := l.Parse("Bogus["); err == nil {
		t.Error("bad expression should error")
	}
	if _, err := l.Parse("Integer Junk"); err == nil {
		t.Error("trailing input should error")
	}
}

func TestV2RichDataAssignableNegatives(t *testing.T) {
	rd := mustParse(t, "RichData")
	if IsAssignable(rd, mustParse(t, "Tuple[Runtime]")) {
		t.Error("Tuple with a non-rich element not <: RichData")
	}
	if IsAssignable(rd, mustParse(t, "Struct[{'a' => Runtime}]")) {
		t.Error("Struct with a non-rich member not <: RichData")
	}
	if IsAssignable(rd, mustParse(t, "Runtime")) {
		t.Error("Runtime not <: RichData")
	}
	// Data default arm: a non-data, non-collection type.
	if IsAssignable(mustParse(t, "Data"), mustParse(t, "Runtime")) {
		t.Error("Runtime not <: Data")
	}
	// Enum assignable default arm.
	if IsAssignable(mustParse(t, "Enum['a']"), mustParse(t, "Integer")) {
		t.Error("Integer not <: Enum")
	}
}

func TestV2TupleFromArray(t *testing.T) {
	// tupleType.isAssignable from an Array.
	if !IsAssignable(mustParse(t, "Tuple[Integer, Integer]"), mustParse(t, "Array[Integer, 2, 2]")) {
		t.Error("Array[Integer,2,2] <: Tuple[Integer,Integer]")
	}
	if IsAssignable(mustParse(t, "Tuple[Integer, String]"), mustParse(t, "Array[Integer, 2, 2]")) {
		t.Error("Array[Integer] not <: Tuple[Integer,String]")
	}
	if IsAssignable(mustParse(t, "Tuple[Integer]"), mustParse(t, "Integer")) {
		t.Error("Tuple not assignable from a scalar")
	}
}

func TestV2CallableParamsAndArity(t *testing.T) {
	// Repeating-parameter position (paramAt fallthrough) with covered arity: the
	// subtype has more declared params, the supertype's last param repeats.
	if !IsAssignable(mustParse(t, "Callable[Integer, 0, 3]"), mustParse(t, "Callable[Integer, Integer, 0, 3]")) {
		t.Error("Callable[Integer,0,3] should accept Callable[Integer,Integer,0,3]")
	}
	// Empty-params callable (Callable[0,0]) exercises the Any paramAt branch.
	if IsAssignable(mustParse(t, "Callable[0, 0]"), mustParse(t, "Callable[String, 0, 0]")) {
		t.Error("Callable[0,0] cannot accept a callable requiring a String arg")
	}
	// Callable block String rendering.
	blk := mustParse(t, "Callable[Integer, Callable[String]]")
	if blk.String() != "Callable[Integer, Callable[String]]" {
		t.Errorf("callable block string = %q", blk.String())
	}
	// A 'default' upper arity bound (open max).
	if got := mustParse(t, "Callable[String, 0, default]").String(); got != "Callable[String, 0, default]" {
		t.Errorf("callable default-arity string = %q", got)
	}
}

func TestV2TimespanDefaultString(t *testing.T) {
	if got := mustParse(t, "Timespan[default, 10]").String(); got != "Timespan[default, '10s']" {
		t.Errorf("Timespan default-bound string = %q", got)
	}
}

func TestV2TimeRangeSecondBoundErrors(t *testing.T) {
	bad := []string{
		"Timestamp['bad', '2020-01-01T00:00:00Z']",
		"Timestamp['2020-01-01T00:00:00Z', 'bad']",
		"Timespan['bad', '1s']",
		"Timespan['1s', 'bad']",
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestV2SemVerRangeMoreErrors(t *testing.T) {
	bad := []string{
		"1.5.0 || *", // an OR branch that is a bare wildcard yields an empty set
		"1.0.0-",     // invalid exact pre-release version
		"^bad",       // invalid caret version
		"1.2.3.4",    // too many segments
	}
	for _, s := range bad {
		if _, err := NewSemVerRange(s); err == nil {
			t.Errorf("NewSemVerRange(%q) should error", s)
		}
	}
	// Hyphen low end as a wildcard (lowerBound wild==3 branch).
	if !mustSemVerRange(t, "* - 2.0.0").Includes(mustSemVer(t, "1.5.0")) {
		t.Error("'* - 2.0.0' should include 1.5.0")
	}
}

func TestV2RichDataSerialization(t *testing.T) {
	vals := []Value{
		mustSemVer(t, "1.2.3-rc.1"),
		mustSemVerRange(t, ">=1.0.0 <2.0.0"),
		NewURI("https://example.com/x"),
	}
	for _, v := range vals {
		d, err := ToData(v)
		if err != nil {
			t.Fatalf("ToData(%v): %v", v, err)
		}
		back, err := FromData(d)
		if err != nil {
			t.Fatalf("FromData(%v): %v", v, err)
		}
		if !equalValue(v, back) {
			t.Errorf("round-trip mismatch: %v vs %v", v, back)
		}
	}
	// Non-string payloads.
	for _, ptype := range []string{"SemVer", "SemVerRange", "URI"} {
		bad := NewHash(HashEntry{"__ptype", ptype}, HashEntry{"__pvalue", int64(1)})
		if _, err := FromData(bad); err == nil {
			t.Errorf("FromData(%s with int payload) should error", ptype)
		}
	}
	// Invalid payload content.
	badVer := NewHash(HashEntry{"__ptype", "SemVer"}, HashEntry{"__pvalue", "not-a-version"})
	if _, err := FromData(badVer); err == nil {
		t.Error("FromData(invalid SemVer) should error")
	}
	badRange := NewHash(HashEntry{"__ptype", "SemVerRange"}, HashEntry{"__pvalue", "^bad"})
	if _, err := FromData(badRange); err == nil {
		t.Error("FromData(invalid SemVerRange) should error")
	}
}

func TestV2TypeSetReferencesString(t *testing.T) {
	l := NewLoader()
	// Two references (only their metadata is rendered; they are unused).
	src := `TypeSet[{
		name => 'App', version => '1.0.0',
		references => {
			A => {name => 'X', version_range => '1.x'},
			B => {name => 'Y'}
		},
		types => {Local => Integer}
	}]`
	ty, err := l.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(ty.String()); err != nil {
		t.Errorf("two-reference TypeSet round-trip: %v", err)
	}
}

func TestV2TypeSetMoreErrors(t *testing.T) {
	bad := []string{
		`TypeSet[{name => Bogus[}]`,          // metadata value parse error via stringValue
		`TypeSet[{references => Bogus[}]`,    // references value parse error
		`TypeSet[{name => 'X'}}]`,            // missing ']' after TypeSet hash
		`TypeSet[{5 => 1}]`,                  // non-string top-level key
		`TypeSet[{types => {5 => Integer}}]`, // non-string type-member key
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

// SPDX-License-Identifier: BSD-3-Clause
// Copyright the go-pcore/pcore authors.

package pcore

import "time"

// builtinNames is the set of reserved Pcore type names; alias declarations may
// not redefine them.
var builtinNames = map[string]bool{
	"Any": true, "Scalar": true, "ScalarData": true, "Data": true, "RichData": true,
	"RichDataKey": true, "Numeric": true, "Integer": true, "Float": true,
	"String": true, "Enum": true, "Pattern": true, "Regexp": true, "Boolean": true,
	"Undef": true, "Default": true, "Binary": true, "Timestamp": true,
	"Timespan": true, "SemVer": true, "SemVerRange": true, "Collection": true,
	"Array": true, "Hash": true, "Tuple": true, "Struct": true, "Optional": true,
	"NotUndef": true, "Variant": true, "Type": true, "Sensitive": true, "Init": true,
	"Object": true, "TypeSet": true, "Runtime": true, "URI": true, "Iterable": true,
	"Iterator": true, "Error": true, "Callable": true,
}

func isBuiltinName(name string) bool { return builtinNames[name] }

// --- Timestamp / Timespan --------------------------------------------------

func buildTimestamp(params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &timestampType{min: minInt, max: maxInt}, nil
	case 1:
		lo, err := tsBoundParam(params[0], minInt)
		if err != nil {
			return nil, err
		}
		return &timestampType{min: lo, max: maxInt}, nil
	case 2:
		lo, err := tsBoundParam(params[0], minInt)
		if err != nil {
			return nil, err
		}
		hi, err := tsBoundParam(params[1], maxInt)
		if err != nil {
			return nil, err
		}
		return &timestampType{min: lo, max: hi}, nil
	default:
		return nil, p.errf("Timestamp takes at most 2 parameters")
	}
}

func tsBoundParam(pr param, def int64) (int64, error) {
	switch pr.kind {
	case pDefault:
		return def, nil
	case pInt:
		return pr.i * int64(time.Second), nil
	case pFloat:
		return int64(pr.f * float64(time.Second)), nil
	case pString:
		t, err := time.Parse(time.RFC3339Nano, pr.s)
		if err != nil {
			return 0, &ParseError{Msg: "invalid Timestamp bound " + quote(pr.s), Pos: pr.pos}
		}
		return t.UnixNano(), nil
	default:
		return 0, &ParseError{Msg: "Timestamp bound must be a string, number or default", Pos: pr.pos}
	}
}

func buildTimespan(params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &timespanType{min: minInt, max: maxInt}, nil
	case 1:
		lo, err := spanBoundParam(params[0], minInt)
		if err != nil {
			return nil, err
		}
		return &timespanType{min: lo, max: maxInt}, nil
	case 2:
		lo, err := spanBoundParam(params[0], minInt)
		if err != nil {
			return nil, err
		}
		hi, err := spanBoundParam(params[1], maxInt)
		if err != nil {
			return nil, err
		}
		return &timespanType{min: lo, max: hi}, nil
	default:
		return nil, p.errf("Timespan takes at most 2 parameters")
	}
}

func spanBoundParam(pr param, def int64) (int64, error) {
	switch pr.kind {
	case pDefault:
		return def, nil
	case pInt:
		return pr.i * int64(time.Second), nil
	case pFloat:
		return int64(pr.f * float64(time.Second)), nil
	case pString:
		d, err := time.ParseDuration(pr.s)
		if err != nil {
			return 0, &ParseError{Msg: "invalid Timespan bound " + quote(pr.s), Pos: pr.pos}
		}
		return int64(d), nil
	default:
		return 0, &ParseError{Msg: "Timespan bound must be a string, number or default", Pos: pr.pos}
	}
}

// --- SemVer ----------------------------------------------------------------

func buildSemVer(params []param, p *parser) (Type, error) {
	if len(params) == 0 {
		return &semVerType{}, nil
	}
	t := &semVerType{}
	for _, pr := range params {
		if pr.kind != pString {
			return nil, &ParseError{Msg: "SemVer ranges must be strings", Pos: pr.pos}
		}
		r, err := NewSemVerRange(pr.s)
		if err != nil {
			return nil, &ParseError{Msg: err.Error(), Pos: pr.pos}
		}
		t.ranges = append(t.ranges, r)
	}
	return t, nil
}

// --- Init ------------------------------------------------------------------

func buildInit(params []param, p *parser) (Type, error) {
	if len(params) == 0 {
		return &initType{}, nil
	}
	if params[0].kind != pType {
		return nil, &ParseError{Msg: "Init's first parameter must be a type", Pos: params[0].pos}
	}
	t := &initType{typ: params[0].typ}
	for _, pr := range params[1:] {
		if pr.kind != pType {
			return nil, &ParseError{Msg: "Init construction arguments must be types", Pos: pr.pos}
		}
		t.args = append(t.args, pr.typ)
	}
	return t, nil
}

// --- Runtime ---------------------------------------------------------------

func buildRuntime(params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &runtimeType{}, nil
	case 1:
		if params[0].kind != pString {
			return nil, &ParseError{Msg: "Runtime's runtime must be a string", Pos: params[0].pos}
		}
		return &runtimeType{runtime: params[0].s}, nil
	case 2:
		if params[0].kind != pString {
			return nil, &ParseError{Msg: "Runtime's runtime must be a string", Pos: params[0].pos}
		}
		t := &runtimeType{runtime: params[0].s}
		switch params[1].kind {
		case pString:
			t.name = params[1].s
		case pRegexp:
			t.pat = params[1].re
		default:
			return nil, &ParseError{Msg: "Runtime's name must be a string or regexp", Pos: params[1].pos}
		}
		return t, nil
	default:
		return nil, p.errf("Runtime takes at most 2 parameters")
	}
}

// --- URI -------------------------------------------------------------------

func buildURI(params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &uriType{}, nil
	case 1:
		if params[0].kind != pString {
			return nil, &ParseError{Msg: "URI's scheme must be a string", Pos: params[0].pos}
		}
		return &uriType{scheme: params[0].s}, nil
	default:
		return nil, p.errf("URI takes at most 1 parameter")
	}
}

// --- Iterable / Iterator ---------------------------------------------------

func buildIterable(params []param, p *parser) (Type, error) {
	elem, err := elementParam("Iterable", params, p)
	if err != nil {
		return nil, err
	}
	return &iterableType{elem: elem}, nil
}

func buildIterator(params []param, p *parser) (Type, error) {
	elem, err := elementParam("Iterator", params, p)
	if err != nil {
		return nil, err
	}
	return &iteratorType{elem: elem}, nil
}

func elementParam(name string, params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &anyType{}, nil
	case 1:
		if params[0].kind != pType {
			return nil, &ParseError{Msg: name + "'s element must be a type", Pos: params[0].pos}
		}
		return params[0].typ, nil
	default:
		return nil, p.errf("%s takes at most 1 parameter", name)
	}
}

// --- Error -----------------------------------------------------------------

func buildError(params []param, p *parser) (Type, error) {
	t := &errorType{kind: &anyType{}, issue: &anyType{}}
	switch len(params) {
	case 0:
		return t, nil
	case 2:
		if params[1].kind != pType {
			return nil, &ParseError{Msg: "Error's issue_code type must be a type", Pos: params[1].pos}
		}
		t.issue = params[1].typ
		fallthrough
	case 1:
		if params[0].kind != pType {
			return nil, &ParseError{Msg: "Error's kind type must be a type", Pos: params[0].pos}
		}
		t.kind = params[0].typ
		return t, nil
	default:
		return nil, p.errf("Error takes at most 2 parameters")
	}
}

// --- Callable --------------------------------------------------------------

func buildCallable(params []param, p *parser) (Type, error) {
	if len(params) == 0 {
		return &callableType{generic: true}, nil
	}
	t := &callableType{}
	i := 0
	for ; i < len(params) && params[i].kind == pType; i++ {
		t.params = append(t.params, params[i].typ)
	}
	// Optional arity: one or two trailing Integer/default parameters. The kinds
	// are already checked, so bound extraction cannot fail.
	if i < len(params) && isSizeParam(params[i]) {
		lo := callableBound(params[i], 0)
		i++
		hi := int64(maxInt)
		if i < len(params) && isSizeParam(params[i]) {
			hi = callableBound(params[i], maxInt)
			i++
		}
		t.minSz, t.maxSz, t.hasSize = lo, hi, true
	}
	// Optional trailing block: a Callable or Optional[Callable] type.
	if i < len(params) && params[i].kind == pType && isBlockType(params[i].typ) {
		t.block = params[i].typ
		i++
	} else if !t.hasSize && len(t.params) > 0 && isBlockType(t.params[len(t.params)-1]) {
		t.block = t.params[len(t.params)-1]
		t.params = t.params[:len(t.params)-1]
	}
	if i != len(params) {
		return nil, &ParseError{Msg: "malformed Callable parameters", Pos: params[i].pos}
	}
	return t, nil
}

// isSizeParam reports whether a parameter is an arity bound (Integer or default).
func isSizeParam(pr param) bool { return pr.kind == pInt || pr.kind == pDefault }

// callableBound extracts an arity bound; pDefault yields def.
func callableBound(pr param, def int64) int64 {
	if pr.kind == pInt {
		return pr.i
	}
	return def
}

// isBlockType reports whether t denotes a Callable block parameter.
func isBlockType(t Type) bool {
	switch x := t.(type) {
	case *callableType:
		return true
	case *optionalType:
		_, ok := x.typ.(*callableType)
		return ok
	default:
		return false
	}
}

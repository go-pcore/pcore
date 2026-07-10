package pcore

// buildType assembles a concrete Type from a parsed name and parameter list.
// Every arity or kind mismatch returns a *ParseError.
func buildType(name string, params []param, p *parser) (Type, error) {
	switch name {
	case "Any":
		return nullary(name, params, p, &anyType{})
	case "Scalar":
		return nullary(name, params, p, &scalarType{})
	case "ScalarData":
		return nullary(name, params, p, &scalarDataType{})
	case "Data":
		return nullary(name, params, p, dataType{})
	case "Numeric":
		return nullary(name, params, p, &numericType{})
	case "Undef":
		return nullary(name, params, p, &undefType{})
	case "Default":
		return nullary(name, params, p, &defaultTypeT{})
	case "Boolean":
		return nullary(name, params, p, &booleanType{})
	case "Binary":
		return nullary(name, params, p, &binaryType{})
	case "RichData":
		return nullary(name, params, p, &richDataType{})
	case "RichDataKey":
		return nullary(name, params, p, &richDataKeyType{})
	case "SemVerRange":
		return nullary(name, params, p, &semVerRangeType{})
	case "Timestamp":
		return buildTimestamp(params, p)
	case "Timespan":
		return buildTimespan(params, p)
	case "SemVer":
		return buildSemVer(params, p)
	case "Init":
		return buildInit(params, p)
	case "Runtime":
		return buildRuntime(params, p)
	case "URI":
		return buildURI(params, p)
	case "Iterable":
		return buildIterable(params, p)
	case "Iterator":
		return buildIterator(params, p)
	case "Error":
		return buildError(params, p)
	case "Callable":
		return buildCallable(params, p)
	case "Integer":
		return buildInteger(params, p)
	case "Float":
		return buildFloat(params, p)
	case "String":
		return buildString(params, p)
	case "Enum":
		return buildEnum(params, p)
	case "Pattern":
		return buildPattern(params, p)
	case "Regexp":
		return buildRegexp(params, p)
	case "Collection":
		return buildCollection(params, p)
	case "Array":
		return buildArray(params, p)
	case "Hash":
		return buildHash(params, p)
	case "Tuple":
		return buildTuple(params, p)
	case "Struct":
		return buildStruct(params, p)
	case "Variant":
		return buildVariant(params, p)
	case "Optional":
		return buildWrapper(name, params, p, func(t Type) Type { return &optionalType{typ: t} })
	case "NotUndef":
		return buildWrapper(name, params, p, func(t Type) Type { return &notUndefType{typ: t} })
	case "Type":
		return buildWrapper(name, params, p, func(t Type) Type { return &typeType{typ: t} })
	case "Sensitive":
		return buildWrapper(name, params, p, func(t Type) Type { return &sensitiveType{typ: t} })
	default:
		if p.loader != nil && isTypeName(name) {
			if len(params) != 0 {
				return nil, &ParseError{Msg: "type alias " + name + " cannot take parameters", Pos: 0}
			}
			return p.loader.ref(name), nil
		}
		return nil, &ParseError{Msg: "unknown type " + name, Pos: 0}
	}
}

func nullary(name string, params []param, p *parser, t Type) (Type, error) {
	if len(params) != 0 {
		return nil, p.errf("%s takes no parameters", name)
	}
	return t, nil
}

// intBound interprets a param as an integer bound, honoring `default`.
func intBound(pr param, def int64, p *parser) (int64, error) {
	switch pr.kind {
	case pInt:
		return pr.i, nil
	case pDefault:
		return def, nil
	default:
		return 0, &ParseError{Msg: "expected an integer bound or default", Pos: pr.pos}
	}
}

func buildInteger(params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &integerType{min: minInt, max: maxInt}, nil
	case 1:
		lo, err := intBound(params[0], minInt, p)
		if err != nil {
			return nil, err
		}
		return &integerType{min: lo, max: maxInt}, nil
	case 2:
		lo, err := intBound(params[0], minInt, p)
		if err != nil {
			return nil, err
		}
		hi, err := intBound(params[1], maxInt, p)
		if err != nil {
			return nil, err
		}
		return &integerType{min: lo, max: hi}, nil
	default:
		return nil, p.errf("Integer takes at most 2 parameters")
	}
}

// floatBound interprets a param as a float bound (ints widen).
func floatBound(pr param, def float64, p *parser) (float64, error) {
	switch pr.kind {
	case pFloat:
		return pr.f, nil
	case pInt:
		return float64(pr.i), nil
	case pDefault:
		return def, nil
	default:
		return 0, &ParseError{Msg: "expected a float bound or default", Pos: pr.pos}
	}
}

func buildFloat(params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &floatType{min: negInf(), max: posInf()}, nil
	case 1:
		lo, err := floatBound(params[0], negInf(), p)
		if err != nil {
			return nil, err
		}
		return &floatType{min: lo, max: posInf()}, nil
	case 2:
		lo, err := floatBound(params[0], negInf(), p)
		if err != nil {
			return nil, err
		}
		hi, err := floatBound(params[1], posInf(), p)
		if err != nil {
			return nil, err
		}
		return &floatType{min: lo, max: hi}, nil
	default:
		return nil, p.errf("Float takes at most 2 parameters")
	}
}

func buildString(params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &stringType{minLen: 0, maxLen: maxInt}, nil
	case 1:
		lo, err := intBound(params[0], 0, p)
		if err != nil {
			return nil, err
		}
		return &stringType{minLen: lo, maxLen: maxInt}, nil
	case 2:
		lo, err := intBound(params[0], 0, p)
		if err != nil {
			return nil, err
		}
		hi, err := intBound(params[1], maxInt, p)
		if err != nil {
			return nil, err
		}
		return &stringType{minLen: lo, maxLen: hi}, nil
	default:
		return nil, p.errf("String takes at most 2 parameters")
	}
}

func buildEnum(params []param, p *parser) (Type, error) {
	if len(params) == 0 {
		return nil, p.errf("Enum requires at least one value")
	}
	e := &enumType{}
	for i, pr := range params {
		switch pr.kind {
		case pString:
			e.values = append(e.values, pr.s)
		case pBool:
			if i != len(params)-1 || !pr.b {
				return nil, &ParseError{Msg: "Enum accepts a trailing true for case-insensitivity", Pos: pr.pos}
			}
			e.ci = true
		default:
			return nil, &ParseError{Msg: "Enum values must be strings", Pos: pr.pos}
		}
	}
	if len(e.values) == 0 {
		return nil, p.errf("Enum requires at least one value")
	}
	return e, nil
}

func buildPattern(params []param, p *parser) (Type, error) {
	if len(params) == 0 {
		return nil, p.errf("Pattern requires at least one pattern")
	}
	pt := &patternType{}
	for _, pr := range params {
		switch pr.kind {
		case pRegexp:
			pt.patterns = append(pt.patterns, pr.re)
		case pString:
			re, err := NewRegexp(pr.s)
			if err != nil {
				return nil, &ParseError{Msg: "invalid pattern string: " + err.Error(), Pos: pr.pos}
			}
			pt.patterns = append(pt.patterns, re)
		default:
			return nil, &ParseError{Msg: "Pattern arguments must be regexps or strings", Pos: pr.pos}
		}
	}
	return pt, nil
}

func buildRegexp(params []param, p *parser) (Type, error) {
	switch len(params) {
	case 0:
		return &regexpType{}, nil
	case 1:
		pr := params[0]
		switch pr.kind {
		case pRegexp:
			return &regexpType{pattern: pr.re}, nil
		case pString:
			re, err := NewRegexp(pr.s)
			if err != nil {
				return nil, &ParseError{Msg: "invalid regexp: " + err.Error(), Pos: pr.pos}
			}
			return &regexpType{pattern: re}, nil
		default:
			return nil, &ParseError{Msg: "Regexp parameter must be a regexp or string", Pos: pr.pos}
		}
	default:
		return nil, p.errf("Regexp takes at most 1 parameter")
	}
}

// sizeParams reads a trailing (min[,max]) size specification from ints/default.
func sizeParams(params []param, p *parser) (int64, int64, error) {
	switch len(params) {
	case 0:
		return 0, maxInt, nil
	case 1:
		lo, err := intBound(params[0], 0, p)
		return lo, maxInt, err
	case 2:
		lo, err := intBound(params[0], 0, p)
		if err != nil {
			return 0, 0, err
		}
		hi, err := intBound(params[1], maxInt, p)
		return lo, hi, err
	default:
		return 0, 0, p.errf("too many size parameters")
	}
}

func buildCollection(params []param, p *parser) (Type, error) {
	lo, hi, err := sizeParams(params, p)
	if err != nil {
		return nil, err
	}
	return &collectionType{minSz: lo, maxSz: hi}, nil
}

func buildArray(params []param, p *parser) (Type, error) {
	if len(params) == 0 {
		return &arrayType{element: &anyType{}, minSz: 0, maxSz: maxInt}, nil
	}
	if params[0].kind != pType {
		return nil, &ParseError{Msg: "Array element must be a type", Pos: params[0].pos}
	}
	lo, hi, err := sizeParams(params[1:], p)
	if err != nil {
		return nil, err
	}
	return &arrayType{element: params[0].typ, minSz: lo, maxSz: hi}, nil
}

func buildHash(params []param, p *parser) (Type, error) {
	switch {
	case len(params) == 0:
		return &hashType{key: &anyType{}, value: &anyType{}, minSz: 0, maxSz: maxInt}, nil
	case len(params) == 1:
		return nil, p.errf("Hash requires both a key and value type")
	default:
	}
	if params[0].kind != pType || params[1].kind != pType {
		return nil, &ParseError{Msg: "Hash key and value must be types", Pos: params[0].pos}
	}
	lo, hi, err := sizeParams(params[2:], p)
	if err != nil {
		return nil, err
	}
	return &hashType{key: params[0].typ, value: params[1].typ, minSz: lo, maxSz: hi}, nil
}

func buildTuple(params []param, p *parser) (Type, error) {
	if len(params) == 0 {
		return nil, p.errf("Tuple requires at least one element type")
	}
	t := &tupleType{}
	i := 0
	for ; i < len(params); i++ {
		if params[i].kind != pType {
			break
		}
		t.types = append(t.types, params[i].typ)
	}
	if len(t.types) == 0 {
		return nil, &ParseError{Msg: "Tuple requires at least one element type", Pos: params[0].pos}
	}
	lo, hi, err := sizeParams(params[i:], p)
	if err != nil {
		return nil, err
	}
	if len(params) == i { // no explicit size
		t.minSz, t.maxSz = int64(len(t.types)), int64(len(t.types))
	} else {
		t.minSz, t.maxSz = lo, hi
	}
	return t, nil
}

func buildStruct(params []param, p *parser) (Type, error) {
	if len(params) != 1 || params[0].kind != pStruct {
		return nil, p.errf("Struct takes a single hash of members")
	}
	return &structType{members: params[0].members}, nil
}

func buildVariant(params []param, p *parser) (Type, error) {
	if len(params) == 0 {
		return nil, p.errf("Variant requires at least one type")
	}
	v := &variantType{}
	for _, pr := range params {
		if pr.kind != pType {
			return nil, &ParseError{Msg: "Variant arguments must be types", Pos: pr.pos}
		}
		v.types = append(v.types, pr.typ)
	}
	return v, nil
}

func buildWrapper(name string, params []param, p *parser, wrap func(Type) Type) (Type, error) {
	switch len(params) {
	case 0:
		return wrap(&anyType{}), nil
	case 1:
		if params[0].kind != pType {
			return nil, &ParseError{Msg: name + " parameter must be a type", Pos: params[0].pos}
		}
		return wrap(params[0].typ), nil
	default:
		return nil, p.errf("%s takes at most 1 parameter", name)
	}
}

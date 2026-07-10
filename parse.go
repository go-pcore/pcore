package pcore

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseError describes a failure to parse a Pcore type expression.
type ParseError struct {
	Msg string
	Pos int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("pcore: parse error at %d: %s", e.Pos, e.Msg)
}

// Parse parses a Pcore type expression, e.g. "Array[Integer[0,10], 1]", into a
// [Type]. It is the Go rendering of Puppet's Type(string); Go cannot name a
// function and the [Type] interface identically, so the parser is Parse.
func Parse(s string) (Type, error) {
	toks, err := tokenize(s)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	t, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if p.cur().kind != tEOF {
		return nil, p.errf("unexpected trailing input %q", p.cur().text)
	}
	return t, nil
}

// tokenize scans the whole input up front so the parser cursor can advance
// without ever failing: all lexical errors surface here.
func tokenize(s string) ([]token, error) {
	lx := newLexer(s)
	var toks []token
	for {
		t, err := lx.next()
		if err != nil {
			return nil, err
		}
		toks = append(toks, t)
		if t.kind == tEOF {
			return toks, nil
		}
	}
}

// --- lexer ------------------------------------------------------------------

type tokKind int

const (
	tEOF tokKind = iota
	tName
	tInt
	tFloat
	tString
	tRegexp
	tLBrack
	tRBrack
	tLBrace
	tRBrace
	tComma
	tArrow
)

type token struct {
	kind tokKind
	text string
	pos  int
}

type lexer struct {
	src []rune
	pos int
}

func newLexer(s string) *lexer { return &lexer{src: []rune(s)} }

func (lx *lexer) peekRune() (rune, bool) {
	if lx.pos >= len(lx.src) {
		return 0, false
	}
	return lx.src[lx.pos], true
}

func (lx *lexer) next() (token, error) {
	for {
		r, ok := lx.peekRune()
		if !ok {
			return token{kind: tEOF, pos: lx.pos}, nil
		}
		if unicode.IsSpace(r) {
			lx.pos++
			continue
		}
		break
	}
	start := lx.pos
	r := lx.src[lx.pos]
	switch r {
	case '[':
		lx.pos++
		return token{tLBrack, "[", start}, nil
	case ']':
		lx.pos++
		return token{tRBrack, "]", start}, nil
	case '{':
		lx.pos++
		return token{tLBrace, "{", start}, nil
	case '}':
		lx.pos++
		return token{tRBrace, "}", start}, nil
	case ',':
		lx.pos++
		return token{tComma, ",", start}, nil
	case '=':
		if lx.pos+1 < len(lx.src) && lx.src[lx.pos+1] == '>' {
			lx.pos += 2
			return token{tArrow, "=>", start}, nil
		}
		return token{}, &ParseError{Msg: "expected '=>'", Pos: start}
	case '\'', '"':
		return lx.lexString(r)
	case '/':
		return lx.lexRegexp()
	}
	if r == '-' || unicode.IsDigit(r) {
		return lx.lexNumber()
	}
	if isNameStart(r) {
		return lx.lexName()
	}
	return token{}, &ParseError{Msg: fmt.Sprintf("unexpected character %q", string(r)), Pos: start}
}

func isNameStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isNamePart(r rune) bool {
	return r == '_' || r == ':' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (lx *lexer) lexName() (token, error) {
	start := lx.pos
	for lx.pos < len(lx.src) && isNamePart(lx.src[lx.pos]) {
		lx.pos++
	}
	return token{tName, string(lx.src[start:lx.pos]), start}, nil
}

func (lx *lexer) lexNumber() (token, error) {
	start := lx.pos
	if lx.src[lx.pos] == '-' {
		lx.pos++
	}
	isFloat := false
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		if unicode.IsDigit(c) {
			lx.pos++
			continue
		}
		if (c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-') && lx.floatContinues(c) {
			isFloat = true
			lx.pos++
			continue
		}
		break
	}
	text := string(lx.src[start:lx.pos])
	if text == "-" {
		return token{}, &ParseError{Msg: "malformed number", Pos: start}
	}
	if isFloat {
		return token{tFloat, text, start}, nil
	}
	return token{tInt, text, start}, nil
}

// floatContinues reports whether c at the current position is part of a float
// mantissa/exponent rather than a following token.
func (lx *lexer) floatContinues(c rune) bool {
	switch c {
	case '.':
		return lx.pos+1 < len(lx.src) && unicode.IsDigit(lx.src[lx.pos+1])
	case 'e', 'E':
		return lx.pos+1 < len(lx.src) && (unicode.IsDigit(lx.src[lx.pos+1]) || lx.src[lx.pos+1] == '+' || lx.src[lx.pos+1] == '-')
	default:
		// A '+' or '-' (the only other chars the caller passes) is part of the
		// number only immediately after an exponent marker.
		return lx.pos > 0 && (lx.src[lx.pos-1] == 'e' || lx.src[lx.pos-1] == 'E')
	}
}

func (lx *lexer) lexString(quoteCh rune) (token, error) {
	start := lx.pos
	lx.pos++ // opening quote
	var b strings.Builder
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		if c == '\\' {
			if lx.pos+1 >= len(lx.src) {
				break
			}
			esc := lx.src[lx.pos+1]
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteRune(esc)
			}
			lx.pos += 2
			continue
		}
		if c == quoteCh {
			lx.pos++
			return token{tString, b.String(), start}, nil
		}
		b.WriteRune(c)
		lx.pos++
	}
	return token{}, &ParseError{Msg: "unterminated string", Pos: start}
}

func (lx *lexer) lexRegexp() (token, error) {
	start := lx.pos
	lx.pos++ // opening slash
	var b strings.Builder
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		if c == '\\' && lx.pos+1 < len(lx.src) {
			nxt := lx.src[lx.pos+1]
			if nxt == '/' {
				b.WriteByte('/')
			} else {
				b.WriteRune(c)
				b.WriteRune(nxt)
			}
			lx.pos += 2
			continue
		}
		if c == '/' {
			lx.pos++
			return token{tRegexp, b.String(), start}, nil
		}
		b.WriteRune(c)
		lx.pos++
	}
	return token{}, &ParseError{Msg: "unterminated regexp", Pos: start}
}

// --- parser -----------------------------------------------------------------

// paramKind discriminates the kinds of bracket argument.
type paramKind int

const (
	pType paramKind = iota
	pInt
	pFloat
	pString
	pRegexp
	pDefault
	pBool
	pStruct
)

type param struct {
	kind    paramKind
	typ     Type
	i       int64
	f       float64
	s       string
	re      *Regexp
	b       bool
	members []structMember
	pos     int
}

type parser struct {
	toks   []token
	i      int
	loader *Loader // type-alias scope, or nil for the package-level Parse
}

func (p *parser) cur() token { return p.toks[p.i] }

// advance moves the cursor one token forward, never past EOF.
func (p *parser) advance() {
	if p.toks[p.i].kind != tEOF {
		p.i++
	}
}

func (p *parser) errf(format string, args ...any) error {
	return &ParseError{Msg: fmt.Sprintf(format, args...), Pos: p.cur().pos}
}

// parseType parses NAME optionally followed by [ params ].
func (p *parser) parseType() (Type, error) {
	if p.cur().kind != tName {
		return nil, p.errf("expected a type name, got %q", p.cur().text)
	}
	name := p.cur().text
	p.advance()
	if name == "TypeSet" {
		return p.parseTypeSet()
	}
	if name == "Object" {
		return p.parseObject()
	}
	var params []param
	if p.cur().kind == tLBrack {
		var err error
		params, err = p.parseParams()
		if err != nil {
			return nil, err
		}
	}
	return buildType(name, params, p)
}

// parseParams parses a bracketed, comma-separated parameter list.
func (p *parser) parseParams() ([]param, error) {
	p.advance() // consume '['
	var params []param
	if p.cur().kind == tRBrack {
		return nil, p.errf("empty parameter list")
	}
	for {
		pr, err := p.parseParam()
		if err != nil {
			return nil, err
		}
		params = append(params, pr)
		switch p.cur().kind {
		case tComma:
			p.advance()
		case tRBrack:
			p.advance() // consume ']'
			return params, nil
		default:
			return nil, p.errf("expected ',' or ']', got %q", p.cur().text)
		}
	}
}

func (p *parser) parseParam() (param, error) {
	tok := p.cur()
	switch tok.kind {
	case tInt:
		v, err := strconv.ParseInt(tok.text, 10, 64)
		if err != nil {
			return param{}, p.errf("invalid integer %q", tok.text)
		}
		p.advance()
		return param{kind: pInt, i: v, pos: tok.pos}, nil
	case tFloat:
		// The lexer only emits a float token for a well-formed mantissa/exponent,
		// so ParseFloat cannot fail here (an out-of-range value becomes ±Inf).
		v, _ := strconv.ParseFloat(tok.text, 64)
		p.advance()
		return param{kind: pFloat, f: v, pos: tok.pos}, nil
	case tString:
		p.advance()
		return param{kind: pString, s: tok.text, pos: tok.pos}, nil
	case tRegexp:
		re, err := NewRegexp(tok.text)
		if err != nil {
			return param{}, p.errf("invalid regexp /%s/: %v", tok.text, err)
		}
		p.advance()
		return param{kind: pRegexp, re: re, pos: tok.pos}, nil
	case tLBrace:
		members, err := p.parseStructBody()
		if err != nil {
			return param{}, err
		}
		return param{kind: pStruct, members: members, pos: tok.pos}, nil
	case tName:
		switch tok.text {
		case "default":
			p.advance()
			return param{kind: pDefault, pos: tok.pos}, nil
		case "true":
			p.advance()
			return param{kind: pBool, b: true, pos: tok.pos}, nil
		case "false":
			p.advance()
			return param{kind: pBool, b: false, pos: tok.pos}, nil
		}
		t, err := p.parseType()
		if err != nil {
			return param{}, err
		}
		return param{kind: pType, typ: t, pos: tok.pos}, nil
	default:
		return param{}, p.errf("unexpected %q in parameter list", tok.text)
	}
}

// parseStructBody parses { key => Type, ... }.
func (p *parser) parseStructBody() ([]structMember, error) {
	p.advance() // consume '{'
	var members []structMember
	for {
		if p.cur().kind == tRBrace {
			p.advance()
			return members, nil
		}
		name, keyOpt, err := p.parseStructKey()
		if err != nil {
			return nil, err
		}
		if p.cur().kind != tArrow {
			return nil, p.errf("expected '=>' in struct member")
		}
		p.advance()
		valType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		members = append(members, structMember{name: name, keyOpt: keyOpt, typ: valType})
		switch p.cur().kind {
		case tComma:
			p.advance()
		case tRBrace:
			// loop; top handles the closing brace
		default:
			return nil, p.errf("expected ',' or '}', got %q", p.cur().text)
		}
	}
}

// parseStructKey parses a struct key: a string, a bareword, or an
// Optional['name'] / NotUndef['name'] wrapper.
func (p *parser) parseStructKey() (name string, keyOpt bool, err error) {
	tok := p.cur()
	switch tok.kind {
	case tString:
		p.advance()
		return tok.text, false, nil
	case tName:
		wrap := tok.text
		if wrap == "Optional" || wrap == "NotUndef" {
			p.advance()
			if p.cur().kind != tLBrack {
				return "", false, p.errf("expected '[' after %s key", wrap)
			}
			p.advance()
			if p.cur().kind != tString {
				return "", false, p.errf("expected a quoted key name inside %s[...]", wrap)
			}
			name = p.cur().text
			p.advance()
			if p.cur().kind != tRBrack {
				return "", false, p.errf("expected ']' after %s key", wrap)
			}
			p.advance()
			return name, wrap == "Optional", nil
		}
		p.advance()
		return wrap, false, nil // bareword key
	default:
		return "", false, p.errf("expected a struct key, got %q", tok.text)
	}
}

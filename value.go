package pcore

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// undefValue is the type of the Undef singleton.
type undefValue struct{}

// Undef is the Pcore undef value. A Go nil is also treated as undef.
var Undef Value = undefValue{}

// String renders undef the way Puppet does.
func (undefValue) String() string { return "undef" }

// defaultValue is the type of the Default singleton.
type defaultValue struct{}

// Default is the Pcore default value (the literal `default`).
var Default Value = defaultValue{}

// String renders the default value.
func (defaultValue) String() string { return "default" }

// Sensitive wraps a value whose content must not be revealed by String or by
// [ToData]. The wrapped value is reachable through Unwrap for callers that
// legitimately need it.
type Sensitive struct{ inner Value }

// NewSensitive wraps v as a Sensitive value.
func NewSensitive(v Value) *Sensitive { return &Sensitive{inner: canon(v)} }

// Unwrap returns the wrapped value.
func (s *Sensitive) Unwrap() Value { return s.inner }

// String is deliberately redacting.
func (s *Sensitive) String() string { return "Sensitive[value redacted]" }

// Regexp is a Pcore regexp value.
type Regexp struct {
	src string
	re  *regexp.Regexp
}

// NewRegexp compiles src into a Regexp value.
func NewRegexp(src string) (*Regexp, error) {
	re, err := regexp.Compile(src)
	if err != nil {
		return nil, err
	}
	return &Regexp{src: src, re: re}, nil
}

// Source returns the pattern source (without delimiters).
func (r *Regexp) Source() string { return r.src }

// MatchString reports whether s matches the pattern.
func (r *Regexp) MatchString(s string) bool { return r.re.MatchString(s) }

// String renders the value in /slash/ form.
func (r *Regexp) String() string { return "/" + r.src + "/" }

// Binary is a Pcore binary value (a byte string).
type Binary struct{ bytes []byte }

// NewBinary wraps b as a Binary value.
func NewBinary(b []byte) *Binary { return &Binary{bytes: b} }

// Bytes returns the underlying bytes.
func (b *Binary) Bytes() []byte { return b.bytes }

// String renders the byte length (content is not dumped).
func (b *Binary) String() string { return fmt.Sprintf("Binary(%d bytes)", len(b.bytes)) }

// Timestamp is a Pcore timestamp (an instant in time).
type Timestamp struct{ t time.Time }

// NewTimestamp wraps t as a Timestamp value.
func NewTimestamp(t time.Time) *Timestamp { return &Timestamp{t: t} }

// Time returns the wrapped time.
func (ts *Timestamp) Time() time.Time { return ts.t }

// String renders the timestamp in RFC 3339 (nanosecond) form.
func (ts *Timestamp) String() string { return ts.t.UTC().Format(time.RFC3339Nano) }

// Timespan is a Pcore timespan (a duration).
type Timespan struct{ d time.Duration }

// NewTimespan wraps d as a Timespan value.
func NewTimespan(d time.Duration) *Timespan { return &Timespan{d: d} }

// Duration returns the wrapped duration.
func (ts *Timespan) Duration() time.Duration { return ts.d }

// String renders the timespan using Go's duration form.
func (ts *Timespan) String() string { return ts.d.String() }

// HashEntry is one key/value pair of a [Hash].
type HashEntry struct{ Key, Value Value }

// Hash is a Pcore hash: an ordered collection of key/value pairs whose keys may
// be any value (not just strings).
type Hash struct{ entries []HashEntry }

// NewHash builds a Hash from ordered entries.
func NewHash(entries ...HashEntry) *Hash {
	es := make([]HashEntry, len(entries))
	for i, e := range entries {
		es[i] = HashEntry{Key: canon(e.Key), Value: canon(e.Value)}
	}
	return &Hash{entries: es}
}

// Len returns the number of entries.
func (h *Hash) Len() int { return len(h.entries) }

// Entries returns the ordered entries.
func (h *Hash) Entries() []HashEntry { return h.entries }

// Get returns the value stored under a key equal to k, and whether it was
// present.
func (h *Hash) Get(k Value) (Value, bool) {
	k = canon(k)
	for _, e := range h.entries {
		if equalValue(e.Key, k) {
			return e.Value, true
		}
	}
	return nil, false
}

// String renders the hash Puppet-style.
func (h *Hash) String() string {
	var b strings.Builder
	b.WriteByte('{')
	for i, e := range h.entries {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%v => %v", e.Key, e.Value)
	}
	b.WriteByte('}')
	return b.String()
}

// canon normalizes a raw Go value into the package's canonical value form: a
// Go nil becomes Undef, the sized integer/float kinds widen to int64/float64,
// and a map[string]Value becomes an ordered *Hash (keys sorted for
// determinism).
func canon(v Value) Value {
	switch x := v.(type) {
	case nil:
		return Undef
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case float32:
		return float64(x)
	case float64:
		return x
	case map[string]Value:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		es := make([]HashEntry, len(keys))
		for i, k := range keys {
			es[i] = HashEntry{Key: k, Value: canon(x[k])}
		}
		return &Hash{entries: es}
	case []Value:
		out := make([]Value, len(x))
		for i, e := range x {
			out[i] = canon(e)
		}
		return out
	default:
		return v
	}
}

// equalValue reports deep equality of two canonical values.
func equalValue(a, b Value) bool {
	a, b = canon(a), canon(b)
	switch x := a.(type) {
	case undefValue:
		_, ok := b.(undefValue)
		return ok
	case defaultValue:
		_, ok := b.(defaultValue)
		return ok
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case int64:
		y, ok := b.(int64)
		return ok && x == y
	case float64:
		y, ok := b.(float64)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case []Value:
		y, ok := b.([]Value)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !equalValue(x[i], y[i]) {
				return false
			}
		}
		return true
	case *Hash:
		y, ok := b.(*Hash)
		if !ok || len(x.entries) != len(y.entries) {
			return false
		}
		for _, e := range x.entries {
			v2, present := y.Get(e.Key)
			if !present || !equalValue(e.Value, v2) {
				return false
			}
		}
		return true
	case *Regexp:
		y, ok := b.(*Regexp)
		return ok && x.src == y.src
	case *Binary:
		y, ok := b.(*Binary)
		if !ok || len(x.bytes) != len(y.bytes) {
			return false
		}
		for i := range x.bytes {
			if x.bytes[i] != y.bytes[i] {
				return false
			}
		}
		return true
	case *Timestamp:
		y, ok := b.(*Timestamp)
		return ok && x.t.Equal(y.t)
	case *Timespan:
		y, ok := b.(*Timespan)
		return ok && x.d == y.d
	case *Sensitive:
		y, ok := b.(*Sensitive)
		return ok && equalValue(x.inner, y.inner)
	case Type:
		y, ok := b.(Type)
		return ok && x.String() == y.String()
	default:
		return false
	}
}

// hashEntriesOf returns the ordered entries of v if it is a hash-shaped value.
func hashEntriesOf(v Value) ([]HashEntry, bool) {
	if h, ok := v.(*Hash); ok {
		return h.entries, true
	}
	return nil, false
}

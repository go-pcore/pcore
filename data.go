package pcore

import (
	"encoding/base64"
	"fmt"
	"time"
)

// Rich-data protocol keys, matching Pcore's serialization.
const (
	keyPType  = "__ptype"
	keyPValue = "__pvalue"
)

// ToData converts an arbitrary Pcore value into its rich-data representation: a
// tree of data-only values (nil, bool, int64, float64, string, []Value and
// string-keyed *Hash) in which non-data values are encoded as tagged hashes
// carrying a "__ptype" key. The result round-trips through [FromData], except
// that [Sensitive] values are redacted by design.
func ToData(v Value) (Value, error) {
	switch x := canon(v).(type) {
	case undefValue:
		return Undef, nil
	case bool, int64, float64, string:
		return x, nil
	case defaultValue:
		return tagged("Default", nil), nil
	case []Value:
		out := make([]Value, len(x))
		for i, e := range x {
			d, err := ToData(e)
			if err != nil {
				return nil, err
			}
			out[i] = d
		}
		return out, nil
	case *Hash:
		return hashToData(x)
	case *Regexp:
		return tagged("Regexp", x.src), nil
	case *Binary:
		return tagged("Binary", base64.StdEncoding.EncodeToString(x.bytes)), nil
	case *Timestamp:
		return tagged("Timestamp", x.t.UTC().Format(time.RFC3339Nano)), nil
	case *Timespan:
		return tagged("Timespan", x.d.String()), nil
	case *SemVer:
		return tagged("SemVer", x.String()), nil
	case *SemVerRange:
		return tagged("SemVerRange", x.String()), nil
	case *URI:
		return tagged("URI", x.uri), nil
	case *Sensitive:
		return tagged("Sensitive", nil), nil // redacted
	case Type:
		return tagged("Type", x.String()), nil
	default:
		return nil, fmt.Errorf("pcore: cannot serialize value of type %T", v)
	}
}

func tagged(ptype string, pvalue Value) *Hash {
	if pvalue == nil {
		return NewHash(HashEntry{Key: keyPType, Value: ptype})
	}
	return NewHash(
		HashEntry{Key: keyPType, Value: ptype},
		HashEntry{Key: keyPValue, Value: pvalue},
	)
}

func hashToData(h *Hash) (Value, error) {
	allStr := true
	hasReserved := false
	for _, e := range h.entries {
		k, ok := e.Key.(string)
		if !ok {
			allStr = false
			break
		}
		if k == keyPType || k == keyPValue {
			hasReserved = true
		}
	}
	if allStr && !hasReserved {
		out := make([]HashEntry, len(h.entries))
		for i, e := range h.entries {
			d, err := ToData(e.Value)
			if err != nil {
				return nil, err
			}
			out[i] = HashEntry{Key: e.Key, Value: d}
		}
		return &Hash{entries: out}, nil
	}
	// Non-string (or reserved) keys: encode as an alternating k,v array.
	arr := make([]Value, 0, len(h.entries)*2)
	for _, e := range h.entries {
		kd, err := ToData(e.Key)
		if err != nil {
			return nil, err
		}
		vd, err := ToData(e.Value)
		if err != nil {
			return nil, err
		}
		arr = append(arr, kd, vd)
	}
	return tagged("Hash", arr), nil
}

// FromData reconstructs a Pcore value from its rich-data representation as
// produced by [ToData].
func FromData(v Value) (Value, error) {
	switch x := canon(v).(type) {
	case undefValue:
		return Undef, nil
	case bool, int64, float64, string:
		return x, nil
	case []Value:
		out := make([]Value, len(x))
		for i, e := range x {
			r, err := FromData(e)
			if err != nil {
				return nil, err
			}
			out[i] = r
		}
		return out, nil
	case *Hash:
		return hashFromData(x)
	default:
		return nil, fmt.Errorf("pcore: cannot deserialize value of type %T", v)
	}
}

func hashFromData(h *Hash) (Value, error) {
	pt, tagged := h.Get(keyPType)
	if !tagged {
		out := make([]HashEntry, len(h.entries))
		for i, e := range h.entries {
			r, err := FromData(e.Value)
			if err != nil {
				return nil, err
			}
			out[i] = HashEntry{Key: e.Key, Value: r}
		}
		return &Hash{entries: out}, nil
	}
	ptype, ok := pt.(string)
	if !ok {
		return nil, fmt.Errorf("pcore: %s must be a string", keyPType)
	}
	pv, _ := h.Get(keyPValue)
	return reconstruct(ptype, pv)
}

func reconstruct(ptype string, pv Value) (Value, error) {
	switch ptype {
	case "Default":
		return Default, nil
	case "Sensitive":
		return NewSensitive(Undef), nil // redacted on the way out
	case "Regexp":
		s, err := needString(ptype, pv)
		if err != nil {
			return nil, err
		}
		return NewRegexp(s)
	case "Binary":
		s, err := needString(ptype, pv)
		if err != nil {
			return nil, err
		}
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("pcore: invalid Binary payload: %w", err)
		}
		return NewBinary(b), nil
	case "Timestamp":
		s, err := needString(ptype, pv)
		if err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil, fmt.Errorf("pcore: invalid Timestamp: %w", err)
		}
		return NewTimestamp(t), nil
	case "Timespan":
		s, err := needString(ptype, pv)
		if err != nil {
			return nil, err
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("pcore: invalid Timespan: %w", err)
		}
		return NewTimespan(d), nil
	case "SemVer":
		s, err := needString(ptype, pv)
		if err != nil {
			return nil, err
		}
		return NewSemVer(s)
	case "SemVerRange":
		s, err := needString(ptype, pv)
		if err != nil {
			return nil, err
		}
		return NewSemVerRange(s)
	case "URI":
		s, err := needString(ptype, pv)
		if err != nil {
			return nil, err
		}
		return NewURI(s), nil
	case "Type":
		s, err := needString(ptype, pv)
		if err != nil {
			return nil, err
		}
		return Parse(s)
	case "Hash":
		arr, ok := canon(pv).([]Value)
		if !ok || len(arr)%2 != 0 {
			return nil, fmt.Errorf("pcore: Hash payload must be an even-length array")
		}
		entries := make([]HashEntry, 0, len(arr)/2)
		for i := 0; i < len(arr); i += 2 {
			k, err := FromData(arr[i])
			if err != nil {
				return nil, err
			}
			val, err := FromData(arr[i+1])
			if err != nil {
				return nil, err
			}
			entries = append(entries, HashEntry{Key: k, Value: val})
		}
		return &Hash{entries: entries}, nil
	default:
		return nil, fmt.Errorf("pcore: unknown %s %q", keyPType, ptype)
	}
}

func needString(ptype string, pv Value) (string, error) {
	s, ok := pv.(string)
	if !ok {
		return "", fmt.Errorf("pcore: %s payload must be a string", ptype)
	}
	return s, nil
}

// Package pcore is a pure-Go (cgo-free) reimplementation of Puppet's Pcore type
// system — the data-type and value model that underpins Puppet, Hiera and
// Facter values.
//
// It provides four things a Puppet-family tool needs:
//
//   - a Type model covering the full Pcore type calculus — scalar, collection,
//     abstract, rich-data and nominal types, recursive type aliases and TypeSet;
//   - a [Parse] function turning a Pcore type expression such as
//     "Variant[Integer[0,10], Enum['a','b']]" into a [Type], plus a [Loader]
//     type environment for named/recursive aliases and TypeSet resolution;
//   - a value model — plain Go values for scalars and collections plus wrappers
//     ([Undef], [Default], [Sensitive], [Regexp], [Binary], [Timestamp],
//     [Timespan], [SemVer], [SemVerRange], [URI], [ObjectValue] and friends);
//   - the load-bearing operations: [IsInstance] (value ∈ type),
//     [IsAssignable] (subtype), [Infer] (a value's most specific type),
//     [Generalize], [CommonType], and the rich-data [ToData]/[FromData]
//     serialization.
//
// Every [Type]'s String method round-trips through [Type]: for any t,
// Type(t.String()) equals t.
//
// The names and semantics deliberately track Puppet's Puppet::Pops::Types so
// the package is a drop-in for Puppet type expressions.
package pcore

// Value is any Pcore value. Scalars are represented by their natural Go types
// (bool, int64, float64, string); arrays by []Value; hashes by *Hash (or a
// map[string]Value for the common string-keyed case); and the remaining kinds
// by the wrapper types in this package ([Undef], [Default], [Sensitive],
// [Regexp], [Binary], [Timestamp], [Timespan]). A Go nil is treated as Undef.
type Value = any

// Type is a Pcore type. The set of implementations is closed to this package;
// use [Type] (the parser) or the exported constructors to obtain one.
type Type interface {
	// Name is the unparameterized Pcore type name, e.g. "Integer".
	Name() string
	// String is the full, parameterized, canonical form. It round-trips:
	// Type(t.String()) reproduces t.
	String() string

	// isInstance answers whether v is an instance of the receiver. g carries the
	// co-inductive recursion guard used to terminate recursive type aliases.
	isInstance(v Value, g *guard) bool
	// isAssignable answers whether other is a subtype of the receiver. g carries
	// the co-inductive recursion guard used to terminate recursive type aliases.
	isAssignable(other Type, g *guard) bool
}

// IsInstance reports whether v is an instance of t.
func IsInstance(t Type, v Value) bool { return t.isInstance(canon(v), newGuard()) }

// IsAssignable reports whether b is a subtype of a, i.e. every instance of b is
// an instance of a. It is the core lattice operation.
func IsAssignable(a, b Type) bool { return assignable(a, b) }

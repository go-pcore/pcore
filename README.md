<p align="center"><img src="https://raw.githubusercontent.com/go-pcore/brand/main/social/go-pcore.png" alt="go-pcore" width="640"></p>

<h1 align="center">go-pcore</h1>
<p align="center"><strong>Puppet's Pcore type system in pure Go — the type calculus, parser, value model and assignability lattice, no cgo.</strong></p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/go-pcore/pcore"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/go-pcore/pcore.svg"></a>
  <a href="https://go-pcore.github.io/docs/"><img alt="Docs" src="https://img.shields.io/badge/docs-mkdocs--material-FBBF24?style=flat-square"></a>
  <a href="https://github.com/go-pcore/pcore/blob/main/LICENSE"><img alt="License: BSD-3-Clause" src="https://img.shields.io/badge/license-BSD--3--Clause-blue?style=flat-square"></a>
  <img alt="Go 1.26.4+" src="https://img.shields.io/badge/go-1.26.4%2B-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="Coverage 100%" src="https://img.shields.io/badge/coverage-100%25-1a7f37?style=flat-square">
</p>

---

`go-pcore/pcore` is a **pure-Go (no cgo) reimplementation of [Pcore](https://github.com/puppetlabs/puppet-specifications/blob/master/language/types_values_variables.md)**,
the data-type and value model that underpins **Puppet**, **Hiera** and **Facter**
values. It gives a Go program the Puppet type calculus: a type model, a type
**parser**, a value model, and the load-bearing operations — instance-of,
assignability (subtyping), inference and rich-data serialization — with names and
semantics that track Puppet's `Puppet::Pops::Types` so it is a drop-in for Puppet
type expressions.

```go
t, _ := pcore.Parse("Variant[Integer[0,10], Enum['a','b']]")

pcore.IsInstance(t, int64(7))          // true
pcore.IsInstance(t, "a")               // true
pcore.IsInstance(t, int64(99))         // false

data, _ := pcore.Parse("Data")
arr, _ := pcore.Parse("Array[Integer]")
pcore.IsAssignable(data, arr)                    // true — Array[Integer] <: Data
pcore.Infer([]pcore.Value{int64(1), int64(2)})   // Array[Integer[1, 2], 2, 2]
t.String()                                       // round-trips through Parse
```

## What it provides

| Area | API |
|------|-----|
| **Parse** a type expression | `Parse(string) (Type, error)` |
| **Instance check** (value ∈ type) | `IsInstance(t Type, v Value) bool` |
| **Assignability** (subtype lattice) | `IsAssignable(a, b Type) bool` |
| **Infer** a value's most specific type | `Infer(v Value) Type` |
| **Generalize** / **CommonType** | `Generalize(Type) Type`, `CommonType(a, b Type) Type` |
| **Rich-data serialization** | `ToData(Value) (Value, error)`, `FromData(Value) (Value, error)` |
| **Canonical string form** | `Type.String()` — round-trips through `Parse` |

## The type calculus

The full Pcore type set is implemented:

- **Scalar:** `Integer[min,max]`, `Float[min,max]`, `Numeric`, `String[min,max]`,
  `Boolean`, `Undef`, `Default`, `Scalar`, `ScalarData`, `Data`, `RichData`,
  `RichDataKey`, `Any`.
- **Collection:** `Array[T,min,max]`, `Hash[K,V,min,max]`, `Tuple[...]`,
  `Struct[{k=>V}]`, `Collection[min,max]`.
- **Abstract:** `Variant[...]`, `Optional[T]`, `NotUndef[T]`, `Enum[...]`,
  `Pattern[/re/]`, `Regexp`, `Type[T]`, `Sensitive[T]`, `Init[T, args...]`,
  `Iterable[T]`, `Iterator[T]`, `Callable[params..., block]`.
- **Rich data:** `Timestamp[from,to]`, `Timespan[from,to]`, `Binary`,
  `SemVer[ranges...]`, `SemVerRange`, `Runtime['go', name]`, `URI[scheme]`,
  `Error[kind, issue_code]`.
- **Nominal / named:** `Object[{name=>..., parent=>..., attributes=>{...}}]`,
  **type aliases** (`type X = <expr>`, including recursive), and
  `TypeSet[{name=>..., version=>..., types=>{...}, references=>{...}}]`.

### Type aliases (recursive) and TypeSet

Aliases live in a [Loader] type environment that resolves forward and recursive
references transparently through `Parse`/`IsInstance`/`IsAssignable`/`Infer`:

```go
l := pcore.NewLoader()
l.Declare("type Tree = Hash[String, Variant[Tree, Integer]]")
tree, _ := l.Parse("Tree")

pcore.IsInstance(tree, map[string]pcore.Value{
    "a": int64(1),
    "b": map[string]pcore.Value{"c": int64(2)},
}) // true

ts, _ := l.Parse(`TypeSet[{
    name => 'MyMod::Types', version => '1.0.0',
    types => { Age => Integer[0,130], Person => Struct[{'age' => Age}] },
}]`)
```

## Value model

Scalars are plain Go values (`bool`, `int64`, `float64`, `string`); arrays are
`[]pcore.Value`; hashes are `*pcore.Hash` (or a `map[string]Value`); and the rest
are wrapper types — `Undef`, `Default`, `Sensitive` (redacts on `String()` and on
serialization), `Regexp`, `Binary`, `Timestamp`, `Timespan`.

## Consumers

`go-pcore` is the foundational type layer for **[go-puppet](https://github.com/go-puppet)**
(the Puppet DSL evaluator) and for **go-ruby-puppet**, which marshals
`rbgo.Value ↔ pcore.Value` across the rich-data protocol.

## Principles

- **Pure Go, zero cgo.** Cross-compiles and embeds anywhere; a static binary by
  default. CI is green across the six 64-bit Go targets (amd64, arm64, riscv64,
  loong64, ppc64le, s390x).
- **Faithful to Pcore.** Type names, the grammar, the assignability rules and the
  rich-data protocol track Puppet's specification.
- **Round-trippable.** Every `Type.String()` parses back through `Parse`.
- **100% test coverage**, enforced as a CI gate, including every parse-error,
  assignability arm and serialization path.

## Status

**v0.2 — full Pcore type calculus.** The complete Puppet `Puppet::Pops::Types`
set is implemented: type model, parser, value model, `IsInstance`/
`IsAssignable`/`Infer`/`Generalize`/`CommonType`, rich-data serialization,
**recursive type aliases** and **`TypeSet`** via a [Loader] type environment,
`Timestamp`/`Timespan` ranges, `SemVer`/`SemVerRange`, `Init`, `Object`,
`RichData`, `Runtime`, `URI`, `Iterable`/`Iterator`, `Error` and `Callable`.
100% test coverage (enforced), `gofmt` + `go vet` clean, CI green across all six
64-bit Go arches. See [BENCHMARKS.md](BENCHMARKS.md) for the perf harness and the
differential-vs-MRI-Puppet methodology.

[Loader]: https://pkg.go.dev/github.com/go-pcore/pcore#Loader

BSD-3-Clause.

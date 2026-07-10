<!-- SPDX-License-Identifier: BSD-3-Clause -->
# Benchmarks & differential methodology

`go-pcore` must be **at least as fast as the reference** — MRI Puppet's
`Puppet::Pops::Types` — on the hot paths: `Parse`, `IsInstance`, `IsAssignable`
and `Infer`.

## Go benchmarks

```sh
GOWORK=off go test -run '^$' -bench=. -benchmem ./...
```

Representative run (Apple M4 Max, `go1.26.4`, `darwin/arm64`; numbers are
illustrative — re-run on your target):

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `Parse` (10 mixed exprs) | ~7200 | 20040 | 207 |
| `ParseAlias` (declare+resolve recursive `Tree`) | ~984 | 2440 | 29 |
| `IsInstance` (Struct over a hash) | ~83 | 8 | 1 |
| `IsInstanceRecursiveAlias` (`Tree` over a nested hash) | ~1159 | 1016 | 29 |
| `IsAssignable` (Hash ⊇ Struct) | ~610 | 544 | 27 |
| `Infer` (mixed array) | ~755 | 624 | 40 |

`IsInstance` on a concrete type is a single-allocation, ~80 ns operation: the
canonicalisation of the value plus a direct structural walk. There is no
per-check parsing — parse once, check many.

## Differential oracle vs MRI Puppet

The reference implementation is the Ruby `puppet` gem. The harness parses the
same expression in both engines and compares `Parse`/`instance?`/`assignable?`
and the round-tripped `to_s`.

### Environment

MRI Puppet needs a compatible Ruby. Puppet 7 depends on `multi_json`, which
requires **Ruby 3.2+**; Puppet 6 supports Ruby 2.4–2.7. The macOS system Ruby
(2.6) and a bleeding-edge Homebrew Ruby (4.0) are **both** incompatible with an
installable Puppet, so run the oracle inside a Tart VM (or any host) with
`ruby 3.2` + `puppet 7`:

```sh
# In a debian Tart VM (see MEMORY: "use Tart VMs")
sudo apt-get install -y ruby ruby-dev build-essential   # ruby 3.x
gem install --no-document puppet -v '~> 7'
ruby oracle/pcore_oracle.rb > oracle/expected.json
```

### Ruby oracle (`oracle/pcore_oracle.rb`)

```ruby
require 'puppet'
require 'puppet/pops'
require 'json'

P  = Puppet::Pops::Types::TypeParser.singleton
TC = Puppet::Pops::Types::TypeCalculator

exprs = File.readlines('oracle/exprs.txt', chomp: true)
out = exprs.map do |e|
  t = P.parse(e)
  { expr: e, to_s: t.to_s }
end
puts JSON.pretty_generate(out)
```

`oracle/exprs.txt` holds the expression corpus (the same strings used in the Go
differential tests). The Go side (`Parse(e).String()`) is compared against the
`to_s` column; `instance?`/`assignable?` pairs are compared the same way over a
value corpus. Because `to_s` formatting differs cosmetically between engines
(quoting, spacing), the comparison normalises whitespace before asserting.

### What is already differential

The Go test suite encodes Puppet Pops semantics directly (spec examples for
aliases, recursive aliases, `TypeSet` resolution, `SemVer`/`SemVerRange`,
`Timestamp`/`Timespan` ranges, `Init`, `Object`, `RichData`, `Callable`,
`Iterable`/`Iterator`, `Runtime`, `URI`, `Error`) as golden expectations, so it
runs as a self-contained differential suite even without a live Ruby oracle.

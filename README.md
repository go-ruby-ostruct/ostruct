<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-ostruct/brand/main/social/go-ruby-ostruct-ostruct.png" alt="go-ruby-ostruct/ostruct" width="720"></p>

# ostruct — go-ruby-ostruct

[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![CI](https://github.com/go-ruby-ostruct/ostruct/actions/workflows/ci.yml/badge.svg)](https://github.com/go-ruby-ostruct/ostruct/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](.github/workflows/ci.yml)

**A pure-Go (no cgo), MRI-4.0.5-faithful core for Ruby's [`OpenStruct`](https://docs.ruby-lang.org/en/master/OpenStruct.html)** (`require "ostruct"`).

It implements the data structure underneath `OpenStruct`: an **ordered attribute
table** keyed by `Symbol`, with the accessors, conversions, comparison, and
inspection MRI exposes — `[]`/`[]=`, `to_h`, `each_pair`, `dig`, `delete_field`,
`==`/`eql?`, and `inspect`/`to_s`. It matches MRI **byte-for-byte** on the
`inspect` format, `to_h` insertion ordering, and `delete_field` semantics.

It is the `OpenStruct` backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime.

## What stays in the host runtime

`OpenStruct`'s defining feature — turning *any* method name into an attribute
read or write — is dynamic and lives in the host (rbgo): `method_missing`,
`define_method`, and `respond_to_missing?`. That glue is implemented **in terms
of this table**: a reader call becomes [`Get`](ostruct.go), a `name=` call
becomes [`Set`](ostruct.go), and `respond_to_missing?` consults
[`RespondToField`](ostruct.go). This package owns the table and its MRI-faithful
behavior; the host owns the dynamic dispatch.

## API

| Go | Ruby |
| --- | --- |
| `New(pairs...)` | `OpenStruct.new(hash)` |
| `Get(name)` / `Set(name, v)` | reader / `name=` writer (the `method_missing` target) |
| `Index(name)` / `SetIndex(name, v)` | `[]` / `[]=` |
| `ToH()` | `to_h` (Symbol keys, insertion order) |
| `EachPair(fn)` | `each_pair` |
| `Members()` | `members` |
| `Dig(keys...)` | `dig(*keys)` |
| `RespondToField(name)` | the table half of `respond_to_missing?` |
| `DeleteField(name)` | `delete_field` (returns old value; `NameError` if absent) |
| `Equal(o)` / `Eql(o)` | `==` / `eql?` |
| `Inspect()` / `String()` | `inspect` / `to_s` |

Keys accept a `Symbol` or `string` (interned via `ToSym`). Values are held as
opaque `any`; `Inspect` and `Dig` route through the `Inspector` and `Digger`
interfaces so the host's Ruby values render and dig exactly as in MRI, with a
built-in renderer covering the common scalar/collection shapes for
deterministic, ruby-free testing.

## MRI-faithful samples

```text
OpenStruct.new(name: "John", age: 70).inspect  #=> #<OpenStruct name="John", age=70>
OpenStruct.new.inspect                          #=> #<OpenStruct>
OpenStruct.new(b: 1, a: 2, c: 3).to_h.keys      #=> [:b, :a, :c]   (insertion order)
OpenStruct.new(a: {b: {c: 1}}).dig(:a, :b, :c)  #=> 1
o.delete_field(:age)                            #=> 70             (old value)
OpenStruct.new.delete_field(:x)                 #=> NameError: no field 'x' in #<OpenStruct>
OpenStruct.new(name: "John") == OpenStruct.new(name: "John")  #=> true
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-ostruct/ostruct authors.

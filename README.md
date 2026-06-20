# goal — the Go Augmented Language

A correctness-oriented dialect of Go that transpiles to plain Go. goal exists to give
AI coding agents (and humans) **fast, located, machine-checkable feedback** — biasing
every feature toward turning *silent runtime failures* and *human-judgment calls* into
*compiler or test errors* the model can act on. It stays as close to Go's shape as
possible so the model's existing fluency carries over; every divergence earns its keep.

```goal
enum Shape {
    Circle { r: float64 }
    Rect   { w: float64, h: float64 }
}

func area(s Shape) float64 {
    return match s {
        Shape.Circle(c) => 3.14159 * c.r * c.r
        Shape.Rect(r)   => r.w * r.h
    }
}
```

lowers to a sealed-interface sum type and an exhaustive type switch — idiomatic Go the
compiler already understands.

## Quick start

Requires Go 1.26+.

```bash
# transpile a .goal program to Go (stdout)
go run ./cmd/goalc path/to/file.goal

# emit the doctest sidecar (_test.go) extracted from /// comments
go run ./cmd/goalc -test path/to/file.goal

# read from stdin
cat file.goal | go run ./cmd/goalc -

# build the CLI
go build -o bin/goalc ./cmd/goalc
```

The output is gofmt-formatted Go. Pipe it to a `.go` file in a real package and it
compiles as-is.

## The 12 features

Each transpiles to idiomatic Go; the static *guarantee* behind each is the job of the
checker (not yet built — see [Status](#status)).

| # | Feature | Surface | Lowers to |
|---|---------|---------|-----------|
| 01 | enums | `enum E { A; B { x: T } }` | sealed interface + per-variant struct + marker |
| 02 | match | `match e { E.A => … }` | exhaustive type switch (panicking default) |
| 03 | Result (open-E) | `Result[T, error]` | native `(T, error)` tuple |
| 04 | Option | `Option[T]` | `*T` (nil = None) |
| 05 | `?` propagation | `x := f()?` | unwrap-or-early-return |
| 06 | Result (closed-E) | `Result[T, MyErr]` | generic sum `Ok[T,E]`/`Err[T,E]` + From-conversion |
| 07 | implements | `type T struct implements I { … }` | compile-time assertion `var _ I = T{}` |
| 08 | no-zero-value | `T{a: x, ...defaults}` | explicit per-field zero expansion |
| 09 | pure | `pure func f()` | plain `func` (marker erased) |
| 10 | assert | `assert cond, "msg", args` | runtime `if !(cond) { panic(...) }` |
| 11 | doctests | `/// >>> f(2)` / `/// 4` | a generated `_test.go` |
| 12 | derive-convert | `derive func g(s S) T` | field-by-field conversion via a `from func` registry |

Locked conventions (see `DECISIONS.md`): qualified construction (`Status.Active(…)`,
`Result.Ok`, `Option.Some`), brace-named payloads, newline-separated variants/arms,
conventional names verbatim (`Ok`/`Err`/`Some`/`None`/`=>`/`_`/`?`), modifiers before
`func` (`from`/`pure`/`derive`), and `__gop_`-prefixed synthesized temporaries.

## The unified front-end

`cmd/goalc` drives one pipeline that transpiles a program using **any combination** of
the 12 features — there is no per-feature tool to pick. The pipeline is an ordered list
of source→source passes; the driver threads the source string through them and formats
**once** at the end.

```
internal/
  scan/       lexer (byte-offset recovery), splice model, balanced-delimiter and
              structural helpers — the shared low-level machinery
  analyze/    name-keyed tables built ONCE from the original source, read-only to passes:
              function signatures (open/closed Result mode + T/E), enums, sealed
              interfaces, structs, type decls, the From-conversion registry
  pass/       one file per construct; each Run(src, *Tables) (string, error) re-lexes,
              splices its construct, and formats nothing
  pipeline/   ordered Passes + driver; returns Output{Go, Test}
cmd/goalc/    the CLI
testdata/     multi-feature .goal/.go.expected programs (the real proof)
features/     the 12 standalone reference transpilers — per-feature source of truth
```

Pass order:

```
pure → implements → defaults → result → option → question → closed → derive → assert → match → enums
```

then format once. Doctests are extracted from the original source as a side output.

### How it works (and why)

Passes splice bytes, so byte offsets shift between them. Two rules make that safe:

- **Name-keyed tables, never offsets.** Every cross-pass fact is keyed by symbol name
  (function, type), built once from the original source, and survives re-lexing. Each
  pass re-lexes the current source and rebuilds the spans it needs.
- **Format once.** No pass calls `go/format`; only the driver does, at the end. An
  intermediate source need only be *lexable*, not parseable.

Several constructs share surface syntax but lower differently. Rather than have passes
fight over them, each is **partitioned by a table fact**:

- `match` is claimed by the open-Result, Option, closed-Result, or enum pass, chosen by
  the arm qualifier and the scrutinee's mode.
- each interface in a struct's `implements` clause becomes a marker method when it is a
  sealed interface, a compile-time assertion otherwise.
- `?` is handled open/Option in one pass and closed-E in another; a shared
  enclosing-function lookup keeps them from both claiming the same `?`.
- `from func` is one registry, shared by closed-E `?` (06) and `derive func` (12).

The headline case the architecture buys you: **open-E and closed-E `Result` in the same
file** — native `(T, error)` tuples beside the `Ok[T,E]`/`Err[T,E]` sum encoding, with a
`from func` conversion firing across error types. No single-feature transpiler can
produce that; the unified pipeline does (`testdata/open_closed_mix.goal`).

## Status

**Front-end: complete.** All 12 features compose. Every reference example and a suite of
multi-feature `testdata/` programs round-trip to correct, independently-compiling Go.

**Checker: not started** — the next major workstream. The front-end lowers proven-valid
input and *defers* (with a located error) anything it cannot resolve, but it emits no
static diagnostics yet. The checker is where each feature's guarantee lands:
exhaustiveness (02), must-use (03), field-completeness (08), `implements` satisfaction
(07), `pure` effect-freedom (09), static asserts (10), conversion totality (12),
closedness & From-totality (06). See `NEXT-SESSION.md`.

## Tests

```bash
go vet ./...
go test -count=1 ./...
```

The `internal/pipeline` suite runs three checks: the multi-feature `testdata/` programs,
every single-feature reference example re-run through the unified pipeline (regression
locks), and the doctest side output. Because golden files are generated *from* the tool,
the real verification is that generated programs **independently compile** in a throwaway
module — and that runtime-preserved output (assert, doctests, derive) actually **runs**.

## Project files

- `goal-design-spec.md` — the language design spec (read-only; covers features 01–11).
- `DECISIONS.md` — the choice / assumption / refusal ledger, §01–§12.
- `TODO.md` — per-feature status and artifact pointers.
- `NEXT-SESSION.md` — current handoff: front-end architecture as built + the checker plan.
- `FEATURE-AUDIT-PROMPT.md` — the per-feature audit loop, for adding any future feature.
- `features/NN-*/` — the 12 standalone reference transpilers (each with its own examples
  and `DECISIONS.md`); the unified front-end reuses their logic, re-keyed by name.

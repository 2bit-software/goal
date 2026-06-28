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

Requires Go 1.26+. There are two binaries: **`goal`**, the umbrella CLI that drives whole
programs through the Go toolchain, and **`goalc`**, the single-file transpiler primitive.

```bash
# whole-program driver (discovers .goal packages under a path; default ".")
go run ./cmd/goal check ./app        # run the static checker; prints "ok" or located diagnostics
go run ./cmd/goal run   ./app        # transpile + `go run` the sole `package main`
go run ./cmd/goal build ./app        # transpile + `go build ./...`  (ephemeral, via -overlay)
go run ./cmd/goal build --emit ./app # ALSO write generated .go (+ _test.go) beside each .goal
go run ./cmd/goal fix ./app          # rewrite plain-Go patterns into idiomatic goal (stdout)
go run ./cmd/goal fix -inplace ./app # ... and write the changes back to each file

# single-file primitive (transpiles one file to stdout)
go run ./cmd/goalc path/to/file.goal       # lowered Go to stdout (checker runs first)
go run ./cmd/goalc -test path/to/file.goal # the doctest sidecar (_test.go) instead
go run ./cmd/goalc -nocheck file.goal      # skip the checker, just lower
cat file.goal | go run ./cmd/goalc -       # read from stdin

# build the CLIs
go build -o bin/goal  ./cmd/goal
go build -o bin/goalc ./cmd/goalc
```

`goal build`/`run` are **ephemeral** by default: the generated Go is mapped into the module
with `go build -overlay`, so nothing is written to your source tree and module/stdlib
imports still resolve. `--emit` instead writes the generated `.go` (and doctest `_test.go`)
to disk, which is how you run doctests:

```bash
go run ./cmd/goal build --emit ./app && go test -count=1 ./app/...
```

`goalc` output is gofmt-formatted Go; pipe it into a `.go` file in a real package and it
compiles as-is. **A from-scratch agent guide lives in
[`AI-KNOWLEDGE-BOOTSTRAP.md`](AI-KNOWLEDGE-BOOTSTRAP.md).**

`goal fix` runs the lowering passes *in reverse*: since goal is a superset of Go, a `.goal`
file can be written in plain-Go style, and `fix` rewrites those patterns into idiomatic
goal. The keystone is collapsing manual error propagation — a function written as a
`(T, error)` tuple with `if err != nil { return zero, err }` blocks becomes one returning
`Result[T, error]` whose body propagates with `?`:

```goal
func load(p string) ([]byte, error) {          func load(p string) Result[[]byte, error] {
    f, err := os.ReadFile(p)                        f := os.ReadFile(p)?
    if err != nil {              ── goal fix ──▶     return Result.Ok(f)
        return nil, err                          }
    }
    return f, nil
}
```

Like `gofmt`, it prints to stdout by default and writes back with `-inplace`. It edits only
what it can prove equivalent: anything ambiguous (a wrapped error, a multi-value return, a
`switch` over an enum) is left untouched and reported to stderr as a suggestion, with
`goal check` remaining the authority on correctness.

## The 11 features

Each transpiles to idiomatic Go; the static *guarantee* behind each is enforced by the
checker (see [Status](#status)).

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
| 10 | assert | `assert cond, "msg", args` | runtime `if !(cond) { panic(...) }` |
| 11 | doctests | `/// >>> f(2)` / `/// 4` | a generated `_test.go` |
| 12 | derive-convert | `derive func g(s S) T` | field-by-field conversion via a `from func` registry |

Locked conventions (see `DECISIONS.md`): qualified construction (`Status.Active(…)`,
`Result.Ok`, `Option.Some`), brace-named payloads, newline-separated variants/arms,
conventional names verbatim (`Ok`/`Err`/`Some`/`None`/`=>`/`_`/`?`), modifiers before
`func` (`from`/`derive`), and `__goal_`-prefixed synthesized temporaries.

## The unified front-end

`cmd/goalc` drives one pipeline that transpiles a program using **any combination** of
the 11 features — there is no per-feature tool to pick. The pipeline is an ordered list
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
features/     the 11 standalone reference transpilers — per-feature source of truth
                (audited-but-cut features are frozen under features/_cut/)
```

Pass order:

```
implements → defaults → result → option → question → closed → derive → assert → match → enums
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

**Front-end: complete.** All 11 features compose. Every reference example and a suite of
multi-feature `testdata/` programs round-trip to correct, independently-compiling Go.

**Checker: implemented and on by default.** Each feature's guarantee now lands as a located
`file:line:col: error: [code] message` diagnostic, emitted by `goal check` (and gating
`goal build`/`run` and `goalc` unless `-nocheck` is given). It runs in two stages: a
**lexical** stage (`internal/check`) over the original source, and a **typed depth** stage
(`internal/typecheck`) that loads the lowered Go into `go/types` to answer what the lexical
stage had to defer; the type-backed finding wins when both flag the same construct.
Coverage spans exhaustiveness (02), must-use / dropped `Result` (03, 06), field-completeness
(08), `implements` satisfaction and method-signature match (07), always-true/false `assert`
(10), and conversion totality & From-completeness (12). See `testdata/check/` for the input
programs each check is expected to flag.

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
- `REWRITE-ARCHITECTURE.md` — the AST front-end architecture as built (lexer → parser → AST → sema → backend).
- `FEATURE-AUDIT-PROMPT.md` — the per-feature audit loop, for adding any future feature.
- `features/NN-*/` — the 12 standalone reference transpilers (each with its own examples
  and `DECISIONS.md`); the unified front-end reuses their logic, re-keyed by name.

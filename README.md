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
go run ./cmd/goal check ./app             # run the static checker; prints "ok" or located diagnostics
go run ./cmd/goal run   ./app             # transpile + `go run` the sole `package main`
go run ./cmd/goal run --engine=interp x.goal  # run a single .goal file under the goscript tree-walking interpreter
go run ./cmd/goal build ./app             # transpile + `go build ./...`  (ephemeral, via -overlay)
go run ./cmd/goal test  ./app             # transpile + `go test` the doctests (ephemeral, via -overlay)
go run ./cmd/goal build --emit ./app      # ALSO write generated .go (+ _test.go) beside each .goal
go run ./cmd/goal fix ./app               # rewrite plain-Go patterns into idiomatic goal (stdout)
go run ./cmd/goal fix -inplace ./app      # ... and write the changes back to each file
go run ./cmd/goal fmt ./app               # format .goal source into the canonical layout (stdout; -w writes back)
go run ./cmd/goal ai                      # print the AI bootstrap guide (how to write goal) to stdout
go run ./cmd/goal lsp                     # run the language server (editor diagnostics) over stdio

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

Locked conventions (see `docs/DECISIONS.md`): qualified construction (`Status.Active(…)`,
`Result.Ok`, `Option.Some`), brace-named payloads, newline-separated variants/arms,
conventional names verbatim (`Ok`/`Err`/`Some`/`None`/`=>`/`_`/`?`), modifiers before
`func` (`from`/`derive`), and `__goal_`-prefixed synthesized temporaries.

## The AST front-end

One engine-neutral front-end, pluggable back-ends. Source is lexed to tokens, parsed to a
real AST with first-class positions, checked over that AST, and then handed to a back-end.
There is no per-feature tool and no source→source splicing: a single front-end handles
**any combination** of the 11 features, and the Go back-end formats the emitted Go once at
the end. (`docs/archive/REWRITE-ARCHITECTURE.md` is the canonical, as-built architecture reference.)

```
source ─▶ token ─▶ lexer ─▶ parser ─▶ AST ─▶ sema (check) ─┬─▶ backend/go    (transpile → Go)   ← today's product
                                                           └─▶ backend/interp (tree-walk)        ← goscript runtime
tools (lsp, fix, fmt, guide) all consume the same AST + sema results
```

The compiler and its tooling live under `internal/`:

| Package | Responsibility |
|---------|----------------|
| `internal/token` | token kinds + `Pos` (offset/line/col); positions are first-class on the AST |
| `internal/lexer` | source → tokens; knows `?`, `=>`, `field:`, `...`, `///`, contextual keywords; reports located lex errors |
| `internal/ast` | node types + `Visitor`/`Walk` for declarations, statements, expressions, and patterns |
| `internal/parser` | tokens → AST via Go-shaped recursive descent + Pratt expressions |
| `internal/sema` | correctness checks over the AST: exhaustiveness, no-zero-value, must-use, implements, `?`-arity |
| `internal/typecheck` | typed depth stage — loads the lowered Go into `go/types` for full Go+ semantics |
| `internal/backend` | typed AST → Go source (the transpiler); doctest sidecar emission |
| `internal/interp` | tree-walking evaluator (`run --engine=interp`), the goscript runtime |
| `internal/pipeline` | driver: file/package discovery orchestration, source maps (`//line` directives), formatter selection |
| `internal/project` | `.goal` package discovery on disk |
| `internal/fix` | rewrite plain-Go patterns into idiomatic goal (`goal fix`) |
| `internal/goalfmt` | canonical, comment-preserving `.goal` formatter (`goal fmt`) |
| `internal/lsp` | language server: diagnostics, code actions |
| `internal/guide` | the AI bootstrap guide + command catalog (`goal ai`) |
| `internal/cap` | capability/effect model (shape now, surfaced later) |
| `internal/corpus`, `internal/backendtest` | plain-Go test infrastructure (the golden corpus + behavioral gates) |

The two CLIs sit above this: `cmd/goal` is the whole-program umbrella driver and
`cmd/goalc` is the single-file transpiler primitive. Doctests are extracted from the AST as
a side output and emitted as a `_test.go` sidecar.

The headline case the architecture buys you: **open-E and closed-E `Result` in the same
file** — native `(T, error)` tuples beside the `Ok[T,E]`/`Err[T,E]` sum encoding, with a
`from func` conversion firing across error types (`testdata/open_closed_mix.goal`).

## Status

> For the empirically-verified, current state of every construct (what's complete, what's
> deferred to real types, the deliberate design boundaries, and what's genuinely open), see
> [`docs/STATUS.md`](docs/STATUS.md) — the authoritative snapshot.

**Front-end: complete.** All 11 features compose. Every reference example and a suite of
multi-feature `testdata/` programs round-trip to correct, independently-compiling Go.

**Checker: implemented and on by default.** Each feature's guarantee now lands as a located
`file:line:col: error: [code] message` diagnostic, emitted by `goal check` (and gating
`goal build`/`run` and `goalc` unless `-nocheck` is given). It runs in two stages: an
**AST** stage (`internal/sema`) over the parsed goal AST, and a **typed depth** stage
(`internal/typecheck`) that loads the lowered Go into `go/types` to answer what the AST
stage had to defer; the type-backed finding wins when both flag the same construct.
Coverage spans exhaustiveness (02), must-use / dropped `Result` (03, 06), field-completeness
(08), `implements` satisfaction and method-signature match (07), always-true/false `assert`
(10), and conversion totality & From-completeness (12). See `testdata/check/` for the input
programs each check is expected to flag.

## The self-hosted compiler

goal is written in goal. Every compiler package under `internal/` is colocated
**canonical goal source (`<file>.goal`)** plus its **committed, generated Go
(`<file>.go`)**: the `.goal` is the single source of truth, and the `.go` is a
build artifact the goal toolchain emits beside it. `task generate` regenerates
the `.go` from the `.goal`, and `task verify-generated` is the drift gate that
fails if any committed `.go` diverges from a fresh emit. `go build` reads the
committed `.go`; the goal front-end reads the `.goal`. The hand-written Go
transpiler that bootstrapped the language is gone — the corpus behavioral tier
and the `task fixpoint` byte-identity check are the correctness gates. Only the
test/dev infra (the corpus, `byexample`, and the `internal/selfhost` build
harness) stays plain Go.

## Tests

```bash
task check      # go vet + verify-generated (drift gate) + the whole test suite
task fixpoint   # rebuild the compiler from its own .goal and assert byte-identical output
```

The `internal/corpus` package drives the golden corpus, organized in manifest tiers:
**transpile** (`.goal` → `.go.expected` programs), **check** (`testdata/check/` inputs and
the diagnostics each is expected to flag), and **doctest** sidecars — plus a **behavioral**
gate that runs generated programs in a throwaway module. Because golden files are generated
*from* the tool, the real verification is that generated programs **independently compile**
and that runtime-preserved output (assert, doctests, derive) actually **runs**. `task check`
and `task fixpoint` are the two correctness gates.

## Project files

- `docs/goal-design-spec.md` — the language design spec (read-only; covers features 01–11).
- `docs/DECISIONS.md` — the choice / assumption / refusal ledger, §01–§12.
- `docs/STATUS.md` — the empirically-verified current language status (the live picture).
- `docs/archive/` — superseded planning docs kept for the record (`TODO.md`,
  `REWRITE-ARCHITECTURE.md`, `FEATURE-AUDIT-PROMPT.md`, `ROADMAP_TO_GOAL.md`, `SELF-HOST-*.md`).
- `features/NN-*/` — per-feature reference material (`SYNTAX.md`, `TRANSPILE.md`, and
  `examples/`). The standalone per-feature transpilers that once lived here were retired
  once the AST front-end subsumed them; these directories now serve as the design source of
  truth for each feature.

---
status: complete
updated: 2026-06-25
---

# Research: `goal fix` command

## Executive Summary

`goal fix` is a source-to-source migration aid that detects "plain Go" patterns
inside `.goal` files and rewrites them into idiomatic goal — the inverse direction
of the existing lowering passes. The single highest-value, provably-safe transform is
collapsing manual error/nil propagation into the `?` operator. goal already has all the
infrastructure needed (lexical scanner, name-keyed `analyze.Tables`, byte-span splicing,
`project.Discover`, and the `--emit` file-writing pattern); `fix` is a new reader of that
infrastructure, not new infrastructure.

## Findings

### Codebase Context

- **No AST.** goal is a *lexical* source-to-source rewriter. The IR is a token stream
  (`[]scan.Token`, each carrying byte spans) plus name-keyed `analyze.Tables`. Passes
  match token patterns and emit `scan.Replacement{Start,End,Text}` lists applied via
  `scan.Splice`. `goal fix` should be built the same way — splice minimal edits, never
  reparse-and-reprint. (`internal/scan/scan.go`, `internal/pass/*.go`.)
- **CLI dispatch.** `cmd/goal/main.go:82` `run()` switches on the subcommand. New
  subcommands register in `guideCommands` (`main.go:36`) and add a `case` plus a
  `cmdFix(...)` function. Flags are parsed by hand (`parseFlags`, `main.go:130`).
- **File discovery.** `project.Discover(root)` (`internal/project/project.go:53`) walks a
  path, collects `.goal` files grouped into `*Package`s, each `File` carrying raw `.Src`.
  Skips hidden/`_`-prefixed dirs and `testdata`. This is the shared entry every command
  uses and is reused as-is.
- **Writing files.** `emitFiles` (`main.go:436`) is the `--emit` analog of `-inplace`:
  `os.WriteFile(path, bytes, 0o644)` and log to stdout. `-inplace` follows this pattern.
- **Output convention.** Success → stdout (`out`), diagnostics → stderr (`errOut`),
  rendered as `file:line:col: severity: [code] message` (`main.go:332`).
- **Tests.** `cmd/goal/main_test.go` builds a temp module via `goalModule(t, files)` and
  calls `run(args, &out, &errOut)` in-process, asserting on side effects (files written,
  stdout/stderr content). No subprocess, no testify (stdlib `testing` only). Golden
  `.goal`/`.go` pairs live under `testdata/`.

### Domain Knowledge — the goal idioms (verified against real `.goal` files)

From `testdata/mixed_result_option.goal` and `internal/pass/question.go`:

- **Open-E Result.** `func f(...) Result[T, error]`. Success is `return Result.Ok(v)`;
  failure is `return Result.Err(e)`. Lowers to a plain Go `(T, error)` tuple.
- **`?` operator.** Always the RHS of an assignment: `v := expr?` keeps the value,
  `_ := expr?` discards it; a bare `expr?` is rejected. Crucially, `?` is lowered purely
  from the **enclosing** function's mode (recovered by name from `analyze.Tables`); the
  callee `expr` only needs to evaluate to a `(value, error)` pair. That means `?` works
  over goal Result calls *and* plain stdlib calls (`f := os.Open(p)?`). **Consequence:
  collapsing `if err != nil` propagation is fully body-local — it changes no signature
  and ripples to no caller**, as long as the enclosing function already returns
  `Result`/`Option`.
- **Option.** `func f(...) Option[T]`; `return Option.Some(v)` / `return Option.None`;
  consumed with `?` or `match`. Lowers to `*T`.
- **Result/Option `?` lowering templates** (`question.go:48-61`) define the exact reverse
  shapes `fix` must recognize:
  - Result keep: `v, err := <expr>; if err != nil { return zero, err }`  ⇄  `v := <expr>?`
  - Result discard: `if _, err := <expr>; err != nil { return zero, err }` ⇄ `_ := <expr>?`
  - Option keep: `o := <expr>; if o == nil { return nil }; v := *o` ⇄ `v := <expr>?`
- **match / enum.** `match scrut { Enum.V(x) => ... }` over a declared `enum`; the variant
  set is available from `analyze.Tables.Enums`. A plain `switch` over an enum-typed value
  is a candidate for `match` (body-local, no contract change).
- **Design philosophy** (`README.md`, `goal-design-spec.md` §0): bias every feature toward
  turning *silent runtime failures* into *located compiler/test errors*; stay Go-shaped.
  This ranks the fixers: ignored/propagated errors and nil-deref are the highest-signal
  silent-failure classes, so the error→`?` and pointer→Option fixers matter most.

### Signature changes ripple; body-only fixes do not

The one structural risk: converting a function's **signature** (`(T, error)` → `Result`,
`*T` → `Option`) changes its contract, so every caller must change too — possibly across
files and packages, and exported functions may have callers outside the discovered set.
Body-local fixes (collapse-to-`?` inside an already-`Result` function; `switch`→`match`)
have zero blast radius. This split is the core of the safety tiering below, and `goal
check` is the post-fix safety net regardless.

## Decision Points

- [x] **D1: stdout vs in-place semantics** — Mirror `gofmt`: default prints the rewritten
  source to stdout (preview, no writes); `-inplace` writes files back. Chosen for
  familiarity (familiarity principle).
- [x] **D2: file vs directory input** — Accept either a single `.goal` file or a directory
  (recurse via `project.Discover`). Default path `.`.
- [x] **D3: reformat or splice?** — Splice minimal edits only; never reflow untouched code.
  goal source is not `go/format`-able (it contains `?`, `match`, `Result[...]`).
- [x] **D4: MVP scope** — Ship the body-local, ripple-free fixer first (collapse manual
  propagation to `?`). Signature-converting fixers (with same-package call-site updates +
  exported-symbol warnings) are the next tier. See spec priorities.
- [x] **D5: idempotence** — Running `fix` twice must be a no-op on the second run.

## Recommendations

1. Implement `fix` as a new package `internal/fix` exposing pure
   `Fix(src string, t *analyze.Tables) (newSrc string, changes []Change, err error)`
   functions per fixer, wired into a thin `cmdFix` in `cmd/goal`. This keeps the rewrite
   logic unit-testable with golden files independent of the CLI.
2. Sequence by blast radius: P1 collapse-to-`?` (body-local) → P2 signature conversion +
   call-site updates → P3 `switch`→`match`. Each fixer is independent and individually
   shippable.
3. Reuse `scan.Lex`, `scan.Splice`, `funcSpans`/`sigAt`, `analyze.Build`, and the
   `parseFlags`/`emitFiles` patterns verbatim. No new infrastructure.

## Sources

- `cmd/goal/main.go` (dispatch, parseFlags, emitFiles, output conventions)
- `internal/project/project.go` (Discover)
- `internal/scan/scan.go` (Lex, Splice, Replacement)
- `internal/analyze/analyze.go` (Tables, FuncSig, Mode, Enums)
- `internal/pass/question.go` (`?` lowering templates — the reverse-mapping source of truth)
- `testdata/mixed_result_option.goal` (canonical Result/Option/`?`/match syntax)
- `README.md`, `goal-design-spec.md` §0/§3 (design philosophy, open-vs-closed lint stance)

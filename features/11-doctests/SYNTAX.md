# 11-doctests — runnable doctests

## Final surface syntax

A doctest is written in a `///` doc comment as a `>>> <expr>` line followed by an expected-output
line:

```goal
/// Adds two ints.
/// >>> add(2, 3)
/// 5
func add(a int, b int) int { return a + b }
```

It is **extracted and executed** as part of the standard build (§4.1) — lowered to a generated
`_test.go` that runs under `go test`. The hard requirement is met by construction: a doctest becomes
a real test, so there is **no way for it to silently not-run**.

### Chosen form (user-selected)

- **Marker: `///` triple-slash doc lines.** Chosen over reusing standard Go `//` doc comments.
  `///` visually flags doctest-bearing documentation and never collides with ordinary `//`
  comments, so a stray `>>>` in normal prose can't accidentally become a test. Lands on the
  Rust / C# doc-comment idiom — a small, deliberate familiarity spend (Go programmers only know
  `//`), justified because doctests are the **top feedback band** and deserve a distinct marker.
  Conveniently, `///` is *also* a valid Go line comment, so the original source still compiles
  untouched.
- **Expectation: expected value on the next line.** Chosen over inline equality
  (`>>> add(2, 3) == 5`). The `>>> expr` / expected-output pair reads like a REPL transcript
  (Python / Rust doctest idiom) and lowers directly to §8.6's `got := <expr>; want := <expected>;
  if got != want` shape.

## Grammar

```ebnf
DocBlock    = { "///" DocText } .                  (* attached to the decl below it *)
Doctest     = "///" ">>>" Expression "\n"
              "///" ExpectedExpression .
```

- A `///` block **attaches to the free function declared immediately below it** (the test is named
  after that function).
- Inside the block, each `>>> <expr>` line forms one doctest; the **next** `///` line is its
  expected output. Lines without `>>>` (prose) are documentation and generate nothing.
- The expected output is written as a **Go expression/literal** (`5`, `"abab"`), because it lowers to
  `want := <expected>`.

## Worked examples

### 1. The spec sample — one doctest

```goal
/// Adds two ints.
/// >>> add(2, 3)
/// 5
func add(a int, b int) int { return a + b }
```

### 2. Multiple doctests on one function

```goal
/// >>> repeat("ab", 2)
/// "abab"
/// >>> repeat("x", 3)
/// "xxx"
func repeat(s string, n int) string { … }
```

Generates `TestDoctest_repeat_1` and `TestDoctest_repeat_2`.

### 3. Prose-only doc generates nothing

```goal
/// half returns x/2. No `>>>` line here.
func half(x int) int { return x / 2 }

/// >>> double(21)
/// 42
func double(x int) int { return x * 2 }
```

Only `double` produces a test; a doc comment without a `>>>` is just documentation.

## Rationale, tied to the two principles

- **Top feedback band (priority #1, best tier):** tests > compiler > prose. Doctests co-locate
  executable verification with the code, and — critically — they *run*. A doctest-shaped comment that
  doesn't run is unverified prose (the *lowest* band) and actively misleading when it drifts; the
  whole feature exists to forbid that (§4.1 "Refused: doctests that can silently not-run").
- **Located failures:** a failing doctest surfaces through `go test` with the function name and the
  got/want values — a located error, not silent drift.
- **Familiarity:** `>>> expr` / expected is the Python & Rust doctest transcript shape; `///` is the
  Rust/C# doc marker. The output is a perfectly ordinary Go `_test.go`.

## Resolved open questions (§9 / §4.1)

- **Doctest marker** → `///` triple-slash. **Expectation form** → expected value on the following
  line. Both decided via `AskUserQuestion`; the rejected alternatives were standard `//` doc comments
  and inline `== ` equality (see `DECISIONS.md`).
- **goscript doctest form** (§4.1 / §9) → out of scope. This audit pins the **Go transpile path
  only**; goscript needs its own runner (no `go test` there), a separate workstream.

## Open against spec

- **Expected output is a Go expression, not free-form REPL text.** Python doctest compares printed
  representations (bare `5`, `hello`); here the expected line lowers to `want := <expected>`, so it
  must be a Go literal/expression (`5`, `"abab"`). Documented divergence — it keeps the generated
  test trivially correct without a value-parsing/printing layer.
- **Scope:** doctests attach to **free functions**; methods, multi-line expected output, and
  doctests on non-function decls are deferred (noted in `TRANSPILE.md`).

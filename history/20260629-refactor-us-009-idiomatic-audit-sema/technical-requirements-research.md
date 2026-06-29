# Technical Requirements & Research — US-009 (sema audit)

## Worklist (from `goal fix selfhost/sema/*.goal` + manual survey)

Every error-returning / fallible function in selfhost/sema, classified:

### CONVERT (genuine, behavior-preserving, no cross-package edits)

- **`Analyze(src string) ([]Diagnostic, error)`** (analyze.goal:21) — exported but
  has ZERO consumers in the selfhost tree and ZERO oracle tests (no analyze_test in
  the behavioral gate). Its body is a textbook propagation site:
  `file, err := parser.ParseFile(src); if err != nil { return nil, err }`.
  Convert to `Result[[]Diagnostic, error]` with `parser.ParseFile(src)?` and
  `return Result.Ok(Check(file, info))`. ModeResult (open-E Result) lowers back to
  native `([]Diagnostic, error)`, so the EMITTED Go signature is unchanged →
  behavior preserved, fixpoint byte-identical, no caller edits.

### REFUSE (record in DECISIONS.md)

- `EnrichForeign([]error)` — exported, error-ACCUMULATOR (returns []error, appends
  and continues), not a (T,error) propagation site; `?` does not apply.
- `foreignDecls (structs, funcs, methods, err)` — unexported but returns 4 values;
  a Result holds one success value, and the sole caller (EnrichForeign) accumulates
  rather than propagates.
- `DefaultResolver (string, error)` — exported and IS the value assigned to the
  exported `DirResolver` func type (`resolve = DefaultResolver`); oracle tests pass
  resolvers of that type. Changing it breaks the type and the tests.
- `goListResolve (string, error)` — unexported but tail-returned by DefaultResolver
  (`return goListResolve(...)`), which stays (string,error); `?` cannot apply at a
  non-propagating tail return into a pinned boundary.
- `AnalyzePackageInDir ([][]Diagnostic, error)` / `AnalyzePackageInDirWith
  ([][]Diagnostic, []error, error)` — exported, oracle-pinned (package_test), and
  the With variant returns 3 values; the InDir variant calls it 3-valued so `?`
  cannot apply.
- `constIntLit (int64, bool)`, `moduleResolve (string, bool)`,
  `readModulePath (string, bool)` — comma-ok value helpers. The inner `err != nil`
  is collapsed to a control-flow bool (two failure modes → one bool); converting to
  Option swallows-not-propagates, so `?` does not fit. Established comma-ok refusal
  (US-005).

### Enum/match

- `Mode (type Mode int + iota)` and `Severity (type Severity int + iota)` —
  exported, ordered, consumed cross-package by `==` and numeric conversions
  (`sema.ModeResultClosed` in backend; `sema.Error/Warning`, `sema.Severity(x)` in
  typecheck/lsp). A goal enum is a boxed sealed interface, not an int, so == and
  conversions break and cross-package consumers would need edits. Canonical
  "ordered/comparable iota int, keep as-is" refusal (token.Kind / FuncMod / ChanDir).
- No in-file `enum` declarations exist, so there is no switch->match candidate
  (switches are over token.Kind / ast node types / Mode / strings). Diagnostic.Code
  and Diagnostic.Feature are stable string identifiers, not enum kinds.

## Verification

- `goal fix selfhost/sema/*.goal` (per file): no source diff; remaining output is
  only skip/suggestion reports for the documented refusals (non-auto-convertible).
- `task check` (incl. internal/selfhost sema port gate + internal/sema), `task
  build`, `task fixpoint` (FIXPOINT OK, byte-identical goal-c-1/goal-c-2).

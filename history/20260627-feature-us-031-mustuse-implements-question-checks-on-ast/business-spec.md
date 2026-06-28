# US-031 Reimplement mustuse/implements/question checks — Business Specification

## Overview

The goal static checker proves the guarantees the front-end erases. As the project
moves from a token-scanning checker to an AST-based one (`internal/sema`), three
remaining guarantees must be reimplemented over the parse tree so `goal check` keeps
rejecting the same programs: Result must-use, `implements` satisfaction, and `?`
propagatability (open-E arity/refusal and closed-E From-totality + error closedness).

The AST reimplementation is correct by construction where the token scanner had to
reconstruct shape from a flat token stream: a match-arm binding and a variant
construction are distinct node types, and an interface method spec is a distinct node
from a statement-leading call — so the legacy false positives are structurally
impossible.

## Functional Requirements

### FR-1: Result must-use (feature 03-result)
A `Result`-returning call (open-E `Result[T, error]` or closed-E `Result[T, E]`)
whose value is dropped as a bare call statement is rejected with an Error
(`dropped-result`). A call consumed by `?`, a `match`, a named bind, a `return`, or
used as an argument carries no obligation. A `_ :=` / `_ =` whole-Result discard is
deferred with an advisory Warning (`unresolved-result-discard`), not rejected.

### FR-2: implements satisfaction (feature 07-implements)
For `type T struct implements I`, T must provide every method I declares — including
methods inherited through embedded interfaces — with a matching normalized signature.
A missing method is an Error (`unimplemented-method`); a present-but-mis-signed method
is an Error (`method-signature-mismatch`); both are located at the `implements` clause.
A sealed interface is trivially satisfied (no diagnostic). An interface that is
qualified (out-of-package) or not declared in this file is deferred with an advisory
Warning (`unresolved-interface`).

### FR-3: open-E `?` propagatability (feature 05-question-prop)
Inside an open-E `Result[_, error]` function, a resolved `?` callee must yield a
trailing `error`: an Option callee, a closed-E Result callee, a void callee, or a
non-error-ending callee is an Error (`question-callee-no-error`); binding a value from
a callee that does not return exactly `(value, error)` is an Error
(`question-binds-nonvalue`). An unresolved discarding `?` callee that is not a
package-qualified call is deferred with an advisory Warning
(`question-callee-unresolved`).

### FR-4: closed-E `?` From-totality and error closedness (feature 06-error-e)
Inside a closed-E `Result[_, E]` function:
- A `?` whose callee is a closed-E Result function with a different error enum `E'`
  requires a registered `from func` converting `E'` to `E`; a missing conversion is an
  Error (`missing-from-conversion`). Same-enum propagation needs none. An unresolvable
  callee is deferred (`unresolved-question-error`).
- Each `Result.Err(X)` must name a variant of `E`: an Err of a different enum is an
  Error (`err-outside-closed-enum`); a name that is not a variant of `E` is an Error
  (`unknown-error-variant`). An `X` that is not a lexically-resolvable `E.Variant`
  construction is deferred (`unresolved-err-value`); an `E` not declared in this file is
  deferred (`unresolved-error-enum`).

### FR-5: Diagnostic-message parity
Every Error/Warning message mirrors the lexical checker's wording so the inline
`// want "substr"` markers in `testdata/check` are satisfied unchanged.

## Acceptance Criteria

- [ ] `internal/sema` implements must-use, implements, and `?`-arity/refusal checks
      over the AST, wired into `sema.Check`.
- [ ] Every `testdata/check/03-result` (must-use) case passes through the AST checker
      via the corpus runner.
- [ ] Every `testdata/check/07-implements` case passes through the AST checker via the
      corpus runner.
- [ ] Every `testdata/check/06-error-e` (closed-E `?`) case passes through the AST
      checker via the corpus runner.
- [ ] A clean open-E `?` (Result callee in a `Result[_, error]` function) produces no
      diagnostic.
- [ ] Adding the new checks does not regress the existing sema runners (02-match
      exhaustiveness, 08-no-zero-value field-completeness) which also exercise
      `sema.Check`.
- [ ] `go build ./...`, `go vet ./...`, `go test ./... -count=1` are green.

## User Interactions

No new user surface. The checks flow through `goal check` (and the corpus
`SemaCheck` adapter); their diagnostics render as `file:line:col: severity: [code] msg`.

## Error Handling

Every check follows the "defer, never guess" discipline: when a fact cannot be
resolved lexically/structurally (unknown callee, out-of-package interface, untyped
binding), the check emits a located advisory Warning naming what could not be resolved,
never a false Error.

## Out of Scope

- The assert (10) and convert (12) checks — those belong to later stories.
- A native goal type checker — `?` arity uses the resolved signature facts only.
- `goal fix`, `goal fmt`, and LSP wiring of these checks (later stories).
- Removing the lexical `internal/check` implementations (US-043).

## Open Questions

None. Behavior is pinned by the existing `testdata/check` `// want` markers, with the
lexical checker as the message-wording oracle.

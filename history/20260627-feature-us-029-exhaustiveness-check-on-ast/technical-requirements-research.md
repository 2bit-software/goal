# Technical Requirements & Research — US-029

## Architecture anchors (REWRITE-ARCHITECTURE.md, progress.txt patterns)

- The check is reimplemented in `internal/sema`, which derives name-keyed facts by
  walking the parsed goal AST (`sema.Resolve(*ast.File) *Info`). `Info.Enums` maps an
  enum name to `*Enum{Variants []Variant (decl order), VSet}`.
- `internal/sema` must stay independent of `internal/analyze` (it is the replacement)
  and should NOT depend on `internal/check` either (check transitively imports
  analyze). So sema defines its own `Diagnostic`/`Severity` and an exhaustiveness
  check returning sema diagnostics.
- The corpus check runner (`internal/corpus`, US-004) drives any `Checker` —
  `Check(src string) ([]check.Diagnostic, error)` — against inline `// want` markers.
  A sema-backed `Checker` adapter lives in `internal/corpus` (which already imports
  `check`; adding a `sema` import creates no cycle — nothing sema imports imports
  corpus). The adapter parses, resolves, runs sema's exhaustiveness check, and
  converts sema diagnostics → `check.Diagnostic` (Pos = token.Pos.Offset).

## Match model (ast/goal_expr.go)

- `MatchExpr{Match token.Pos, Subject, Arms []*MatchArm, ...}`. `MatchArm.Pattern`
  is a `VariantPattern` (Enum: *Ident/*SelectorExpr, Variant: *Ident) or a
  `RestPattern` (`_`). Walk descends into MatchExpr (US-016).
- Resolve the enum from the FIRST variant-qualified arm (qualifier = the `Enum`
  ident), never from the scrutinee — so the check is position-independent. Skip
  `Result`/`Option`. Unknown enum → Warning "exhaustiveness deferred". Missing
  variants with no `_` → Error.

## Diagnostic message parity

Reuse the legacy message wording so the existing `// want` markers match:
- deferred: "... exhaustiveness deferred"
- non-exhaustive: "non-exhaustive `match` on enum `X`: missing variant(s)
  `X.V`[, `X.W`] — handle it/them, or add a `_` rest-arm to dismiss the rest"

Helpers `plural`/`pronoun`/`quoteVariants` are tiny and copied into sema (no
cross-package borrow needed).

## Verification

- `go build ./...`, `go vet ./...`, `go test ./... -count=1` (prd verifyCommands).
- New test in `internal/corpus` drives the 02-match check cases through the
  sema-backed checker via `RunCheck` and asserts all pass.
- New unit test in `internal/sema` over a representative source.

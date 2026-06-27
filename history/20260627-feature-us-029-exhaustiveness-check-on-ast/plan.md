# Implementation Plan — US-029 Match exhaustiveness over the AST

## Components

### 1. `internal/sema/check.go` (new) — diagnostic spine + exhaustiveness check

```go
type Severity int
const ( Error Severity = iota; Warning )
func (s Severity) String() string

type Diagnostic struct {
    Pos      token.Pos
    Severity Severity
    Feature  string
    Code     string
    Message  string
}

// CheckExhaustive walks file for MatchExpr nodes and enforces feature-02 coverage
// using info.Enums. Position-independent: enum resolved from the first
// variant-qualified arm, never the scrutinee. Result/Option skipped; unknown enum
// deferred as a Warning; missing variants without `_` -> Error.
func CheckExhaustive(file *ast.File, info *Info) []Diagnostic

// Check runs every sema check currently implemented (US-029: exhaustiveness only;
// US-030/031 extend it).
func Check(file *ast.File, info *Info) []Diagnostic
```

Helpers (copied locally, tiny): `plural`, `pronoun`, `quoteVariants`, `enumName(Expr)`.

Algorithm per MatchExpr:
- Iterate Arms; classify Pattern: `*RestPattern` -> hasRest; `*VariantPattern` ->
  qualifier name via `enumName(p.Enum)` + variant `p.Variant.Name`.
- First qualified arm fixes `enumName`; collect `covered[variant]` for matching qual.
- enumName=="" -> skip. enumName in {Result, Option} -> skip.
- info.Enums[enumName]==nil -> Warning "... exhaustiveness deferred" at MatchExpr.Match.
- hasRest -> skip. missing variants (decl order) empty -> skip.
- else -> Error "non-exhaustive `match` on enum `X`: missing variant(s) ... — handle
  ..., or add a `_` rest-arm to dismiss the rest" at MatchExpr.Match.

### 2. `internal/corpus/sema_checker.go` (new) — sema-backed Checker adapter

```go
// SemaCheck parses src, resolves it, runs sema.Check, and converts the sema
// diagnostics to check.Diagnostic (Pos = token.Pos.Offset). It satisfies Checker
// via CheckerFunc, so the corpus check runner can judge the AST-based checker
// against the same inline // want markers as the lexical one.
func SemaCheck(src string) ([]check.Diagnostic, error)
```

Imports: `goal/internal/parser`, `goal/internal/sema`, `goal/internal/check`. No cycle
(nothing those import imports corpus).

## Integration Points

- `sema.Resolve(*ast.File) *Info` (resolve.go) supplies `Info.Enums` (Variants in decl
  order, VSet).
- `ast.Walk` already descends into `MatchExpr`/`MatchArm`/`VariantPattern`/`RestPattern`
  (US-016), so CheckExhaustive uses a `Walk` visitor (or `ast.Inspect`-style closure)
  to collect MatchExpr nodes.
- `internal/corpus` `Checker`/`CheckerFunc`/`RunCheck` (US-004) consume `SemaCheck`.

## Testing Strategy

- `internal/sema/check_test.go` (package sema): unit test over a representative source
  with an exhaustive match (no diag), a non-exhaustive match (Error + missing variant),
  a rest-arm match (no diag), and a Result match (skipped).
- `internal/corpus/sema_checker_test.go` (package corpus): `TestSemaExhaustiveRunner`
  loads the manifest, selects check cases under `testdata/check/02-match`, and drives
  each through `RunCheck(repoRoot, c, CheckerFunc(SemaCheck))`; asserts all pass and
  t.Fatalf on zero cases.

## Dependency order

1. `internal/sema/check.go` (no new deps).
2. `internal/corpus/sema_checker.go` (depends on sema).
3. Tests.

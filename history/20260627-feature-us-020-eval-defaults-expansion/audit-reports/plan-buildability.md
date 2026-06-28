# Plan Audit: Buildability — US-020

## Findings

No CRITICAL findings. No MAJOR findings.

- Dependency order is valid: helper -> evalCompositeLit change -> test. No
  forward references.
- Interface contract `zeroValue(typ string, depth int) Value` is concrete and
  uses existing value.go constructors (StrVal/BoolVal/IntVal/FloatVal/NilVal/
  SliceVal/StructVal) and `ip.info.Structs` (verified present in sema.Info).
- File paths verified: internal/interp/eval.go exists; defaults_test.go does not
  yet exist (no conflict).
- Integration point is specific: evalCompositeLit `case *ast.Ident:` branch,
  reached from evalExpr's CompositeLit case.
- `*ast.SpreadElement` with `X *ast.Ident` is the verified parse shape (per
  internal/ast/goal_expr.go and backend lower.go compositeLit handling).

### MINOR-1: complex/array zero
zeroValue should fall through complex/array kinds gracefully; not exercised by
fixtures. Implementation will treat complex as 0 and an array name as a struct-
like composite — acceptable, untested.

## Assumptions
- `ip.info.Structs` is populated for the program under test (true: newInterp
  calls sema.Resolve). A struct type absent from Structs is a descriptive
  refusal, not a silent empty struct.

# Technical Requirements / Research — US-022

## Existing seam

- `internal/interp/interp.go` `Run()` is the single entry: finds `func main`
  and calls it. The gate belongs at the TOP of `Run()` (before evaluation).
- `internal/sema` already exposes `Check(file *ast.File, info *Info) []Diagnostic`
  aggregating every native AST check (exhaustiveness, fields, must-use,
  implements, question, closed, assert, convert). A `Diagnostic` carries
  `Pos token.Pos` (with `.String()` => "line:col"), `Severity` (Error/Warning),
  `Feature`, `Code`, `Message`.
- The gate should collect `sema.Check(ip.file, ip.info)`, and if any
  `Severity == sema.Error` diagnostic is present, refuse with a located error
  (include Pos.String(), Code, Message) BEFORE running main. Warnings (located
  deferrals) do not block.

## Dependency envelope

- `go list -deps ./internal/interp` currently shows NO go/types, no
  internal/typecheck, no internal/backend. The new test must lock this in:
  scan `go list -deps ./internal/interp` output and fail if `go/types` or
  `goal/internal/typecheck` appears.

## No new dependencies

- The interpreter already imports `goal/internal/sema`. Adding the gate needs
  no new import.

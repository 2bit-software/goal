# Verify — US-040

## verifyCommands (prd.json)
- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (all packages green)

## Acceptance criteria
- AC1 "new backend extracts /// doctests and emits the _test.go sidecar lowered
  through the same path": emitDoctests (internal/backend/doctest.go) reads the
  structured ast.FuncDecl.Doc.Doctests, renders a goal-shaped sidecar, and lowers
  it via emitFile with the original *sema.Info — same emit path as bodies.
  goBackend.Emit now populates Output.Test. VERIFIED.
- AC2 "the 11-doctests cases pass the doctest tier through the new backend":
  TestASTEngineDoctestTier (sidecar-vs-golden) + TestASTEngineDoctestExecTier
  (behavioral go test in a temp module) over all 4 cases — green.
  TestASTEngineDoctestEnumLowering pins that the enum case's nested variant
  constructions lower to the §8.1 sum encoding in the sidecar. VERIFIED.

Result: PASS.

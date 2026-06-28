# Tasks — US-028

Status: T1 completed, T2 completed, T3 completed.

## T1: Define TypeChecker interface + GoTypesChecker
- New file `internal/typecheck/checker.go`.
- Add `TypeChecker interface { Check(pkg *project.Package) ([]Diagnostic, error) }`.
- Add `GoTypesChecker struct{}` implementing `Check` by calling `Load` then
  `CheckImplements` + `CheckMustUse` + `CheckNoZeroValue`.
- Covers: FR-1, FR-3.
- Verify: `go build ./internal/typecheck`.

## T2: Test depth checks through the interface
- New file `internal/typecheck/checker_test.go`.
- Drive a fixture package (existing `pkgOf` helper / harness sources) through a
  `var tc TypeChecker = GoTypesChecker{}` value; assert parity with the concrete
  `Load`+`Check*` path and an error case for an untranspilable package.
- Covers: AC "test exercises the depth checks through the interface".
- Verify: `go test ./internal/typecheck -count=1`.

## T3: Route caller through the interface
- Modify `cmd/goal/main.go` `runDepthChecks` to obtain diagnostics via a
  `typecheck.TypeChecker` value instead of calling `Load`/`Check*` directly.
- Covers: FR-2.
- Verify: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.

Order: T1 → T2 → T3 (each compiles independently; T3 depends on T1's interface).

# Verify — US-011 Host-function bridge for stdlib

## Acceptance criteria

AC1 — registry resolving fmt.Sprintf, fmt.Sprint, fmt.Println, fmt.Errorf,
errors.New to native Go; unresolved imported call -> located, named error.
- MET. internal/interp/host.go `hostFuncs` registers all five symbols.
  `evalHostCall` raises `interp: <line:col>: unresolved imported call
  <pkg.Sym> (no host function registered)` for any imported symbol with no
  shim. evalCallMulti routes non-shadowed imported-package selector calls to it.

AC2 — unit test runs a goal program calling fmt.Sprintf and asserts the produced
string; asserts an unregistered imported call yields a descriptive error naming
the missing symbol.
- MET. internal/interp/host_test.go:
  - TestHostSprintf asserts "x-7".
  - TestUnregisteredImportedCallNamedError asserts the error names
    "strings.ToUpper" and is located (contains a ":").
  - Supporting: TestHostSprint, TestHostErrorsNew, TestHostErrorfWrapsError,
    TestShadowedPackageNameNotRoutedToHost.

## Gates (prd.json verifyCommands)

- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages)

## US-022 dependency envelope

`go list -deps ./internal/interp` excludes go/types, internal/backend,
internal/typecheck — the native-front-end seam stays clean.

## Findings

CRITICAL: none. MAJOR: none. MINOR: none.

Recommendation: PASS.

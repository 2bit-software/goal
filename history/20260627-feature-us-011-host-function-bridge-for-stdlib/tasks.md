# Tasks — US-011 Host-function bridge for stdlib

## T1 — Import registration (foundation)
- Add `imports map[string]string` to `Interp`; populate in `New` via
  `registerImports` (local name -> import path from `file.Imports`).
- Files: internal/interp/interp.go
- Covers: AC1 (recognizing imported packages).

## T2 — Host registry + dispatch
- New internal/interp/host.go: `hostFunc`, `hostFuncs` registry (fmt.Sprintf,
  fmt.Sprint, fmt.Println, fmt.Errorf, errors.New), `errVal`, `goArg`/`goArgs`,
  `evalHostCall` (located, named refusal for an unregistered symbol).
- Files: internal/interp/host.go
- Depends on: T1.
- Covers: AC1 (registry + named refusal).

## T3 — Wire interception into call evaluation
- In `evalCallMulti`, route a non-shadowed imported-package selector call to
  `evalHostCall` before `tryMethodCall`.
- Files: internal/interp/eval.go
- Depends on: T1, T2.
- Covers: AC1.

## T4 — Tests
- internal/interp/host_test.go: fmt.Sprintf result, errors.New / fmt.Errorf
  message, unregistered imported call named+located error, shadowing fall-through.
- Files: internal/interp/host_test.go
- Depends on: T2, T3.
- Covers: AC2.

## T5 — Verify gates
- `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
- Confirm `go list -deps ./internal/interp` excludes go/types, internal/backend,
  internal/typecheck (US-022 envelope).

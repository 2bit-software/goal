# Verify — Acceptance Audit — US-004

Source of truth: business-spec.md acceptance criteria.

## Results

- [x] AC-1: Trivial `package main\nfunc main() {}` parsed via parser + resolved
      via sema runs through the interpreter with no error.
      → TestRunTrivialMain (PASS).
- [x] AC-2: Construction takes the parsed `*ast.File` + `*sema.Info` and requires
      no Go-lowered input. → New(file, info); TestConstructFromSharedFrontEnd
      (PASS). interp.go imports only internal/ast, internal/sema, stdlib errors —
      NOT internal/backend.
- [x] AC-3: A program with no top-level `func main` yields a descriptive error
      naming the missing entry point. → ErrNoMain ("interp: no func main
      declared"); TestRunMissingMainErrors asserts errors.Is + message contains
      "main" (PASS).

## Gate commands
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages)

No CRITICAL or MAJOR findings; every acceptance criterion is satisfied and tested.

## Assumptions
- Single-file `*ast.File` input (package-mode entry deferred), matching the AC.
- Empty `main` body is a no-op; statement evaluation is US-005+.
- `ErrNoMain` is a stable sentinel so callers can match it with errors.Is.

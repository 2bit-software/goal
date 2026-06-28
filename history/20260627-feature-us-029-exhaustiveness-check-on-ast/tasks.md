# Tasks — US-029

- [ ] T1. Add `internal/sema/check.go`: `Severity`, `Diagnostic`, `CheckExhaustive`,
  `Check`, and local helpers (`enumName`, `plural`, `pronoun`, `quoteVariants`).
- [ ] T2. Add `internal/corpus/sema_checker.go`: `SemaCheck(src) ([]check.Diagnostic, error)`
  — parse → resolve → sema.Check → convert to check.Diagnostic.
- [ ] T3. Add `internal/sema/check_test.go`: unit coverage (exhaustive / non-exhaustive /
  rest-arm / Result-skip).
- [ ] T4. Add `internal/corpus/sema_checker_test.go`: drive every `testdata/check/02-match`
  check case through `RunCheck` + `CheckerFunc(SemaCheck)`; t.Fatalf on zero cases.
- [ ] T5. Verify: `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green.

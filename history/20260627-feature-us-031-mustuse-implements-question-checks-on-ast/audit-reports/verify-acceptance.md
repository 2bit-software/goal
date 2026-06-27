# Verify — Acceptance Coverage

Verify gates (prd.json verifyCommands), all green:
- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (every package passes)

| Acceptance criterion | Evidence |
|----------------------|----------|
| sema implements must-use, implements, ?-arity/refusal over the AST, wired into `sema.Check` | `internal/sema/{mustuse,implements,question}.go`; `check.go` appends `CheckMustUse`/`CheckImplements`/`CheckQuestion`/`CheckClosed`. |
| Every 03-result must-use case passes via corpus runner | `TestSemaQuestionImplementsRunner` — 5 subtests under `testdata/check/03-result/` PASS. |
| Every 07-implements case passes via corpus runner | same test — 9 subtests under `testdata/check/07-implements/` PASS. |
| Every 06-error-e closed-E `?` case passes via corpus runner | same test — 7 subtests under `testdata/check/06-error-e/` PASS. |
| Clean open-E `?` (Result callee) produces no diagnostic | `internal/sema/question_test.go` `TestQuestionOpenEResultCalleeClean`; corpus `03-result/consumed_clean`. |
| No regression of 02/08 sema runners | `TestSemaExhaustiveRunner` + `TestSemaFieldsRunner` still PASS under the extended `sema.Check`. |
| build/vet/test green | see gates above. |

Total US-031 corpus cases driven through the AST checker: 21, all PASS. Unit tests:
must-use (6), implements (9), question/closed (9) — all PASS.

No acceptance criterion is unmet.

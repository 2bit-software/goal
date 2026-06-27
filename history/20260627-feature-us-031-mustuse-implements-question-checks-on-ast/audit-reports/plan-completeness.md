# Plan Audit — Coverage

Every spec requirement traces to a plan element:

| Spec | Plan element |
|------|--------------|
| FR-1 must-use | `internal/sema/mustuse.go` `CheckMustUse` |
| FR-2 implements | `internal/sema/implements.go` `CheckImplements` + `Info.Interfaces/EmbeddedIfaces` |
| FR-3 open-E `?` | `internal/sema/question.go` `CheckQuestion` |
| FR-4 closed-E `?` + Err closedness | `internal/sema/question.go` `CheckClosed` |
| FR-5 message parity | mirror `internal/check/*` wording (called out per check) |
| AC corpus 03/06/07 | `internal/corpus/sema_question_test.go` |
| AC no 02/08 regression | risk note + `go test ./...` gate |
| AC build/vet/test | verifyCommands |

No orphan plan elements (no scope creep). No unmapped acceptance criteria. No CRITICAL/MAJOR findings.

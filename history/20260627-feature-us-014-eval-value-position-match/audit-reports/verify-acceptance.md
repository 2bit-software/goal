# Verification — Acceptance Coverage

Full suite green: `go build ./...`, `go vet ./...`, `go test ./... -count=1`
(all packages `ok`). Dependency envelope preserved: `go list -deps
./internal/interp` shows no go/types, internal/backend, internal/typecheck.

| Acceptance criterion | Evidence |
|---|---|
| `return match` returns the matched arm's value per variant | TestValueMatchInReturn (Point=0, Circle=9, Square=16) |
| `x := match` binds the matched value per variant | TestValueMatchInShortVar |
| `var x = match` binds the matched value per variant | TestValueMatchInVarDecl |
| Payload arm computes from the bound payload | Circle arm `c.radius*c.radius` asserted = 9 across all three positions |
| `_` rest arm supplies the value when no variant arm matches | TestValueMatchRestArmSuppliesValue (descSquare=0, descCircle=7) |
| Tag matching no arm raises an `unreachable` panic | TestValueMatchDefaultIsUnreachable (panicSignal, message contains "unreachable") |
| Statement-position match unchanged | match_test.go still passes; execMatch/execArm refactored behaviour-preserving |
| Non-variant scrutinee refused | TestValueMatchOnNonVariantIsRefused |

All acceptance criteria are covered by an asserting test.

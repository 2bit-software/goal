# Tasks — US-025 Add AST snapshot tests

- [ ] T1: Add `internal/ast/dump.go` with `Sexpr(Node) string` (reflection-based,
      position-omitting, deterministic s-expression renderer).
- [ ] T2: Add `internal/parser/snapshot_test.go` — `-update-snapshots` flag, table
      of one representative input per goal construct, parse → render → compare.
- [ ] T3: Generate goldens with `go test ./internal/parser -run TestASTSnapshots
      -update-snapshots`; inspect for sanity.
- [ ] T4: Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.

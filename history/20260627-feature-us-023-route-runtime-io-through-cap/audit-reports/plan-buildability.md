# Plan Audit — Buildability

Plan: implementation-plan.md

## Findings

- Dependency order is valid: cap (exists) → interp.go fields/gate → host.go
  routing → tests. No forward references.
- Interface contracts are concrete: `Option`, `WithCapabilities`, `WithStdout`,
  `New(file, info, opts ...Option)`, `emitStdout(write func(io.Writer) error) error`.
  Signatures agree across interp.go and host.go.
- File paths verified: `internal/interp/interp.go`, `internal/interp/host.go`
  exist; `internal/cap/cap.go` exports `GrantAll`/`Has`/`Capability`. New test
  path does not collide.
- Integration point is specific: `evalHostCall` branches on `key == "fmt.Println"`.
- No CRITICAL/MAJOR findings.
- MINOR: confirm `host.go` still imports `os` only if another reference remains;
  otherwise remove it to keep `go vet`/build clean. Noted in the plan.

## Assumptions

- Variadic functional options are acceptable (keeps every existing `New(file,
  info)` call site compiling — verified there are no external importers).
- `internal/cap` is dependency-free, so importing it does not breach the US-022
  no-`go/types`/no-`typecheck` dependency gate.

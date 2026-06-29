# AI-Consumer Readiness Audit — US-003

## Findings

- The mechanics are fully specified in `technical-requirements-research.md`:
  Discover→emit path semantics, the nested-module `go.mod`, `go build -C`, the
  port_test key rename, and the BuildAndTest test-file import rewrite. An
  implementer can proceed without guessing.
- Acceptance criteria are testable as assertions: `grep` for `goal/internal/` in
  `selfhost/*.goal` must be empty; `task fixpoint`/`check`/`build` must exit 0.
- MINOR: The exact go directive version for the nested go.mod must match the repo
  (`go 1.26`). Recorded in the research doc.

No CRITICAL or MAJOR findings.

## Assumptions

- Blanket string rewrite `goal/internal/` → `goal/selfhost/` is safe in both the
  selfhost sources and the copied behavioral test files (verified empirically:
  every occurrence is an import line or a comment, no string literals).
- `selfhost_test.go` (operating on the real `internal/*` sources) stays unchanged;
  only `port_test.go` (operating on selfhost sources) flips its layout keys.

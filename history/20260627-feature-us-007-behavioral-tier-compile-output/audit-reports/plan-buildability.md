# Plan Buildability Audit — US-007

## Findings

- Dependency order is valid: runner func has no forward references; test depends
  on it plus existing manifest/pipeline.
- Interface contract `RunCompile(root string, c Case, tp Transpiler) error`
  matches the existing `RunTranspile` shape and the established `Transpiler`
  seam — types agree.
- File paths verified against the codebase: `internal/corpus/` exists with the
  sibling runner files; new files do not collide.
- Integration point is specific (reuse `Transpiler`; mirror
  `cmd/goal/main.go` exec pattern with a standalone temp module).

No CRITICAL or MAJOR findings.

## Assumptions

- A minimal `go.mod` (`module goalcorpus` + `go 1.26`) lets stdlib-only
  generated Go build and vet offline.
- Non-`main` package compiling in isolation is acceptable for the behavioral
  judgement (no `func main` required).

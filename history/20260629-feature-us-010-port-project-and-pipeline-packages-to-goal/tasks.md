# Implementation Tasks — US-010

## Task 1: Port project package to goal
**Status**: completed
**Files**: `selfhost/project/project.goal` (new)
**Depends on**: (none — token/lexer/ast/parser already ported)
**Spec coverage**: FR-1, FR-3 (project half), AC 1-2
**Verify**: file exists; covered by Task 3's BuildTranspiled gate.

### Instructions
Copy `internal/project/project.go` verbatim to `selfhost/project/project.goal`
(Go superset = valid goal; no bare match/enum/assert identifier collisions).

## Task 2: Port pipeline package to goal
**Status**: completed
**Files**: `selfhost/pipeline/pipeline.goal`, `selfhost/pipeline/sourcemap.goal` (new)
**Depends on**: (none)
**Spec coverage**: FR-2, FR-3 (pipeline half), AC 1-2
**Verify**: files exist; covered by Task 3's BuildTranspiled gate.

### Instructions
Copy `internal/pipeline/pipeline.go` -> `selfhost/pipeline/pipeline.goal` and
`internal/pipeline/sourcemap.go` -> `selfhost/pipeline/sourcemap.goal` verbatim.

## Task 3: Add port_test gates for project and pipeline
**Status**: completed
**Files**: `internal/selfhost/port_test.go` (modify)
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-3, FR-4, AC 2-3
**Verify**: `go test ./internal/selfhost -run 'TestPortedProjectPackage|TestPortedPipelinePackage'`

### Instructions
Add `TestPortedProjectPackage` and `TestPortedPipelinePackage` mirroring
`TestPortedSemaPackage`:
- Discover token, lexer, ast, parser, and the package under test from
  `../../selfhost/<pkg>`.
- COMPILE gate: BuildTranspiled over the full layout (deps + package under test).
- BEHAVIORAL gate: BuildAndTest with deps {token,lexer,ast,parser}:
  - project: testFiles = ["../project/project_test.go"].
  - pipeline: testFiles = ["../pipeline/sourcemap_test.go"]; EXCLUDE
    pipeline_test.go (backend/corpus/manifest-dependent).

## Task 4: Verify and finalize
**Status**: completed
**Files**: `prd.json`, `progress.txt` (modify)
**Depends on**: Task 3
**Spec coverage**: AC 4-5
**Verify**: `task check`, `task build`, `task fixpoint` all green.

### Instructions
Run the project-wide gates. When green, mark US-010 passes:true in prd.json and
append the progress.txt entry. (Commit handled by the workflow / loop-runner.)

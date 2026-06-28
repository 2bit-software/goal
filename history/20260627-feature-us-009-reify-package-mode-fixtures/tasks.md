# Implementation Tasks

## Task 1: Extend corpus model with PackageSpec
**Status**: completed
**Files**: `internal/corpus/corpus.go`
**Depends on**: (none)
**Spec coverage**: FR-3 (Mode=package indexing)
**Verify**: `go build ./internal/corpus`

### Instructions
Add `PackageSpec{Name string; Files []string; Imports map[string]string}` (json
tags `name`/`files`/`imports,omitempty`) and a `Package *PackageSpec
json:"package,omitempty"` field on `Case`. Document that package cases are
Kind=transpile + Mode=package and carry their files/import-map here.

## Task 2: Author on-disk package fixtures
**Status**: completed
**Files**: `testdata/package/cross-file-demo/{math.goal,types.goal,pkg.json}`,
`testdata/package/foreign-derive/{conv.goal,pkg.json}`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2
**Verify**: `ls testdata/package/*/*.goal`
### Instructions
Lift `mathGoal`/`typesGoal` (pipeline_package_test.go) into math.goal/types.goal
(package demo); `pkg.json` = `{"name":"demo","imports":{}}`. Lift foreign_test.go
`src` into conv.goal (package conv); `pkg.json` =
`{"name":"conv","imports":{"goal/internal/pipeline/testdata/extpkg":"internal/pipeline/testdata/extpkg"}}`.

## Task 3: Discover package fixtures in Generate
**Status**: completed
**Files**: `internal/corpus/generate.go`
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-3
**Verify**: `go test ./internal/corpus -run TestGenerate`
### Instructions
After the existing transpile/check walks, glob `testdata/package/*/pkg.json`,
decode each, glob the dir's `*.goal` (sorted), and append a Case{Kind:transpile,
Mode:package, Input: fixture-dir-rel, Package:&PackageSpec{...}}. Keep paths
repo-relative slash-form; keep deterministic sort by Input.

## Task 4: Regenerate the manifest
**Status**: completed
**Files**: `corpus/manifest.json`
**Depends on**: Task 3
**Spec coverage**: FR-3
**Verify**: `go run ./cmd/corpus-gen -root .` then grep `"mode": "package"` count == 2

## Task 5: Add the package-mode runner
**Status**: completed
**Files**: `internal/corpus/package_runner.go`
**Depends on**: Task 1
**Spec coverage**: FR-4, Error Handling
**Verify**: `go build ./internal/corpus`
### Instructions
Define `PackageTranspiler`/`PackageTranspilerFunc` over
`pipeline.TranspilePackage`. `RunPackage(root, Case, pt)`: build
`*project.Package{Dir: root/Input, Name: spec.Name, Files}`, transpile, assert
every Files/Tests entry is valid Go (`format.Source`), then write the package +
each declared foreign import (module-relative tail) into a temp `module goal`
and `go build ./...`. Descriptive, case-identified errors.

## Task 6: Tests + count/shape updates
**Status**: completed
**Files**: `internal/corpus/package_runner_test.go`, `internal/corpus/generate_test.go`
**Depends on**: Task 4, Task 5
**Spec coverage**: FR-3, FR-4, all ACs
**Verify**: `go test ./internal/corpus -count=1`
### Instructions
New test loads the manifest and runs every Mode=package case through
`PackageTranspilerFunc(pipeline.TranspilePackage)` via `RunPackage`; loud
zero-case guard; `-short`-skip the compile. Update generate_test: count file-mode
transpile (51) vs package (2), keep check 50 / doctest 4; exempt package cases
from the Expected-non-empty assertion.

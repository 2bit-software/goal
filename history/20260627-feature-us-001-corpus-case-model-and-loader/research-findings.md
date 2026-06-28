# Research Findings — US-001

## Codebase conventions (verified)

- Module is zero-dependency (`go.mod`: `module goal`, go 1.26). Use stdlib only.
- Existing packages with `testdata/` dirs: `internal/pipeline`, `internal/analyze`,
  `internal/check`. New `internal/corpus/testdata` follows the same convention.
- `pipeline.Output` (internal/pipeline/pipeline.go:69) and `check.Diagnostic`
  (severity Error/Warning) are the types that later corpus runners (US-003/004)
  will adapt to — out of scope for US-001, which only defines the data model.

## Decisions for this story

- Manifest format: JSON (stdlib `encoding/json`), matching REWRITE-ARCHITECTURE
  §10 Phase 0.1 ("generated JSON/TOML index"). JSON chosen — stdlib, no dep.
- `Kind`, `Mode`, `Normalize` are string-typed named types with exported
  constants (`KindTranspile`, `KindCheck`, `KindDoctest`; `ModeFile`,
  `ModePackage`; `NormalizeGofmt`, `NormalizeNone`) so JSON is human-readable and
  round-trips.
- `Manifest` is a struct wrapping `[]Case` so it can carry metadata later
  (e.g. corpus root) without breaking callers.
- `Load(path string) (Manifest, error)` reads the file and unmarshals JSON,
  wrapping I/O and parse errors with context.

## Confidence: High — small, fully-specified internal model, stdlib only.

# Plan Buildability Audit — US-002

- Dependency order valid: renames first, then harness, then test. No forward refs.
- Interface contracts concrete and verified against real signatures:
  `backend.TranspilePackage`, `project.Package`/`project.File`, `parser.ParseFile`.
- File paths verified: `internal/selfhost/` does not exist; the 8 covered dirs do.
- No import cycle: backend/parser/pipeline/project do not import `internal/selfhost`.
- Approach empirically validated by an exploratory probe: 6 packages transpile+build
  clean as-is; sema and backend need the two `enum` renames, after which all 8 build.

No CRITICAL or MAJOR findings.

## Assumptions
- Temp module declares `module goal`, `go 1.26` (matches the real go.mod) so
  in-module import paths resolve. Confirmed working in the probe.
- Only `out.Files` are written for the build; `out.Tests` are excluded.

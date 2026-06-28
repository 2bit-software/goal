# Plan Audit — Buildability

- Dependency order is a valid topological sort (model → fixtures → generate →
  manifest → runner → tests); no forward references.
- Interface contracts are concrete signatures; `corpus` already imports
  `pipeline`, and adding `internal/project` introduces no cycle (project imports
  only scan).
- File paths verified against the tree: `internal/pipeline/testdata/extpkg`
  exists as the foreign Go source; `testdata/` and `testdata/check/` precedent
  confirms the `testdata/package/` location.
- Integration points name the exact seam (`pipeline.TranspilePackage`) and the
  resolver (`analyze.DefaultResolver` invoked from `pkg.Dir`).

Risk (from research): the foreign-derive generated Go must compile in the temp
module. Mitigation already noted — if the derive body is not self-contained,
`RunPackage` reduces its bar to transpile-success + valid-Go. Non-blocking.

No CRITICAL or MAJOR findings.

## Assumptions
- Temp compile module is named `goal` so the in-module foreign import path
  resolves; foreign packages are copied under their import-path tail.
- `Case.Input` for package cases is the fixture dir (keeps Input non-empty for
  the shape test and gives `RunPackage` the resolution `Dir`).

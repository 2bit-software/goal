# Audit — AI-Consumer Readiness

## Findings

- An implementer has everything needed: the seam (`pipeline.TranspilePackage`),
  the data model (`corpus.Case` + a `PackageSpec` carrier), the generator
  (`Generate`/`cmd/corpus-gen`), and the resolution mechanism
  (`DefaultResolver`) are all named and located.
- Data formats: `PackageSpec{Name, Files []string, Imports map[string]string}`
  with import-path → repo-relative foreign dir is specific enough to write test
  assertions from.
- Acceptance criteria are independently verifiable via `go test ./...` plus a
  manifest grep for `Mode=package`.

No CRITICAL or MAJOR findings.

## Assumptions

- Fixtures live under `testdata/package/<name>/` with a `pkg.json` descriptor so
  `Generate` stays data-driven (no hardcoded fixture names).
- The declared import map is consumed by the runner to wire foreign packages
  into the temp compile module; in-module transpile-time resolution is handled
  by `DefaultResolver` from the fixture dir.

# Verify — Acceptance Coverage (US-002)

Full suite: `task check` (go vet + `go test ./... -count=1`) green; `task build`
green. Story tests green: `go test ./internal/sema/ -count=1`.

| Acceptance criterion | Evidence |
|---|---|
| Driver parses each file, ResolvePackage, EnrichForeign, then Check per file | `internal/sema/package.go` AnalyzePackageInDirWith steps 1-5 |
| Returns one []Diagnostic per file in input order | `out := make([][]Diagnostic, len(files)); out[i] = Check(...)`; asserted by TestAnalyzePackageInDirCrossFileExhaustiveness (len==2, index-specific) |
| Multi-file fixture returns expected diagnostics incl. a foreign-enrichment-dependent finding | TestAnalyzePackageInDirForeignEnrichedDeriveFinding: unsourced-field Error only when the resolver loads extpkg; control (failing resolver) yields the unresolved-derive-type Warning |
| task check / task build pass | Run green this iteration |

All acceptance criteria covered by passing tests.

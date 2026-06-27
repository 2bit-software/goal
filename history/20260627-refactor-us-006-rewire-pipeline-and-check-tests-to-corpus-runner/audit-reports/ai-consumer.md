# Audit: AI-Consumer Readiness — US-006

## Findings

No CRITICAL findings.
No MAJOR findings.

The spec is implementable without guessing:
- The seams (`corpus.Load`, `RunTranspile`, `RunDoctest`, `RunCheck`,
  `TranspilerFunc`, `CheckerFunc`) already exist with documented signatures.
- The manifest path and repo-root depth (`../..`) are identical to the existing
  corpus tests, which compile and pass.
- The import-cycle constraint and its resolution (external `_test` packages) are
  explicit in the technical research.

### MINOR-1: Test function names not prescribed
The spec does not name the new test functions. This is intentional latitude;
implementation will keep clear, conventional names (e.g. TestCorpusTranspile /
TestCorpusDoctest / TestCorpusCheck) and preserve TestRegistryRuns. Non-blocking.

## Assumptions

- Per-case reporting via `t.Run(case.ID, ...)` is the desired granularity (matches
  the existing corpus runner tests).
- Removing the now-unused local helpers (`mustFormat`, `wantRe`, `parseWants`,
  `runCase`) is in scope since they exist only to serve the code being replaced.

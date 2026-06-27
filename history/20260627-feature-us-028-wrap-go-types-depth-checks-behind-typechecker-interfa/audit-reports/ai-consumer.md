# AI-Consumer Readiness Audit — US-028

## Findings

No CRITICAL or MAJOR findings. An implementer has everything needed:

- The three depth checks, their signatures (`Check*(p *Package) []Diagnostic`),
  and the `Load(pkg *project.Package) (*Package, error)` entry point already
  exist and are referenced by the technical-requirements research.
- The single existing caller (`cmd/goal/main.go` `runDepthChecks`) is identified,
  so FR-2 (caller indirection) has a concrete target.
- Acceptance criteria are directly testable: define interface, implement it,
  route caller through it, drive a test through the interface value, keep the
  verify gates green.

- MINOR: data formats (`Diagnostic`, `Package`) are pre-existing types; no new
  data shapes are introduced, so nothing further to specify.

## Assumptions

- `[]Diagnostic` is the natural return of the seam (the caller already returns
  `[]typecheck.Diagnostic`), so a `Check(pkg) ([]Diagnostic, error)` interface
  swaps cleanly without exposing go/types internals — chosen over a
  `Load`-returning interface because a native checker would not produce a
  `*types.Package`.

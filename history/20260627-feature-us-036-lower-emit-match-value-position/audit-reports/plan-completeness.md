# Plan Completeness Audit — US-036

## Findings

### MAJOR (resolved in plan) — Corpus count assertion
`internal/corpus/generate_test.go:36` hard-asserts `transpile == 51`. Adding a
new file-mode transpile case bumps it to 52; the assertion (and its message)
MUST be updated in the same change or `go test` goes red. Captured in PLAN risk
#1; this is a required step, not optional.

### MAJOR (resolved in plan) — New golden must match the canonical engine
The new `.go.expected` is consumed by the US-003 transpile runner
(`TestTranspileRunner`), which runs the DEFAULT (splice) engine and compares
gofmt-normalized output. The golden therefore MUST be generated from the splice
engine for the chosen input, not hand-written, or that runner fails. The US-036
behavioral test additionally drives the AST engine (build+vet only).

### MINOR — `var name T = match` AST shape
The plan assumes `var d T = match` parses to `DeclStmt{Decl: GenDecl{ValueSpec}}`.
Confirm the shape against `internal/ast` / a quick parse before wiring; the
`status_var` golden already proves the form is supported by the splice engine.

## No CRITICAL findings.
Components, ordering, and integration points are concrete and map to existing
seams (matchStmt/returnStmt/stmt dispatch; renames/gensym/usesIdent helpers).

## Assumptions
- The new case is added as a feature-02 example (or testdata pair) so `Generate`
  indexes it as a file-mode transpile case.
- Golden generated via the existing splice transpile path so the legacy runner
  stays green.

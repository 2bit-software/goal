# Audit — AI-Consumer Readiness (US-002)

All referenced types/functions exist and are named in
technical-requirements-research.md (ParseFile, ResolvePackage, EnrichForeign,
Check, DirResolver, DefaultResolver). Return type `[][]Diagnostic` and the
input-order contract are explicit. Acceptance criteria are directly assertable:
parse + resolve + enrich + check; len(out)==len(srcs); a fixture producing an
unsourced-field Error that depends on foreign enrichment.

## Findings

- MINOR: "the package directory" — confirmed to be the dir passed to
  EnrichForeign as `dir` (import paths resolve relative to it). Tests inject a
  fake DirResolver so the literal dir value is irrelevant under test.

No CRITICAL or MAJOR findings. An AI agent can implement this without guessing.

## Assumptions

- The driver lives in internal/sema (new file package.go) so it can reach the
  unexported merge internals via the exported ResolvePackage/Check/EnrichForeign.
- sema importing parser introduces no cycle (parser deps = token/ast/lexer only).

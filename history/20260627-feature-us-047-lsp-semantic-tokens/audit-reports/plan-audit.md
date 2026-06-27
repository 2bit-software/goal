# Plan Audit

## Coverage
- FR-1 (advertise capability) -> server.go initialize edit + protocol.go legend.
- FR-2 (serve full, from AST) -> semantictokens.go handler + astRoles walk.
- FR-3 (role distinctions) -> classifyToken + roleVisitor mapping.
- FR-4 (document order, delta-encoded) -> computeSemanticTokens encoder.
Every functional requirement traces to a concrete file + function.

## Findings
No CRITICAL or MAJOR findings.

- MINOR: `check.OffsetToPosition` returns 1-based line/col; the encoder must
  subtract 1 (noted, mirrors `rangeOf`/`toLSP`). Verified against existing
  internal/lsp usage.
- MINOR: Legend ordering must match the index constants exactly; the plan calls
  this out. Cheap to get right; a unit test pins the enum/match/`?` indices.

## Dependency ordering
protocol.go (types) -> semantictokens.go (uses types) -> server.go (wires
handler). No cycles; all within package lsp.

## Assumptions
- ASCII length == byte length (corpus-true; same assumption the diagnostics path
  already makes).
- Unknown identifiers are intentionally not emitted (avoids miscoloring builtins
  without a type resolver).

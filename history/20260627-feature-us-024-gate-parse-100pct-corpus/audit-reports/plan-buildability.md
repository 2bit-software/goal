# Plan Buildability Audit — US-024

## Findings

No CRITICAL. No MAJOR.

- Dependency order (ast → parser → corpus) is a valid topological sort; no
  forward references. The new `IndexListExpr` node is built before the parser that
  emits it.
- No import cycle: verified that pipeline/check/project do not import parser, so
  corpus→parser is one-directional.
- Interface contracts are concrete (Go signatures given for IndexListExpr and the
  Parser/ParserFunc/RunParse seam).
- File paths verified against the existing tree (internal/ast, internal/parser,
  internal/corpus all exist; sibling runner files follow the same layout).
- Integration points name exact functions (`parseIndexSuffix`, `parseOperand`,
  `compositeOK`, `parsePayloadField`) confirmed present during recon.

### MINOR-1: Re-run after each gap
After closing the three known gaps, the gate may surface a deeper category the
recon's first-error-per-file view masked. Mitigation: the implement step re-runs
the gate iteratively until all inputs parse, then runs the project gates.

## Assumptions

- `parseOperand` accepting `[`/`map`/`struct` will not regress func-literal or
  control-header parsing (func literals are out of corpus scope; exprLev guards
  composites in headers). To be confirmed by the full test suite.

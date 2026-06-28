# Audit — AI-Consumer Readiness

## Findings

No CRITICAL or MAJOR findings. An implementer has: the exact source examples, the
expected Go (the checked-in `.go.expected` goldens), and the known-good reference
encoder (`internal/pass/derive.go`). Every lowering shape is concretely specified.

- MINOR: The acceptance criteria are phrased as behavioral-tier checks (build +
  vet), which are directly translatable into test assertions via
  `corpus.RunCompile`.

## Assumptions

- Resolved facts come from `sema.Info` (Structs + FromRegistry), keyed by sema's
  canonical type strings.
- The `...derive(src)` spread is recognized inside the derive body's returned
  composite literal (the other SpreadElement case alongside `...defaults`).

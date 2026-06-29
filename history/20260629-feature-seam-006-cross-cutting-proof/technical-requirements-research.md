# SEAM-006 Technical Requirements / Research

## Gates (verifyCommands)

- `task check` (go vet + full `go test ./...`, includes selfhost port gate + corpus tiers)
- `task build`
- `task fixpoint` (stage1 == stage2 byte-identical self-host proof)

## Method

- Quantify from the real tree via grep over `selfhost/**/*.goal`:
  - `match {` / type-pattern arms vs remaining plain `switch ... .(type)`
  - `enum ` declarations (FuncMod, ChanDir, Mode, Severity); confirm token.Kind/litClass kept iota
  - `Result[` exported/interface signatures
- `goal fix` over the whole selfhost tree -> confirm no residual result-sig suggestions.

## META-finding to record

The deep idioms were blocked not just by per-package audit scope but by MISSING
compiler features. Four new capabilities were built in this PRD:
- SEAM-CAP: cross-package enum-match lowering (.go-defining-package case)
- SEAM-CAP-2: cross-.goal-package enum/sema-fact propagation during self-host build
- SEAM-CAP-3a-d: sealed-interface type-pattern match (method-sig preservation,
  same-package match, cross-.goal-package match, nested hierarchies)

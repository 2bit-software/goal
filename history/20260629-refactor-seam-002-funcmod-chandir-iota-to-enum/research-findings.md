# Research findings — SEAM-002

This is an internal, self-contained transpiler change; no external/web research
applies. Findings are from reading the codebase and the SEAM-CAP/CAP-2 ledger.

## Summary

The two prerequisites that twice BLOCKED SEAM-002 are now landed:

- SEAM-CAP — backend lowers `match` over an enum defined in an imported package
  (matchQualifier resolves `pkg.Enum.Variant`; enumOf resolves imported enums).
- SEAM-CAP-2 — sibling-`.goal`-package enums are visible to cross-package
  consumers during the real `goal build ./selfhost` bootstrap (goalForeignDecls
  projects exported enums; enumRef lowers bare cross-package construction).

The exact forms SEAM-002 needs are each proven by an existing fixture:

- value-position cross-package match: `return match m { mood.Mood.Happy => "happy" ... }`
  (internal/backend/testdata/goalenum/use/use.goal).
- statement-position cross-package match: `match l { light.Light.On => println("on") ... }`
  (testdata/package/cross-pkg-enum/use.goal).
- bare cross-package construction: `mood.Mood.Happy` -> `mood.Mood(mood.Mood_Happy{})`.

Qualified pattern/construction spelling is `pkg.Enum.Variant` (same-package:
`Enum.Variant`).

## The one design decision

token.Kind stays iota (numeric-identity: kindNames[k] indexing,
literalBeg<k<literalEnd range arithmetic, contiguous numbering) — AC-1 escape
hatch. FuncMod/ChanDir are closed, unordered, tag-only sets used only via
==/switch/construction — a perfect enum fit.

## The one tension

internal/ast/ast_test.go (the only port-gated test referencing these symbols)
must compile against BOTH Go-iota internal/ast and enum-transpiled selfhost/ast.
Resolution: relocate the FuncMod assertions to a new internal-only
internal/ast/funcmod_test.go (outside the port-gate file list).

## Confidence

High — every required lowering form has a passing fixture, the consumer set is
fully enumerated (9 sites + 1 zero-value fix), and the test divergence has a
clean, precedented resolution (split test files, as the backend port did).

## Open questions

None blocking. Empty/no-op match arms avoided by using match->bool->if for the
control-flow switch in emit.funcDecl.

# Plan Audit: Buildability

## Findings

### MINOR — foreignDecls signature change ripples to one caller
Adding a 4th return value to `foreignDecls` requires updating its sole caller
`EnrichForeign` (same file). Verified there is exactly one caller in each of
internal/sema and selfhost/sema. Low risk, explicitly in the plan.

### MINOR — Enum reconstruction must precede struct keying harmlessly
Variant structs `Name_Variant` will also land in `info.Structs[alias.Name_Variant]`.
This is harmless (no consumer looks up a variant struct by that key for match
lowering). No de-duplication needed.

No CRITICAL or MAJOR findings. Dependency order is a valid topological sort;
file paths verified against the codebase (testdata/package/ exists, corpus
manifest.json exists, internal/backend/testdata/ pattern exists via extpkg).

## Assumptions
- `DefaultResolver` resolves the foreign import module-relatively at transpile
  time (confirmed: foreign-derive case relies on the same mechanism).
- A tag-only enum is sufficient to prove the capability; payload field-set
  reconstruction is deferred (real seam targets are tag-only).

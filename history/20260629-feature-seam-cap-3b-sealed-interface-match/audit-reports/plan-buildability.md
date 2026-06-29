# Plan Buildability Audit — SEAM-CAP-3b

Dependency order (AST -> parser/sema -> backend -> tests -> mirror) is valid with
no forward references. Interface contracts (TypePattern, SealedImpls, isSealedMatch,
sealedMatch) are concrete with real signatures. File paths verified against the
existing tree.

No CRITICAL/MAJOR.

## MINOR
- The PostToolUse `task check` hook will report transient failures during the
  multi-file edit (undefined symbols until the last edit lands) — expected per the
  Codebase Patterns note; only the final green state matters.
- selfhost mirror is real Go compiled by the port gate; an emitter helper added to
  one of internal/selfhost emit must be added to the other or the port test fails.

## Assumptions
- The emitter renders a `case *T:` label via the existing expr renderer (StarExpr
  handled). If not, a tiny type-string render helper is added (mirrors typeString).
- `e.renames` (already used by enum/Result arms) is the binding-rename mechanism for
  the narrowed value; no new emitter state needed.

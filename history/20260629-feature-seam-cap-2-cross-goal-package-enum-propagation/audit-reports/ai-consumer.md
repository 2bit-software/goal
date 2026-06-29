# Audit 2: AI-Consumer Readiness

## Findings

- **MINOR** — Terms (§8.1 form, marker interface, `pkg.Enum_Variant`) are defined by the
  existing SEAM-CAP implementation and DECISIONS.md; an implementer has concrete
  reference shapes (`reconstructForeignEnums`, `matchQualifier`).
- **MINOR** — Data shapes (`sema.Enum`: Name/Variants/VSet/FieldSet) are fixed by the
  existing type; the projection target is unambiguous.
- Acceptance criteria are specific enough to write assertions from (string-contains on
  emitted Go + a build/run reference-switch comparison).

No CRITICAL or MAJOR findings. The spec plus the verified root cause are sufficient to
implement without further clarification.

## Assumptions

- Implementation lives in `internal/sema/foreign.go` (+ selfhost mirror) and
  `internal/backend/emit.go`/`lower.go` (+ selfhost mirror), matching SEAM-CAP's split.
- The proof fixture sits under `internal/backend/testdata/` so module-relative import
  paths resolve via `moduleResolve`.

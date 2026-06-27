# Audit — Completeness

Spec is a behavior-preserving port of an existing, passing in-repo check
(internal/check/fields.go) onto the AST, pinned by 9 golden cases with inline
`// want` markers. Findings:

- MINOR: FR-1 does not state ordering of omitted field names; the AC clarifies
  "in declaration order". Consistent — no change required.
- MINOR: Spec defers qualified/generic struct literals (Out of Scope). The
  corpus has no such case, so this cannot regress the gate. Acceptable.
- MINOR: Positional (unkeyed) construction of a known struct is unspecified.
  goal struct literals are keyed in the corpus; legacy treats an unkeyed known
  struct as all-fields-missing. Edge case absent from the gate; not blocking.

No CRITICAL or MAJOR findings. Acceptance criteria are individually verifiable
against named golden files.

## Assumptions

- "Field-completeness deferred" Warning text and the omission Error wording match
  internal/check/fields.go verbatim (required for `// want` marker parity).
- Data-less variant construction (`Enum.Dot`) parses to a SelectorExpr (not a
  VariantLit) and is therefore trivially complete — confirmed against the parser.
- Reusing sema/check.go `plural`/`pronoun` helpers; adding `quoteJoin`.

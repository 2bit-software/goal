# Audit — AI-Consumer Readiness (US-050)

The spec is implementable without guessing: every data shape (ReferenceParams,
RenameParams, WorkspaceEdit/TextEdit) is either already in protocol.go or
precisely named in technical-requirements-research.md, and the resolution
algorithm reuses named functions from definition.go.

## Findings

- **MINOR** — Occurrence identity. The spec says "every occurrence of the symbol"
  but does not state the identity key. Resolution (in research doc): a symbol key
  of {func,name} / {type,name} / {variant,enum,name}; same-key occurrences are
  the result set. Unambiguous.
- **MINOR** — WorkspaceEdit version. Use the buffer version from `s.buffer(uri)`
  for the version-pinned `documentChanges`, exactly as codeaction.go does.

## Assumptions surfaced

- Single-document scope (explicit in Out of Scope) — no workspace indexing.
- Reference coverage equals go-to-definition coverage (same structural keying),
  so the two features stay consistent by construction.
- Null (not an empty array vs. null distinction) is the failure response, since
  a nil Go slice marshals to JSON null — consistent with sibling handlers.

No CRITICAL or MAJOR findings. Cleared to plan.

# Plan Audit — Coverage (US-050)

Every spec FR and AC traces to a plan element.

| Spec item | Plan element |
|-----------|--------------|
| FR-1 find references | `references` handler + `collectOccurrences` (references.go) |
| FR-2 includeDeclaration | `ReferenceContext` (protocol.go) + isDecl filter in `references` |
| FR-3 rename WorkspaceEdit | `rename` handler -> WorkspaceEdit/TextEdit (references.go) |
| FR-4 variant/type distinctness | `symKey{kind,enum,name}`, variants keyed under enum |
| FR-5 capability advertisement | server.go initialize + protocol.go capability fields |
| AC null fallbacks | best-effort nil returns in both handlers |
| AC test | references_test.go |
| AC build/vet/test | verifyCommands |

No scope creep: plan adds only the two handlers, their shared collector, the
protocol types, and routing. No element lacks a requirement.

No CRITICAL/MAJOR findings.

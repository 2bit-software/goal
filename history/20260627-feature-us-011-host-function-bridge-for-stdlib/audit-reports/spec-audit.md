# Spec Audit — US-011

The spec is a direct restatement of the PRD US-011 acceptance criteria. It is
small, testable, and scoped to the interpreter package.

## Findings

- CRITICAL: none.
- MAJOR: none.
- MINOR: "located" is interpreted as line:col from token.Pos; the spec does not
  mandate a particular location format. Acceptable — token.Pos.String() already
  renders line:col.

## Assumptions

- Errors (fmt.Errorf / errors.New results) are represented as a struct value
  with TypeID "error" and a "message" field. The PRD does not prescribe an error
  encoding for the interpreter; Result/Option tagged-union encodings are
  separate later stories (US-015/US-016), so a lightweight error struct here
  does not conflict with them.
- fmt.Println writes to os.Stdout directly for now; US-023 will route host
  effects through the cap.CapabilitySet. This story only requires resolution to
  a native implementation, not capability mediation.
- "Imported call" means a selector call whose receiver identifier names an
  imported package (from file.Imports) and is not shadowed by a local binding,
  mirroring the existing builtin-shadowing rule.

Recommendation: PASS (only MINOR).

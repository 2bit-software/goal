# AI-Consumer Readiness Audit

## Findings

- The change touches three known sites: parser `parseFuncDecl`, `ast.FuncType`
  struct (+ Walk), and backend `funcSig`. All are concrete and locatable.
- Acceptance criteria are test-writable (parse with no diagnostic, transpile,
  `go build`).

No CRITICAL or MAJOR findings.

## Assumptions

- Backend emits the func type-param list by reusing `e.fieldList(tp, "[", "]")`,
  mirroring the existing TypeSpec emission.
- `ast.FuncType.TypeParams` is nil for non-generic funcs and all func-type
  expressions, preserving current output.

# Verify — Quality — US-032

## Checks
- Error handling: every emit type switch retains its descriptive `default` fail
  arm; `switchStmt` also fails on a non-CaseClause body element. The
  `struct implements` guard is preserved (moved into `structType`). No silent
  fallthrough was introduced.
- Edge cases: tag-less switch (`switch { case cond: }`), switch with init, and
  multi-expression `case e1, e2:` are all handled by `switchStmt`/`caseClause`;
  empty struct / embedded interface elements are handled
  (`structType`/`interfaceType` guard nil field lists and unnamed elements).
- Format-once discipline upheld: emitter writes token-correct Go; `GoFormatter`
  (go/format) normalizes layout. The behavioral tier (real `go build`/`go vet`)
  proves the generated Go is valid, not merely string-checked.
- Tests assert real behavior: `TestASTEngineEmitsSwitch` parses the output with
  go/format (not just substring), and the behavioral test compiles+vets.

## Findings
- No CRITICAL or MAJOR.
- MINOR: interface method emission (`Name(params) results`) is now correct but is
  only indirectly exercised (the fixture declares a one-method interface compiled
  by the behavioral tier); no dedicated unit test isolates it. Acceptable — the
  behavioral tier is the stronger gate.

## Assumptions
- gofmt is the canonical layout authority; the emitter intentionally does not
  pretty-print (spacing/alignment irrelevant pre-format).
- Goal-specific nodes remain unsupported-by-design here (US-033+ lower them); the
  unchanged fail arms are the intended behavior, not a gap.

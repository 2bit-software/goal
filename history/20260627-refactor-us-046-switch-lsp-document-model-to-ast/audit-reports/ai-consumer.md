# AI-Consumer Readiness Audit — US-046

## Findings

- The node-type -> symbol-kind mapping is fully enumerated in
  research-findings.md (EnumDecl->enum, SealedInterfaceDecl->interface,
  TypeSpec alias->class / struct->struct / interface->interface, FuncDecl
  method->method else function). No guessing required.
- Range/selection mapping is concrete: ranges from node Pos()/End() offsets,
  selection from the name Ident. Offset->Position via check.OffsetToPosition.
- The malformed-source contract (empty non-nil slice, no panic) is explicit.
- Acceptance criteria are directly testable and already backed by existing test
  assertions.
- None CRITICAL or MAJOR.

## Assumptions

- `parser.ParseFile` returns an error (not a panic) on malformed source; the
  rewrite returns an empty outline in that case.
- Replacing `scan.Lex` with `lexer.Tokens` in tokenEnds preserves diagnostic
  ranges because a token end = Pos.Offset + len(Lit), and no test asserts a
  diagnostic range produced via the live tokenEnds path.

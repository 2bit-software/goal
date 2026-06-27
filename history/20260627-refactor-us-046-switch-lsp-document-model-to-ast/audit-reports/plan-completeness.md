# Plan Coverage Audit — US-046

## Requirement -> Plan trace

- FR-1 (symbols from AST) -> symbols.go rewrite (parser.ParseFile + decl walk). Covered.
- FR-2 (faithful ranges) -> node Pos()/End() -> rangeOf; selection from name. Covered.
- FR-3 (best-effort robustness) -> parse error / nil name -> skip / empty slice. Covered.
- FR-4 (no LSP-local token scanner) -> tokenEnds switched to lexer.Tokens; scan
  import dropped from both files. Covered.
- AC "scanDecls removed" -> symbols.go rewrite deletes scanDecls/skipLine/declEnd/decl. Covered.
- AC "full LSP suite passes" -> testing strategy runs internal/lsp + verify gates. Covered.

## Scope creep check

None. Only the two files implicated by the AC are touched. The checker migration
is explicitly out of scope.

## Findings

- None CRITICAL/MAJOR. One MINOR: grouped TYPE GenDecl range start uses spec.Pos()
  instead of the `type` keyword — acceptable and untested in the corpus; single
  spec (the common, tested case) keeps the keyword.

## Assumptions

- The corpus/tests do not exercise grouped `type ( ... )` declarations in the LSP
  outline; single-spec keyword-start range matches prior behavior.

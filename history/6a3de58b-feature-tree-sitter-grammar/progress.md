# Progress Log — tree-sitter-goal (M1)

### T001-T008 — grammar.js + scanner
- Status: Complete
- `grammar.js`: full goal grammar — package/import, func/method/from/derive decls,
  type decls, `sealed interface`, `struct implements`, enums with payloads, all
  statement forms (incl. Go `switch`/type-switch/`fallthrough`), expressions
  (selector/call/index/type-assertion/composite-literal with keyed+spread elements/
  unary/binary/postfix-`?`/match), Result/Option generic types, conversions
  (`[]byte(x)`), labeled args, `///` doc comments.
- `src/scanner.c`: external scanner implementing Go automatic-semicolon insertion.
- Conflicts resolved iteratively against `tree-sitter generate`; unnecessary ones pruned.

### T009 — highlight queries
- Status: Complete
- `queries/highlights.scm`: standard captures (@keyword, @type, @type.builtin,
  @constant.builtin, @function, @property, @operator, @comment[.documentation],
  @string[.escape], @number, @namespace). Verified via `tree-sitter query`.

### T010 — corpus tests
- Status: Complete
- `test/corpus/goal.txt`: 9 tests (enum payload, match+rest, Result/`?`, Option,
  sealed/implements, from/derive+spread, defaults spread, doctest, go switch). All pass.

### T011-T012 — verification + docs
- Status: Complete
- **Acceptance gate met**: parses all **99** repo `.goal` files (excluding abandoned
  `features/_cut/`) with **zero ERROR/MISSING nodes**.
- `tree-sitter generate` clean (no conflicts/warnings); `tree-sitter test` 9/9.
- `README.md` written; generated `src/` (parser.c, grammar.json, node-types.json,
  headers) checked in. tree-sitter-cli installed via Homebrew per request.

## Decisions
- From-scratch focused grammar (no tree-sitter-go fork) — validated empirically
  against the corpus; avoids MIT-attribution (repo has no LICENSE).
- `match` removed `unwrap_binding`/`match_statement` duplication — `?` is an
  unwrap_expression in short-var RHS; match is an expression usable as a statement.
- Go `switch` included (goal is a superset of Go) at the user's direction.
- ASI via a minimal external scanner; composite-literal elements are comma-separated
  (no terminators), so stray newlines are `extras`.

## Blockers
- None.

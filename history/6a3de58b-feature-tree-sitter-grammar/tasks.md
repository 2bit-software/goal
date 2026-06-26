# Tasks — tree-sitter-goal (M1)

**Complexity**: Complex (grammar + scanner + queries + tests; iterative against the corpus).

- [ ] T001 `editors/tree-sitter-goal/package.json` + `.gitignore` + `grammar.js` skeleton
  (source_file, package, import); `tree-sitter generate` succeeds.
- [ ] T002 `src/scanner.c` external scanner: automatic statement terminator (Go ASI);
  wire `externals` in grammar.js.
- [ ] T003 Declarations: func (receiver/generics), type (struct + `implements`, interface,
  alias), enum (+ payload), const/var, `from func`, `derive func`, `sealed`.
- [ ] T004 Types: pointer/slice/array/map/qualified/generic; Result/Option.
- [ ] T005 Statements: block, return, if/else, for(+range), go, defer, `:=`, `=`/op-assign,
  expression stmt, assert, match statement, `name := expr?`.
- [ ] T006 Expressions: identifier, selector, call, index, composite literal (keyed +
  spread), unary/binary, postfix `?`, parenthesized, literals.
- [ ] T007 match_expression + match_arm (`pattern => body`, `_` rest).
- [ ] T008 Comments: `//`, `/* */`, `///` doc_comment with `>>>` doctest_marker.
- [ ] T009 `queries/highlights.scm` — standard captures per technical-spec.
- [ ] T010 `test/corpus/*.txt` — corpus tests for enum, match, result/option, `?`, spreads,
  doctest, implements, derive/from.
- [ ] T011 Verify: `tree-sitter generate` clean; parse ALL 103 `.goal` files → zero ERROR
  (fix grammar until clean); `tree-sitter test` passes; `tree-sitter highlight` sample.
- [ ] T012 `README.md` (build/test/editor-wiring notes) + check in generated `src/`.

Traceability: T001-T008 → FR-003/004; T009 → FR-002; T011 → FR-001/005; corpus parse is the
acceptance gate.

# Technical Requirements & Decisions — tree-sitter-goal (M1)

| Decision | Choice | Rationale |
|---|---|---|
| Location | `editors/tree-sitter-goal/` | Beside `editors/vscode/`; the editors home |
| Approach | From-scratch focused grammar | Constrained corpus; no MIT-fork attribution; verifiable |
| Toolchain | Homebrew `tree-sitter-cli` 0.26.9 | User-requested brew over npm/cargo |
| Statement sep | Newline/brace terminated (Go ASI) | goal has no explicit `;` |
| Verification | Parse all 103 corpus files, zero ERROR; corpus tests | Empirical, strong |
| Highlights | Standard captures in `queries/highlights.scm` | Portable across nvim/Helix/Zed/GitHub |

## Files to produce

- `editors/tree-sitter-goal/grammar.js` — the grammar.
- `editors/tree-sitter-goal/queries/highlights.scm` — highlight queries.
- `editors/tree-sitter-goal/test/corpus/*.txt` — tree-sitter corpus tests for goal constructs.
- `editors/tree-sitter-goal/package.json` — metadata + scripts (generate/test) for the
  tree-sitter toolchain; `grammar` field.
- `editors/tree-sitter-goal/README.md` — what it is, how to build/test, editor wiring notes.
- Generated `src/` (parser.c etc.) — checked-in is conventional for tree-sitter grammars so
  consumers don't need the CLI; `.gitignore` keeps node_modules/build out.

## Build/verify commands (add to README; optionally a local Taskfile target)

```
cd editors/tree-sitter-goal
tree-sitter generate
tree-sitter test
tree-sitter parse <all .goal files>   # expect zero ERROR
tree-sitter highlight examples/sample.goal
```

## Risks

- Grammar conflicts (precedence) — resolved iteratively against `tree-sitter generate`.
- Expression/statement coverage gaps vs real Go — caught by parsing the full corpus;
  any uncovered form surfaces as an ERROR node and gets a grammar rule.
- ASI / raw-string edge cases — add a minimal external scanner (`src/scanner.c`) only if
  the pure-DSL grammar can't express them.

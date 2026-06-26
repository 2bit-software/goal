# Reuse Audit — tree-sitter-goal M1

| Planned item | Existing in repo? | Verdict | Action |
|---|---|---|---|
| tree-sitter grammar / parser | none (no tree-sitter, no other grammar) | CREATE NEW | Build in `editors/tree-sitter-goal/` |
| goal token/keyword set | encoded in `internal/lex`/passes and the L1 TextMate grammar | RELATED | Reuse the *known set* (enum/match/sealed/…); no code to import — tree-sitter grammar is independent of the Go lexer |
| highlight scope intent | `editors/vscode/syntaxes/goal.tmLanguage.json` (L1) | RELATED | Mirror the same token→meaning decisions in `highlights.scm`; different format, not shared code |
| corpus for validation | 103 `.goal` files under `features/`, `testdata/`, `editors/vscode/examples/` | REUSE (as test input) | Parse them all as the acceptance gate |
| build CLI | none | CREATE NEW | Homebrew `tree-sitter-cli` (installed) |

Nothing duplicates existing code. The grammar is a standalone, parallel front-end (by
design — tree-sitter is a separate parser from goal's Go lexer). The only reuse is the
*corpus as test input* and the *token-meaning decisions* already settled in Layer 1.

# Initiative: vscode-highlighting

**Type**: feature
**Status**: complete
**Created**: 2026-06-25
**ID**: 6a3dd48c-feature-vscode-highlighting

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | complete | 2026-06-25 |
| plan | plan | complete | 2026-06-25 |
| tasks | tasks | complete | 2026-06-25 |
| implement | implement | complete | 2026-06-25 |

## Description

Layer 1 editor integration for the Goal language: a VS Code extension providing
TextMate (regex) syntax highlighting for `.goal` files, plus language configuration
(comment toggling, bracket matching, auto-closing pairs). No language server.

Lives in `editors/vscode/` (new convention; no prior editor tooling existed).
Built on `main` per user request ("don't branch"). Run in AutoMode.

## Goals

- Color `.goal` files in VS Code, recognizing Goal's deltas over Go: `enum`,
  `sealed`, `match`, `assert`, `from`, `derive`, `implements`, `Result`/`Option`
  and their constructors, the `=>` match arrow, postfix `?` unwrap, `...defaults`
  / `...derive` spreads, and `///` doctest comments with `>>>` markers.
- Self-contained grammar written from scratch (no third-party grammar vendored,
  avoiding license entanglement — the repo has no LICENSE file).

## Progress

- Scaffolded extension: `package.json`, `language-configuration.json`,
  `.vscodeignore`, `.gitignore`, `README.md`, `examples/sample.goal`.
- Authored `syntaxes/goal.tmLanguage.json` (Go core + Goal-specific tokens).
- Verified with VS Code's own tokenizer (`vscode-textmate` + `vscode-oniguruma`):
  `test/tokenize.test.mjs` asserts 19 representative token→scope mappings — all pass.
- Confirmed the extension packages cleanly to a `.vsix` via `vsce`.

## Notes / follow-ups

- The initiative tooling created a `feat/vscode-highlighting` branch and committed
  a pre-existing staged CLI change onto it. Honored "don't branch": restored that
  change to `main`'s working tree (unchanged) and deleted the stray branch.
- Layer 2 (LSP semantic highlighting over the existing analyze/check passes) and
  Layer 3 (tree-sitter) remain future work, per ROADMAP Phase D.

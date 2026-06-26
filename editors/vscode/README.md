# Goal — VS Code language support

Language support for the Goal language — a thin, orthogonal dialect of Go.

- **Layer 1** — syntax highlighting (TextMate grammar): coloring for `.goal` files
  plus comment toggling, bracket matching, and auto-closing pairs.
- **Layer 2 (milestone 1)** — **inline diagnostics**: a language client launches
  `goal lsp` and shows the language's check violations (non-exhaustive `match`,
  must-use `Result`, no-zero-value, etc.) as squiggles, live as you edit. Hover,
  go-to-definition, and semantic highlighting are future milestones.

## Requirements for diagnostics

The diagnostics feature runs the `goal` binary as a language server (`goal lsp`).
Install it so it is on your `PATH`:

```sh
go install ./cmd/goal   # from the repo root (or: task install)
```

Then set `goal.lsp.path` if your binary is elsewhere. Disable the server with
`goal.lsp.enable: false`. Highlighting (Layer 1) works with no binary.

## What it highlights

On top of standard Go syntax, the grammar adds Goal-specific tokens:

| Token | Scope |
|---|---|
| `enum`, `sealed`, `from`, `derive` | `storage.type.goal` |
| `implements` | `storage.modifier.goal` |
| `match`, `assert` | `keyword.control.goal` |
| `Result`, `Option` | `support.type.goal` |
| `Ok` / `Err` / `Some` / `None` and `Enum.Variant` | `variable.other.enummember.goal` |
| `=>` (match arm) | `keyword.operator.arrow.goal` |
| `?` (postfix unwrap) | `keyword.operator.unwrap.goal` |
| `...defaults`, `...derive` | `keyword.operator.spread.goal` |
| `///` doc comments and `>>>` doctest markers | `comment.line.documentation.goal`, `keyword.control.doctest.goal` |

## Develop

Open this folder in VS Code and press <kbd>F5</kbd> to launch an Extension
Development Host. Open any `.goal` file (e.g. `examples/sample.goal`) to see
highlighting. Use **Developer: Inspect Editor Tokens and Scopes** from the command
palette to debug which scope each token receives.

## Test

The grammar is verified with VS Code's own tokenizer engine:

```sh
npm install
npm test
```

`test/tokenize.test.mjs` loads the grammar via `vscode-textmate` + `vscode-oniguruma`
and asserts that representative tokens get the expected scopes.

## Package / install

```sh
npm install
npm run package        # produces goal-lang-<version>.vsix
npm run install-local  # package + install into your local VS Code
```

## License

This grammar was written from scratch for this repository (no third-party grammar
was vendored). It inherits the repository's license.

# Goal — VS Code language support

Syntax highlighting (TextMate grammar) for the Goal language —
a thin, orthogonal dialect of Go. Provides coloring for `.goal` files plus comment
toggling, bracket matching, and auto-closing pairs.

This is the **Layer 1** editor integration: a regex-based TextMate grammar. It does
not require a running language server. Semantic highlighting / LSP (Layer 2) is a
future addition tracked in the project ROADMAP.

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

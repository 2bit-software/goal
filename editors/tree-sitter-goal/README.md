# tree-sitter-goal

A [tree-sitter](https://tree-sitter.github.io/) grammar for the **Goal** language
(`.goal`) — a thin dialect of Go. This is the **Layer 3** editor integration: a
structural parser plus highlight queries that any tree-sitter host can load
(Neovim/nvim-treesitter, Helix, Zed, Emacs, GitHub) with **no language server**.

## What it covers

Go's commonly-used surface as it appears in goal code, plus goal's additions:

- `enum` declarations with payloads (`Active { since: Time }`)
- `match` expressions with `=>` arms and the `_` rest pattern
- `sealed interface` declarations and `struct implements I { … }`
- `from func` / `derive func` conversions, and `...defaults` / `...derive` spreads
- `Result[T, error]` / `Option[T]` types and their constructors
- the postfix `?` unwrap, labeled call arguments (`Status.Active(since: now())`)
- `///` doc comments (carrying `>>>` doctests)
- plain Go: `switch`/`case`/`default`/`fallthrough`, type switches/assertions, etc.

Statement termination follows Go's automatic semicolon insertion, implemented by a
small external scanner (`src/scanner.c`).

## Status / scope

The grammar parses the entire repository `.goal` corpus (99 files) with **zero
ERROR nodes**. It targets goal's real surface; exotic Go forms not present in goal
code are out of scope for this milestone. The abandoned `features/_cut/` examples
(e.g. `pure func`) are intentionally excluded.

## Build & test

Requires the tree-sitter CLI (`brew install tree-sitter-cli`) and a C compiler.

```sh
tree-sitter generate          # regenerate src/ from grammar.js
tree-sitter test              # run the corpus tests in test/corpus/
tree-sitter parse path.goal   # parse a file (exit non-zero on ERROR)
tree-sitter query queries/highlights.scm path.goal   # inspect highlight captures
```

The generated parser (`src/parser.c`, `src/grammar.json`, `src/node-types.json`,
`src/tree_sitter/`) is checked in, so consumers don't need to run `generate`.

## Editor wiring (examples)

- **Neovim / nvim-treesitter**: register a custom parser pointing at this directory
  and copy `queries/highlights.scm` into `queries/goal/`.
- **Helix / Zed**: add a language entry referencing this grammar's git source and the
  `queries/` directory.
- **VS Code**: the [`editors/vscode`](../vscode) extension provides highlighting via a
  TextMate grammar (Layer 1) and diagnostics via the language server (Layer 2);
  wiring this tree-sitter grammar into VS Code's tree-sitter API is a future step.

## License

The grammar was written from scratch for this repository (no third-party grammar was
vendored). It inherits the repository's license.

# Research Summary — tree-sitter grammar for goal

## Toolchain (verified available)

- `tree-sitter` CLI **0.26.9** via Homebrew (`brew install tree-sitter-cli`).
- Apple clang 21 (compiles the generated parser), node 25, cargo 1.94 (all present).
- Build/verify loop: `tree-sitter generate` → `tree-sitter parse <files>` → `tree-sitter test`.

## Corpus (the verification target)

- **103 `.goal` files, ~2253 lines** under `features/`, `testdata/`, `editors/vscode/examples/`.
- Construct frequency (drives grammar priorities): func 223, return 184, Result 157,
  type 144, struct 113, package 107, enum 52, Option 45, match 40, if 31, derive 31,
  from 29, implements 26, assert 25, interface 20, import 13, map 10, for 9, sealed 6,
  var/const 5/4, go 3, defer 2, range 1.
- Implication: goal code is a **constrained subset of Go** + goal extensions. A focused
  grammar that parses this corpus cleanly covers real-world goal.

## goal's deltas over Go (first-class nodes needed)

From the lexer/passes (authoritative, established in prior layers):
- Keywords: `enum`, `sealed`, `match`, `assert`, `from`, `derive`, `implements`.
- Types: `Result[T, E]`, `Option[T]` (bracketed generics, always explicit args).
- Constructors: `Result.Ok/Err`, `Option.Some/None`, `Enum.Variant[(args)]` (qualified).
- Operators: `=>` (match arm), postfix `?` (unwrap), `...defaults` / `...derive` (spreads).
- Match: `match expr { Pat => body ... }`, `_` rest arm; arms use `=>` not `:`.
- Enum decl: `enum E { Variant | Variant { field: Type } ... }`.
- Comments: Go `//`, `/* */`; plus `///` doc comments containing `>>>` doctest markers.
- Struct impl clause: `type T struct implements I { ... }`.
- Conversions: `from func ...`, `derive func ...`.

## Lexical structure (inherited from Go)

- Automatic semicolon insertion (no explicit `;` in source) — the grammar must treat
  newlines as statement terminators (tree-sitter handles this with an external scanner
  or, for a focused grammar, by making statements newline/brace-delimited).
- String literals: `"..."` (interpreted, with escapes), `` `...` `` (raw), `'...'` (rune).
- Numbers: Go int/float/hex/oct/bin/imaginary.

## Design decision

**From-scratch focused grammar** (not a tree-sitter-go fork):
- The corpus is small and constrained; a fork drags a ~2k-line grammar + C scanner and
  raises the same MIT-attribution issue the repo has no LICENSE for (Layer 1 precedent).
- Validation is empirical and strong: parse all 103 corpus files with zero ERROR nodes.
- Statements are newline/brace terminated; a small external scanner handles raw strings
  and line-terminator semantics if the pure-DSL approach proves insufficient.

## Highlight queries

`queries/highlights.scm` maps nodes → standard captures (`@keyword`, `@type`,
`@type.builtin`, `@constant`, `@function`, `@variable.member` for enum variants,
`@operator`, `@comment`, `@comment.documentation`, `@string`, `@number`). Standard
captures make the grammar render under nvim-treesitter, Helix, Zed, and GitHub.

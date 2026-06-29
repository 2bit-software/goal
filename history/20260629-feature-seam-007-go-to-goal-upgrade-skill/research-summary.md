# Research Summary — SEAM-007

## goal syntax (verified against the selfhost tree + docs/by-example.md)

- enum: `enum Name { Variant; Variant { x: T } }` — closed sum, lowers to a
  sealed interface + per-variant struct + `isName()` marker. Zero value is nil.
- match: `match e { Name.Variant => ...; _ => ... }` — exhaustive; `_` opts out.
  Lowers to a Go type switch. Value-position match lowers only as `x := match`,
  `var x T = match`, or `return match`.
- sealed interface: `sealed interface Node { Pos() token.Pos; End() token.Pos }`;
  embedding for hierarchies: `sealed interface Expr { Node }`.
- implements: `type T struct implements Expr { ... }` — concrete type joins a
  sealed interface; the embedding cascade emits markers for embedded interfaces.
- Result/Option: `func f() Result[T, error] { return Result.Err(e); return
  Result.Ok(v) }`; postfix `?`: `x := f()?` (Err/None -> early return).

## goal CLI (cmd/goal/main.go)

- `goal fix [-inplace] [path]` — autofixer. Without `-inplace` writes the fixed
  source to STDOUT and diagnostic reports to STDERR (`path:line: level: [rule]
  msg`). With `-inplace` it rewrites changed files and prints `fixed <path>`.
  Accepts a single `.goal` file or a directory of `.goal` packages.
- `goal build [path]`, `goal check [path]`, `goal run`, `goal fmt [-w]`,
  `goal ai [section]` (the in-binary knowledge guide; `docs/by-example.md` is the
  human reference).

## Reserved words

`match`, `enum`, `assert` are fully reserved beyond Go keywords.
`implements`/`sealed`/`from`/`derive` are contextual (not reserved).
`enumOf`/`enumName`/`.Enum` are fine — only the bare word collides. Go source
using a reserved word as a bare identifier must be renamed before parsing.

## Idiom catalogue (distilled from SEAM-002..006, recorded in DECISIONS.md)

| Idiom | Convert when | Keep (documented non-fit) |
|-------|--------------|---------------------------|
| iota const block -> enum | closed set, no numeric/wire/ordering use | array index, wire value, ordered compare (e.g. token.Kind, litClass) |
| type-switch over closed scrutinee -> sealed interface + match | scrutinee is a closed set of concrete types | open/extensible interface |
| method on would-be-enum -> free label fn | type becomes an enum | n/a (mechanical) |
| exported fallible (T,error) -> Result/? | pure single-value propagation | accumulator ([]error), multi-value return, comma-ok control flow |

Cross-package idioms rely on whole-program enum/sealed-fact propagation
(SEAM-CAP / CAP-2 / CAP-3c). Within a single-file or single-package scope, a
consumer living in another package is itself a documented non-fit ("cross-package
consumer not in scope").

## Skill location

No pre-existing project skill. `.claude/skills/<name>/SKILL.md` is the standard
Claude Code project skill dir. Match the bootstrap-project SKILL.md frontmatter
format (`name`, `description`, `compatibility` + markdown body); supporting files
under `scripts/` and `references/`.

## No-regression

The skill lives under `.claude/skills/` — invisible to the Go toolchain and to
`project.Discover` over `selfhost/`, so it cannot affect `task check/build/
fixpoint`. Dogfood on a COPY in a scratch dir, never on tree source.

## Confidence

High — syntax and CLI behavior verified directly against source; idiom catalogue
matches the completed SEAM stories' DECISIONS records.

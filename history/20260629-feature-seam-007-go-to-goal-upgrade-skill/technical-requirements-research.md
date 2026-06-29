# Technical Requirements / Research — SEAM-007

## Skill location / convention

The repo has empty `.morse-code/` and `.claude/` dirs and no pre-existing
project skill. `.claude/skills/<name>/SKILL.md` is the standard Claude Code
project skill location. Match the house SKILL.md frontmatter format used by the
global `bootstrap-project` skill: YAML frontmatter (`name`, `description`,
optional `compatibility`) followed by markdown body. Supporting scripts/refs go
in `scripts/` and `references/` subdirs alongside SKILL.md.

## goal CLI surface (cmd/goal/main.go)

- `goal build [--emit[=dir]] [path]` — transpile + build a package dir.
- `goal run [--engine=ast|interp] [--emit[=dir]] [path] [args...]`.
- `goal check [path]` — checker only.
- `goal fix [-inplace] [path]` — the mechanical autofixer (Step 2).
- `goal fmt [-w] [path]`.

`goal fix` is the autofixer that converts `(T,error)` + manual `if err != nil`
propagation to `Result`/`?`. By default it prints to stdout / reports; `-inplace`
rewrites. It does NOT do the manual idioms (enum/sealed-match) — that is Step 3.

## Reserved words

goal reserves three words beyond Go keywords: `match`, `enum`, `assert`.
`implements`/`sealed`/`from`/`derive` are contextual keywords (not reserved).
Go using any reserved word as a bare identifier must be renamed before parsing.
`enumOf`/`enumName`/`.Enum` are fine — only the bare word collides.

## Idiom catalogue source

The manual idiom catalogue is distilled from the completed SEAM stories:
- SEAM-002: token.Kind/FuncMod/ChanDir iota -> enum (token.Kind+litClass kept
  iota for numeric/wire/ordering identity; FuncMod/ChanDir converted).
- SEAM-003: Mode/Severity iota -> enum; Severity.String() method -> free
  SeverityLabel func (enum lowers to sealed interface; Go forbids method on it);
  enum zero is nil so set Mode.ModeNone explicitly.
- SEAM-004: seal ast.Node/Expr/Stmt/Decl/Spec; ~43 type-switches -> exhaustive
  match; nested sealed hierarchy via embedding cascade (CAP-3d).
- SEAM-005: pure-propagation fallible API -> Result/?; accumulators (EnrichForeign
  []error), multi-value returns, comma-ok kept as documented non-fits.
- SEAM-CAP / CAP-2 / CAP-3c: cross-package enum-match + sealed-match lowering and
  whole-program enum/sealed-fact propagation underpin cross-package idioms.

## Dogfood candidate

Pick a small self-contained Go package/file (e.g. under attic/ or a tiny helper,
or a purpose-made tiny example). Run the pipeline on a COPY in a scratch dir,
prove `goal build` produces buildable idiomatic goal. Document the example in the
skill's references.

## No-regression gate

`task check`, `task build`, `task fixpoint` must stay green. The skill is
docs+assets only (lives under .claude/skills/, invisible to the Go toolchain and
to project.Discover over selfhost/), so it cannot affect those gates — but run
them to confirm.

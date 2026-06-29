# SEAM-002: FuncMod & ChanDir iota -> goal enum — Business Specification

## Overview

The self-hosted goal compiler defines several small iota types. Two of them —
`FuncMod` (a FuncDecl's `from`/`derive` modifier) and `ChanDir` (a channel's
direction) — are closed, unordered, tag-only value sets consumed only through
equality comparisons, switches, and construction. These should read as
idiomatic goal `enum`s, with every cross-package consumer expressed as exhaustive
`match`. A third type, `token.Kind`, is deliberately kept as iota because its
values carry numeric identity (array indices, range predicates, contiguous
numbering); converting it would change behavior, not improve idiom.

This is a SEAM story under the relaxed equivalence gate (DECISIONS.md "Seam
methodology"): emitted Go is allowed to change, and behavior preservation is
re-proven by self-host fixpoint self-consistency, the corpus behavioral tier, and
reviewed regeneration of any oracle/golden whose bytes legitimately move.

## Functional Requirements

### FR-1: FuncMod is a goal enum
`FuncMod` SHALL be a goal `enum` with tag-only (payloadless) variants
`FuncPlain`, `FuncFrom`, `FuncDerive` (replacing `type FuncMod int` + iota). The
declaration form is newline-separated tag-only variants (DECISIONS.md §01-enums
"Variant separator: newline-separated, no trailing punctuation"):

```
enum FuncMod {
    FuncPlain
    FuncFrom
    FuncDerive
}
```

The existing per-constant `///`/`//` doc text SHALL be preserved as comments
attached to the enum / its variants.

### FR-2: ChanDir is a goal enum
`ChanDir` SHALL be a goal `enum` with tag-only variants `SendRecv`, `SendOnly`,
`RecvOnly` (replacing `type ChanDir int` + iota), same declaration form as FR-1,
doc text preserved.

### FR-3: All consumers (cross- and same-package) use match
Every `==`/`!=` comparison and every value-`switch` over a `FuncMod` or
`ChanDir` value across `selfhost/{ast,sema,backend,parser}` SHALL be expressed
as an exhaustive `match` — one arm per variant, NO wildcard `_` arm (the
compile-time-error-on-new-variant guarantee depends on the absence of a
catch-all). This INCLUDES the one same-package site
(`selfhost/ast/ast.goal:170`). A tagless `switch {}` whose cases are booleans
(not the enum value itself, e.g. `resolve.goal:216` `switch { case <bool>: }`)
is NOT a switch over the enum and stays a `switch {}`; only the inner `==`
comparison inside its case converts to a match-bound bool.

### FR-4: Construction uses qualified variant syntax
Every site that produces a `FuncMod`/`ChanDir` value SHALL construct it via the
qualified variant form: cross-package `ast.FuncMod.FuncFrom` /
`ast.ChanDir.SendRecv`; same-package (inside `selfhost/ast`) `FuncMod.FuncPlain`
(no `ast.` prefix). This SUPERSEDES the old unqualified spelling
(`ast.FuncFrom`, `ast.SendRecv`) that the consumer inventory shows for those
same lines. The parser SHALL set a plain FuncDecl's modifier to
`ast.FuncMod.FuncPlain` explicitly at `parser.goal:365` (the enum zero value is
nil, not FuncPlain; the from/derive path in `goal_stmt.goal` overwrites it
afterward). `ChanType` needs no analogous fix because its sole constructor
(`parser.goal:511`) already sets `Dir` explicitly.

### FR-5: token.Kind stays iota (documented refusal)
`token.Kind` SHALL remain `type Kind int` + iota, with the numeric-identity
justification recorded in DECISIONS.md per AC-1's escape hatch.

### FR-6: Behavior preserved + zero-value invariant
The compiled compiler SHALL behave identically: same self-host output, same
corpus behavioral results. Because an enum's zero value is `nil` (no variant) —
unlike iota, where `FuncMod`'s zero was `FuncPlain` and `ChanDir`'s was
`SendRecv` — every construction site SHALL set `Mod`/`Dir` to an explicit
variant, so no `FuncDecl`/`ChanType` ever carries a `nil` modifier/direction
into a `match`. The two construction owners are `parser.goal` (FuncDecl: covered
by FR-4's explicit `FuncPlain`; ChanType: already explicit). This invariant is
what makes `match` total at run time.

## Acceptance Criteria

- [ ] `selfhost/ast` declares `FuncMod` and `ChanDir` as goal enums.
- [ ] No plain `switch`/`==`/`!=` over `FuncMod` or `ChanDir` remains in
  `selfhost/{ast,sema,backend,parser}`; each is a `match`.
- [ ] A plain (non-from/derive) function still resolves as `FuncPlain` after
  parsing, and every `ChanType` carries an explicit `Dir` (zero-value gap
  closed for both enums). Enforced end-to-end by `task fixpoint` (a nil modifier
  would diverge or fault the bootstrap) plus the parser/sema/backend behavioral
  port gates; no new unit test is added.
- [ ] `token.Kind` remains iota; DECISIONS.md records the refusal reason.
- [ ] `task check` is green (after the `internal/ast` FuncMod-oracle relocation).
- [ ] `task build` is green (both binaries).
- [ ] `task fixpoint` reports FIXPOINT OK (stage1 == stage2).
- [ ] The corpus behavioral tier
  (`internal/corpus` `TestASTEngineWholeCorpusBehavioralGate`, run within
  `task check`) is unchanged.
- [ ] DECISIONS.md records the FuncMod/ChanDir conversion and the token.Kind refusal.

## User Interactions

None (compiler-internal). The observable contract is the transpiler's output and
the test suite.

## Error Handling

Exhaustive `match` replaces the prior `default`/fallthrough arms. A NEW variant
added later with no arm is a compile-time exhaustiveness error (this is the idiom
gain). Note the precise semantics: the three named variants are covered at
compile time, but the enum's zero value `nil` is NOT a variant and is therefore
NOT a compile-time-covered case — a `nil` modifier/direction reaching a `match`
would be a run-time fault. Safety thus rests on the FR-6 construction invariant
(no site ever leaves `Mod`/`Dir` nil), not on the match alone. The dropped
`default: e.fail("unsupported func modifier")` defensive arm at `emit.goal:361`
was unreachable for the three valid variants and is intentionally removed.

## Out of Scope

- `sema.Mode` / `sema.Severity` (SEAM-003).
- Sealing `ast.Node/Expr/Stmt/Decl/Spec` and converting type-switches (SEAM-004).
- Any change to the bootstrap reference compiler's own Go AST representation in
  `internal/ast` (it stays Go iota; only its FuncMod oracle test is relocated).
- Converting `token.Kind` (deliberate, documented refusal).

## Open Questions

None blocking. All required lowering forms (value-/statement-position
cross-package match, bare cross-package construction) are proven by existing
SEAM-CAP/CAP-2 fixtures.

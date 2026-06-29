# AI-Consumer Readiness Audit — SEAM-002 (FuncMod & ChanDir iota → goal enum)

**Question audited:** Could an AI agent implement this spec without guessing?

**Scope of audit:** `business-spec.md`, `research-findings.md`, `technical-requirements-research.md`,
cross-checked against the actual source (`selfhost/{ast,sema,backend,parser}`, `internal/ast/ast_test.go`,
`internal/selfhost/port_test.go`).

**Verdict:** Mostly implementable. The consumer-site inventory is accurate and exhaustive (verified
against the tree), and the lowering forms are precedented. But several load-bearing terms are
undefined or under-pointed, the enum *declaration* syntax is never shown, and a couple of acceptance
criteria lack an identified test vehicle. An average agent would proceed but make 4–6 inferences,
two of which (the enum declaration form and the "port-gate list" location) are real guesses with a
material chance of being wrong.

---

## CRITICAL

### C-1. The goal `enum` declaration syntax is never shown — the core deliverable's exact form must be guessed
FR-1/FR-2 say `FuncMod`/`ChanDir` "SHALL be a goal `enum` with variants ...", and the AC says
`selfhost/ast` "declares `FuncMod` and `ChanDir` as goal enums." But **no file shows what a goal
`enum` declaration actually looks like.** The spec gives the *construction* and *pattern* spelling
(`pkg.Enum.Variant`, research-findings.md line 25–26) and the lowered Go (`mood.Mood(mood.Mood_Happy{})`),
but never the source-level declaration: is it `enum FuncMod { FuncPlain, FuncFrom, FuncDerive }`? Does
each variant get its own line? Are payloadless ("tag-only") variants written differently from
payload variants? Does the enum carry the existing per-variant doc comments
(`selfhost/ast/goal_decl.goal:23–28` and `ast.goal:675–680` currently have `///`-style docs on each
constant)?

This is the single most important artifact the story produces, and an implementer must reverse-engineer
its syntax from the referenced fixtures (`internal/backend/testdata/goalenum/use/use.goal`,
`testdata/package/cross-pkg-enum/use.goal`) rather than read it in the spec. The fixtures are cited
for *match* forms, not for the *declaration* form. **Quote (business-spec FR-1):** "`FuncMod` SHALL be
a goal `enum` with variants `FuncPlain`, `FuncFrom`, `FuncDerive` (replacing `type FuncMod int` +
iota)." — the replacement target is named but not spelled.

*Why CRITICAL:* you cannot write the central declaration without opening unrelated fixture files and
inferring grammar; whether doc comments survive is unspecified and affects the diff reviewers will gate on.

---

## MAJOR

### M-1. "Port-gate" / "port-gate test list" is undefined and its location is never given
The entire test-divergence resolution hinges on this term, used 5+ times
(research-findings.md line 36–39; technical-requirements-research.md line 36–50: "relocate the FuncMod
assertions into a new internal-only `internal/ast/funcmod_test.go` (NOT in the port-gate test list)").
The term **does not appear anywhere in the codebase** (grep returns nothing outside `history/`). An
implementer is told to keep a file "out of the port-gate list" without being told where that list
lives or what it is.

Verified ground truth (which the spec omits): the list is a hard-coded `[]string` of `_test.go`
paths passed to `selfhost.BuildAndTest` in **`internal/selfhost/port_test.go`** — line 142 is
`BuildAndTest("selfhost/ast", astPkg, []string{"../ast/ast_test.go"}, deps)`. "Not in the port-gate
list" concretely means: do not add `../ast/funcmod_test.go` to that slice. Without this pointer the
agent must discover the mechanism by reading `internal/selfhost/`.

*Why MAJOR:* the resolution is correct and precedented, but the agent has to rediscover the
mechanism the spec assumes it already knows. A wrong guess (e.g. adding the new file to the slice, or
editing the wrong list) breaks the bootstrap behavioral gate.

### M-2. Match arm *bodies* for the existing `default` cases are not enumerated; "exhaustive" forces a non-obvious mapping
Two of the four switches map a variant to a `default` arm, not an explicit case:

- `selfhost/sema/resolve.goal:458` — `SendRecv` falls into `default: return "chan " + ...` (only
  `RecvOnly`/`SendOnly` are explicit).
- `selfhost/backend/emit.goal:2309` — `SendRecv` falls into `default: e.p("chan ")`.
- `selfhost/backend/emit.goal:361` — groups `FuncPlain, FuncFrom` (no-op), `FuncDerive` (return),
  and a `default: e.fail("unsupported func modifier")`.

FR-3 + Error Handling require an **exhaustive** match with "an explicit arm" per variant. So the
implementer must (a) realize the bidirectional `SendRecv` arm body is the former `default` body, and
(b) decide what happens to the `default: e.fail(...)` safety arm (it must vanish, per the Error
Handling section). The spec states the *principle* ("exhaustiveness replaces the prior
default/fallthrough arms") but never enumerates the resulting arm→body mapping. This is inferable from
reading the code but is left implicit; the spec gives line numbers without the target arm bodies.

*Why MAJOR:* correctness of the conversion depends on the agent reading the `default` body and
re-homing it under the right variant; nothing in the spec states the mapping, so it is an inference,
not a transcription.

### M-3. Internal inconsistency in construction spelling: FR-4 vs. the "Consumers to convert" list
FR-4 mandates qualified construction: "Every site that produces a `FuncMod`/`ChanDir` value SHALL
construct it via the qualified variant form (`ast.FuncMod.FuncFrom`, `ast.ChanDir.SendRecv`, etc.)."
But the authoritative site list in technical-requirements-research.md lines 31–32 still spells them
the **old** way:
- line 31: "construction `ast.FuncFrom`/`ast.FuncDerive`"
- line 32: "construction `ast.SendRecv`/`RecvOnly`/`SendOnly`"

An agent following the consumer list literally would emit the unqualified (pre-conversion) spelling
and violate FR-4. The two documents disagree on the exact target tokens for the same lines
(`parser.goal:226,229,511,515,520`).

*Why MAJOR:* two parts of the same spec prescribe different output tokens for the same edits; the
agent must notice the conflict and pick FR-4, which is a judgment call the spec should have removed.

### M-4. AC "a plain function still resolves as `FuncPlain` after parsing" has no identified test vehicle
The AC "A plain (non-from/derive) function still resolves as `FuncPlain` after parsing (zero-value gap
closed)" is the only AC that asserts new *runtime* behavior, yet no test is named to enforce it. In
Go-iota `internal/parser`, the zero value already *is* `FuncPlain` (0), so `internal/parser`'s
`parser_test.go` (the only port-gated parser test, `port_test.go:232`) passes regardless of the fix.
The fix only matters for the enum-transpiled `selfhost/parser` (enum zero is nil). I confirmed the
fix site: `selfhost/parser/parser.goal:365` `fd := &ast.FuncDecl{}` (and `goal_stmt.goal:49`
`parseFuncDecl()` then `fd.Mod = mod`, so the from/derive path overwrites the explicit `FuncPlain` —
correct). But **nothing in the spec identifies an assertion that would fail if the fix were omitted**;
the only safety net is `task fixpoint` catching an emission divergence. The AC reads like a unit-test
target but maps to no test.

*Why MAJOR:* an AC phrased as a checkable assertion ("resolves as FuncPlain") cannot be turned into a
direct test from the spec; the agent must either invent a test or rely on fixpoint, and the spec
doesn't say which.

---

## MINOR

### m-1. `resolve.goal:218` is a tagless `switch {}`, not a simple boolean — integration not shown
Technical-requirements-research.md line 26 describes `selfhost/sema/resolve.goal:218` as
`d.Mod == ast.FuncFrom || d.Mod == ast.FuncDerive` and the Conversion-idiom section says to "Replace
`==`/`!=` boolean uses with a value-position `match` bound to a bool." But the site is the first
`case` of a multi-arm **tagless** `switch { case <bool>: ...; case d.Recv != nil: ...; default: ... }`
(`resolve.goal:216–222`). The idiom (bind a bool via match, use it in the case) works, but the spec
classifies it as a plain boolean and doesn't show the tagless-switch integration. Inferable; noting
for completeness.

### m-2. Same-package vs. cross-package pattern spelling stated only in research, not the normative spec
The same-package pattern form (`FuncMod.FuncPlain` for `selfhost/ast/ast.goal:170`'s `d.Mod != FuncPlain`)
vs. cross-package (`ast.FuncMod.FuncPlain`) is given only in research-findings.md line 26 as a
parenthetical. The business-spec FR-4 examples are all cross-package (`ast.FuncMod.FuncFrom`). An agent
editing the one same-package site (`ast.goal:170`) must remember to drop the `ast.` qualifier. Minor
because the rule is stated, just not in the normative FR.

### m-3. `parseModFuncDecl(mod ast.FuncMod)` signature use of the type is unmentioned
`selfhost/parser/goal_stmt.goal:47` declares `func (p *parser) parseModFuncDecl(mod ast.FuncMod)`.
This is a type reference (parameter), not a comparison/switch/construction, so it needs no change — but
the spec's exhaustive-looking consumer list never mentions it, which could make an agent second-guess
whether the count ("9 sites + 1 zero-value fix") is complete. I verified the count is right and this
site legitimately needs no edit; the omission is harmless but unexplained.

### m-4. "exhaustive match → compile-time error on unhandled value" assumes a goal-`match` semantic not cited
The Error Handling section asserts "an unhandled value is a compile-time error rather than a runtime
fallthrough." This is a property of the goal compiler's match checker, not established within the spec
or its references. True per the project's design, but an agent unfamiliar with goal's exhaustiveness
checking takes it on faith. Minor.

### m-5. "corpus behavioral tier unchanged" is an AC with no in-spec definition of the tier or how to run it
The AC "The corpus behavioral tier is unchanged" (and FR-6) is checkable in principle, but the spec
never names the command/target for the corpus behavioral tier (unlike `task check/build/fixpoint`,
which are named). The agent must locate it. Minor because the other three gates are concrete and
fixpoint is the load-bearing one for this change.

---

## What is well-specified (no action needed)

- **Consumer-site inventory is accurate and exhaustive.** I grepped `selfhost/` for every symbol;
  the 9 sites + 1 zero-value fix in technical-requirements-research.md match the tree exactly
  (`question.goal:210`, `resolve.goal:218`, `resolve.goal:458`, `convert.goal:34`, `emit.goal:361`,
  `emit.goal:2309`, `ast.goal:170`, `parser.goal:226/229`, `parser.goal:511/515/520`,
  `parser.goal:365`). No missed site.
- **token.Kind refusal** is clearly justified (numeric-identity: `kindNames[k]` indexing, range
  arithmetic) and the AC requires it recorded in DECISIONS.md. Unambiguous.
- **Short-circuit preservation note** (`question.goal:210` has `!ok ||` before the Mod check; idiom
  says split the `!ok` guard out first) is concrete and correct.
- **The `emit.funcDecl` control switch → `isDerive := match d.Mod {...}; if isDerive`** rewrite is
  explicitly prescribed (technical-requirements-research.md line 56–57), which resolves the empty-arm
  problem the research flags.
- **Test-divergence resolution** (relocate FuncMod assertions out of `ast_test.go` into a new
  internal-only `funcmod_test.go`) is correct; I confirmed `ast_test.go:250–255` is the only FuncMod
  reference and that `ast_test.go` is what `port_test.go:142` compiles against the transpiled enum.

---

## Assumptions (inferred choices an implementer must make)

1. **Enum declaration syntax** (C-1): the agent will reverse-engineer the goal `enum` declaration form
   from `internal/backend/testdata/goalenum/use/use.goal` and `testdata/package/cross-pkg-enum/use.goal`,
   producing payloadless variants. Assumed form ≈ `enum FuncMod { FuncPlain; FuncFrom; FuncDerive }`
   (exact punctuation unverified against grammar).
2. **Variant doc comments preserved** (C-1): the existing per-constant `///` docs in
   `goal_decl.goal:23–28` / `ast.goal:675–680` are assumed to be re-attached to the new enum variants
   (spec is silent).
3. **"Port-gate list" = `internal/selfhost/port_test.go`** (M-1): "not in the port-gate test list"
   means do not add `../ast/funcmod_test.go` to the `[]string{...}` at line 142.
4. **`SendRecv` / `FuncPlain` arm bodies** (M-2): the former `default` arm bodies move under the
   bidirectional/plain variants verbatim (`return "chan " + typeString(x.Value)`, `e.p("chan ")`,
   no-op for FuncPlain), and the `default: e.fail(...)` safety arms are dropped.
5. **Construction spelling = FR-4 (qualified)** (M-3): where FR-4 and the consumer list disagree,
   the agent will use `ast.FuncMod.FuncFrom` / `ast.ChanDir.SendRecv`, not the old `ast.FuncFrom` /
   `ast.SendRecv` tokens in the consumer list.
6. **Zero-value fix verified by fixpoint, not a unit test** (M-4): the agent will rely on
   `task fixpoint` to catch a regression of the `Mod: ast.FuncMod.FuncPlain` set at `parser.goal:365`,
   since no AC-named test asserts it.
7. **Same-package pattern drops the qualifier** (m-2): `ast.goal:170` becomes a match over
   `FuncMod.FuncPlain` (no `ast.` prefix).

---

## Recommendation
Before handing to an autonomous agent, add to the spec: (1) a concrete goal `enum` declaration example
(syntax + whether variant docs are kept), (2) the literal location of the port-gate list
(`internal/selfhost/port_test.go:142`) with the exact "do not add this file" instruction, (3) the
per-variant arm→body mapping for the four switches (especially the `SendRecv`/`default` re-homing),
and (4) reconcile FR-4 with the consumer-list construction tokens. Those four edits remove every real
guess; the rest is transcription.

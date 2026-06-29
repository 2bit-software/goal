# SEAM-002 Completeness Audit — FuncMod/ChanDir iota → goal enum

Scope: business-spec.md, research-findings.md, technical-requirements-research.md.
Grounded against the live `selfhost/` sources (commit on `main`).

Verdict: implementable, with several completeness gaps that should be closed
before coding. No hard blocker found (the one capability I worried about —
cross-package enum-typed parameters — is proven by an existing fixture, see
note under MAJOR-1). Highest-severity findings are MAJOR.

---

## CRITICAL

None. Every required lowering form I could identify is either proven by a
fixture or is a trivial extension of a proven form, and the open-questions
sections correctly report no blocking dependency.

---

## MAJOR

### MAJOR-1 — "Fully enumerated consumer set" is incomplete: `goal_stmt.goal` is missing
research-findings.md asserts: *"the consumer set is fully enumerated (9 sites +
1 zero-value fix)."* technical-requirements-research.md's "Consumers to convert"
list does not mention `selfhost/parser/goal_stmt.goal` at all, yet it contains
two live references to the symbols under conversion:

- `goal_stmt.goal:47` — `func (p *parser) parseModFuncDecl(mod ast.FuncMod) *ast.FuncDecl` (a cross-package enum-typed **parameter**).
- `goal_stmt.goal:50` — `fd.Mod = mod` (assignment of an enum value to a struct field).

This file is inside `selfhost/parser`, squarely in the FR-3 scope
`selfhost/{ast,sema,backend,parser}`. Neither site needs a `match` (one is a
type annotation, one a plain assignment), so the conversion is benign — but the
spec's exhaustiveness claim is false, and an implementer working the "9 sites"
checklist will not verify it. The spec should either list it as "reference, no
change required" or fix the enumeration.

Mitigating note for the related capability worry: the cross-package
enum-typed-parameter form (needed here and identical in shape to the existing
`func label(m mood.Mood)` / `func label(l light.Light)` fixtures in
`internal/backend/testdata/goalenum/use/use.goal` and
`testdata/package/cross-pkg-enum/use.goal`) **is** proven, so MAJOR-1 is an
enumeration/documentation defect, not a capability blocker.

### MAJOR-2 — Error-handling reasoning conflates non-exhaustive `match` (compile-time) with the enum nil zero value (run-time)
business-spec.md "Error Handling" states: *"Every variant has an explicit arm,
so an unhandled value is a compile-time error rather than a runtime
fallthrough."* This is only true for the three real variants. The enum's zero
value is **nil** (no variant), which an exhaustive 3-arm `match` does **not**
cover; a nil `FuncMod`/`ChanDir` reaching a `match` is a **run-time** fault, not
a compile-time error. The whole safety argument therefore rests on the
construction-side invariant "no `FuncDecl`/`ChanType` ever carries a nil
Mod/Dir," which is asserted nowhere as a requirement and is tested only by the
single happy-path AC ("a plain function still resolves as `FuncPlain`"). There
is no negative/edge test asserting the invariant globally, and no statement of
what happens if it is violated. Recommend: (a) correct the rationale, and (b)
add a requirement/test that the zero-value invariant holds at every
construction site.

### MAJOR-3 — ChanDir zero-value shift is not explicitly addressed in any AC
The task's central risk — "the enum zero-value differs from the iota
zero-value" — is handled explicitly for **FuncMod** (FR-4 + the AC "a plain
function still resolves as `FuncPlain`… zero-value gap closed") but is **silent
for ChanDir**. Under iota, `ChanDir`'s zero is `SendRecv` (a meaningful
"bidirectional" value, `ast.goal:676`); under enum it becomes nil. The code is
in fact safe because the sole constructor `parser.goal:511`
(`&ast.ChanType{… Dir: ast.SendRecv}`) always sets `Dir` — but the spec never
says so, provides no ChanDir analogue to the FuncPlain AC, and leaves a reviewer
unable to confirm ChanDir safety from the spec alone. The asymmetric treatment
of the two enums (the entire point of the story) is a completeness gap.

### MAJOR-4 — The nil-sensitive short-circuit guard transform is under-specified at the site level
The trickiest consumer, `question.goal:210`:

```
if !ok || fn.Mod != ast.FuncPlain || fn.Recv != nil || fn.Name == nil || fn.Type == nil || fn.Body == nil {
```

Here `fn, ok := d.(*ast.FuncDecl)` means `fn` is nil when `!ok`; today the `||`
short-circuit protects `fn.Mod` from a nil deref. The conversion idiom only says
*"preserving short-circuit by splitting any `!ok ||` type-assert guard out
first"* — correct in spirit, but if an implementer naively hoists
`isPlain := match fn.Mod {…}` above the `!ok` check, it dereferences nil and
panics. This is the one transform where a mechanical reading of the idiom
silently changes behavior; it deserves a concrete before/after recipe rather
than a one-line general note.

---

## MINOR

### MINOR-1 — `default` arms silently absorb a variant; spec never names which
`resolve.goal:458` and `emit.goal:2309` switch on `x.Dir` with explicit
`RecvOnly`/`SendOnly` cases and a `default:` that handles `SendRecv` ("chan ").
The conversion idiom says "replace value-switches with match" but never states
that the default maps to `SendRecv`. Exhaustiveness checking will catch a
forgotten `SendRecv` arm at compile time, so this is low risk, but the spec
should record the mapping. (Same shape, FuncMod side: `emit.goal:361`'s
`default: e.fail("unsupported func modifier")` is dropped entirely — fine for
the 3 valid variants, but see MAJOR-2 re: the lost defensive path.)

### MINOR-2 — FR-3 title vs. body contradiction (cross-package vs. same-package)
FR-3 is titled "All cross-package consumers use match," but its body and the
ACs include the **same-package** site `ast.goal:170` (`d.Mod != FuncPlain`).
"Cross-package" in the title is inaccurate; the requirement is "all consumers,
including same-package."

### MINOR-3 — "exhaustive match" does not state whether wildcard arms are forbidden
The FR value proposition (Error Handling: a new variant becomes a compile-time
error) holds only if `match` exhaustiveness cannot be satisfied by a wildcard
`_` arm. The spec never says wildcards are disallowed, so a conforming
implementation could add `_ => …` and defeat the stated benefit. Make the
"no wildcard / one arm per variant" rule explicit.

### MINOR-4 — Proven-forms list omits construction in struct-literal-field and assignment-RHS positions
The "Proven forms" enumeration covers value/statement-position match and *bare*
construction, but the consumer set also needs construction inside a struct
literal field (`&ast.ChanType{Dir: ast.SendRecv}`, parser.goal:511) and on an
assignment RHS (`c.Dir = ast.RecvOnly`, parser.goal:515/520; `fd.Mod = mod`,
goal_stmt.goal:50). These are almost certainly covered by "bare construction
lowers to an expression," but they are distinct positions and are not listed as
proven.

### MINOR-5 — Grouped `match` arms / OR-patterns left implicit
`emit.goal:361` groups `case ast.FuncPlain, ast.FuncFrom:`. The spec resolves
this via the `match → bool → if` idiom (research-findings) but never states
whether goal `match` supports multi-pattern arms; if it does not and the bool
idiom is mandatory, that should be a stated constraint, not folded into a note.

### MINOR-6 — External references assumed but not defined here
FR-5 cites "AC-1's escape hatch" and the docs repeatedly cite the "port-gate
file list" and DECISIONS.md "Seam methodology." These live outside the three
audited files; the spec assumes the reader/implementer already has them. Fine
for a continuation story, but the relocation requirement ("new internal-only
`internal/ast/funcmod_test.go`, NOT in the port-gate test list") cannot be
self-checked from these docs alone — the mechanism that *defines* the port-gate
list is not referenced.

### MINOR-7 — Golden/oracle regeneration scope is open-ended
AC "task check is green (after any oracle-test relocation/regeneration)" and
"reviewed regeneration of any oracle/golden whose bytes legitimately move" never
enumerate *which* goldens move. The SEAM relaxed gate + `task fixpoint`
self-consistency make this acceptable, but discovery of the affected goldens is
left entirely to the implementer.

---

## Assumptions

Choices the spec relies on that are inferred rather than explicitly stated:

1. **Single FuncDecl constructor.** The zero-value fix is specified only at
   `parser.goal:365` (`&ast.FuncDecl{}`). This is correct *because*
   `parseModFuncDecl` (goal_stmt.goal:47) funnels through `parseFuncDecl`
   (parser.goal:363) before overwriting `Mod` — so one fix covers both the plain
   and the from/derive paths. The spec presents 365 as "the" site without noting
   this funneling.

2. **ChanType always initializes Dir.** ChanDir needs no zero-value fix only
   because the lone `ChanType` constructor (parser.goal:511) always sets
   `Dir: ast.SendRecv`. The spec assumes this rather than asserting it.

3. **`match` exhaustiveness is enforced and wildcard-free**, yielding the
   compile-time-error-on-new-variant guarantee (MINOR-3).

4. **Cross-package enum lowering generalizes** from the proven match/parameter/
   bare-construction fixtures to struct-literal fields and assignment RHS
   (MINOR-4). The enum-typed-parameter case (goal_stmt.goal:47) is the only one
   actually backed by a fixture.

5. **The tagless `switch {}` survives.** `resolve.goal:218`
   (`case d.Mod == ast.FuncFrom || d.Mod == ast.FuncDerive:`) is a switch over
   booleans, not over `FuncMod`; the spec assumes only the inner `==` converts to
   a bool-bound `match` while the `switch {}` itself remains. The ACs' "no plain
   switch over FuncMod/ChanDir" is read as not covering tagless switches.

6. **The dropped defensive `e.fail` paths are dead code.** Converting
   `emit.goal:361` to an exhaustive bool `match` discards
   `default: e.fail("unsupported func modifier")`; the spec assumes that path was
   unreachable for valid (non-nil) inputs (true given Assumption 1).

7. **Emitted-Go byte drift is absorbed by the SEAM gate.** The enum
   representation changes generated Go; the spec assumes `task fixpoint`
   self-consistency + the corpus behavioral tier suffice, rather than pinning
   specific golden updates.

8. **internal/ast needs no consumer changes** beyond relocating the FuncMod
   oracle assertions; it stays Go-iota, and no other internal package depends on
   selfhost's repr.

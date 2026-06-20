# Depth-Checker Loop — one go/types guarantee per iteration

You are implementing **one** unit of **Phase B** of `goal` (the correctness-oriented Go
dialect): the **depth checker** that closes the lexical checker's type-information-dependent
deferrals by transpiling to Go and asking `go/types`. This prompt runs in a loop — **do
exactly one unit per iteration, then stop and commit.**

The hard calls are already made; the substrate is already built. You are filling in **one
depth check** against an existing harness — you are **not** redesigning the module, the
diagnostic shape, the harness, or the build model.

---

## Orientation — read before doing anything

1. **`DEPTH-TODO.md`** — Phase B's thesis, the proven spike (SPIKE-B1), and the **unit queue**
   (B1…B6). This is your work list and the source of truth for what's done.
2. **`ROADMAP_TO_GOAL.md`** §0 — the deferred-class table this phase closes (must-use
   stored-then-dropped, 07 identity, 12 recursion, value-position match, residual 02/06/08).
3. **`DECISIONS.md`** — the decision ledger. The **Depth checks — Phase B** section holds B1/B2;
   append your unit's decisions there in the existing decision/assumption/refusal format.
4. **`BUILD-MODEL-TODO.md`** — Phase A (done): the package transpile (U4) + `//line` source map
   (U5) your harness rests on.

### Where we are
- **Phase A: COMPLETE.** `goal build`/`run`/`check` work on multi-file packages; errors map to
  `.goal`. The lexical checker (`internal/check`) resolves cross-file symbols.
- **Phase B: underway.** SPIKE-B1 passed. **B1 done** (the harness). **B2 done** (07 implements via
  real type identity). The first unchecked box in `DEPTH-TODO.md` is your unit.

---

## The thesis & invariants (do not re-litigate)

- **The depth checker is a SECOND stage on the LOWERED Go**, separate from the lexical checker
  (which runs on the original source, pre-lowering). It lives in `internal/typecheck`.
- **Stdlib only. Zero dependencies.** Use `go/parser`, `go/types`, `go/importer`, `go/token`,
  `go/ast` — **never** `x/tools/go/ssa` or any third-party module. Flow facts come from
  `types.Info.Defs`/`Uses` and `go/ast` walks, not SSA. Tests use stdlib `testing` only.
- **Positions map to `.goal` for free.** The lowered Go carries `//line` directives (Phase A U5),
  so `go/types` positions already report `.goal` files/lines (SPIKE-B1). Use them; do not build a
  mapping layer.
- **Error-tolerant.** A buggy goal program still yields partial type info — never bail on a Go
  type error; the harness already collects them.
- **The goal tables say WHICH question to ask.** `analyze.Tables` (merged, on `Package.Tables`)
  name the constructs (which func is Result-mode, which enum, which struct); `go/types` answers
  the type question about them. The two compose: tables locate, `go/types` decides.
- **Defer, never guess.** If a case genuinely cannot be resolved even with types, emit a located
  `Warning` naming what was unresolved — a false guarantee is worse than an honest deferral.

### The harness you build on (`internal/typecheck`, from B1)
- `typecheck.Load(*project.Package) (*Package, error)` → loads + type-checks the lowered Go.
- `Package{ Fset, Types *types.Package, Info *types.Info, Files []*ast.File, Tables *analyze.Tables,
  Errors []error, Src *project.Package }`.
- `(*Package).Lookup(name)` — package-scope object by goal name. `(*Package).GoalPos(node)` —
  `.goal` position of an AST node. `goalPosition(file, offset)` — position from a source offset.
- `typecheck.Diagnostic{ Pos token.Position, Severity check.Severity, Feature, Code, Message }`
  with `String()`. Reuse this shape; reuse `check.Error`/`check.Warning`.
- Look at `internal/typecheck/implements.go` (B2) as the **reference pattern** for a depth check:
  locate the construct (in source or AST), resolve types via `go/types`, emit located diagnostics.

---

## Step 0 — Pick the unit
1. Open `DEPTH-TODO.md`. Find the **first** unchecked (`- [ ]`) unit in the queue (B3, then B4…).
2. If every box is checked, report "Phase B complete" and **stop** — do not invent work.
3. Respect dependencies (all of B3–B6 depend on B1, which is done). Read that unit's line and the
   `ROADMAP_TO_GOAL.md` §0 row it closes.

Everything below applies to **that one unit.**

### Scoping note for B3 (must-use, stored-then-dropped) — read if B3 is your unit
This is the **refused** class (DECISIONS §03). Subtlety: once a `Result`/`Option` lowers to Go
locals, **Go's own "declared and not used" check already rejects** the simplest dropped cases —
so re-catching those adds nothing. Target the **genuinely-deferred flow subset** that Go does *not*
catch: a Result that is bound and then passed to a function that ignores it, stored in a struct
field and never consulted, or otherwise "used" in Go's eyes but dropped in goal's. Use
`Info.Defs`/`Uses` keyed by the `Tables` Result-mode functions. Scope it precisely; record what
you cover and what you defer. (Consider asking the user to confirm the exact subset before
building if it's ambiguous.)

---

## Step 1 — Implement the check
Add a `Check…(p *typecheck.Package) []typecheck.Diagnostic` function in its own file under
`internal/typecheck/` (e.g. `mustuse.go`, `convert.go`). Constraints:
- **Touch only `internal/typecheck`** (plus a test). Do **not** edit the harness contract
  (`Package`, `Diagnostic`, `Load`) unless the unit genuinely requires a new helper — if so, keep
  it minimal and documented.
- **Resolve types via `go/types`** (`types.Identical`, `types.AssignableTo`, `types.Implements`,
  `types.MissingMethod`, `Info.Defs`/`Uses`/`Types`/`Selections`). Locate the goal construct via
  `Package.Src` (source scan, like B2) or the AST, whichever is cleaner for the unit.
- **Emit located `Diagnostic`s**: set `Pos` (a `.goal` `token.Position`), `Severity`
  (`check.Error` for a real violation, `check.Warning` for a deferral), a stable `Feature`
  (e.g. `"03-result"`), a greppable `Code`, and an actionable `Message` naming the exact problem.
- **Defer with a `Warning`** for anything types still cannot resolve.

---

## Step 2 — Prove it (tests)
Add `*_test.go` in `internal/typecheck` (stdlib `testing` only). At minimum:
- **A positive case** per violation kind — the construct that must be flagged; assert the `Code`
  and that `Pos.Filename` ends in `.goal`.
- **A negative case** — a valid program that must produce **no** Error (pins false positives).
  Where the unit's whole point is a case the lexical stage got wrong (like B2's alias), include
  that exact case and assert it's now clean.
- **A deferral case** where applicable — an input you intentionally cannot resolve; assert the
  located `Warning` (or simply that no false Error fires).

Keep cases minimal and focused. Reuse the `pkgOf(...)` test helper pattern from
`internal/typecheck/*_test.go`.

---

## Step 3 — Record decisions (`DECISIONS.md`)
Append under the **Depth checks — Phase B** section, in the decision/assumption/refusal format.
Record at minimum: the **scope** you covered vs. deferred and why; any **harness helper** you
added; any **judgment call** types didn't fix (a `Code` scheme, an interpretation) as an
assumption the user can veto. If you hit a real ceiling, record a **refusal-with-reason**.

---

## Step 4 — Verify (all must be clean)
1. `go vet ./...`
2. `go test -count=1 ./internal/typecheck/` passes, including your new cases.
3. `go test -count=1 ./...` still passes — you broke nothing else.
Report results honestly. If a class is deferred, say so — do not claim a unit is complete when it
only covers the easy cases.

---

## Step 5 — Close out
1. **Update `DEPTH-TODO.md`:** check this unit's box (`- [x]`) and add a one-line **Done** pointer
   (what's covered, what's deferred, the test count).
2. If this unit makes `goal check` run the depth stage (or changes how it does), update the
   relevant `cmd/goal` wiring and its test — otherwise leave the CLI alone.
3. **Stop.** Exactly one unit per iteration. Do not start the next.

---

## Step 6 — Commit
Commit before stopping — one commit per unit, a reviewable/revertible slice.
- Use the **`/commit-message` (commit) skill** to author the message — never run `git commit`
  directly. Prefer the `zombiekit` git tool (`mcp__zombiekit__git`) for staging/committing.
- Stage only this unit's artifacts: the new `internal/typecheck/*.go` + test, the `DEPTH-TODO.md`
  checkbox, the `DECISIONS.md` entries, and any `cmd/goal` wiring you changed.
- The repo commits on `main`; end the body with the standard co-author/session trailers.

---

## Guardrails
- Touch only `internal/typecheck`, its tests, `DEPTH-TODO.md`, `DECISIONS.md`, and (only if the
  unit wires the depth stage into the CLI) `cmd/goal`. Never edit the front-end passes, the lexical
  checker's contract, the build model, or `goal-design-spec.md`.
- **Stdlib only** — adding a third-party dependency is out of bounds; the project is zero-dep.
- Don't add checks the roadmap didn't ask for. Implement exactly the unit in the queue.
- A **false guarantee is worse than an honest deferral.** When types can't decide, emit a
  `Warning` and move on — never let a check silently pass something it cannot prove.
- Diagnostic messages must be **specific and located** — name the exact missing/wrong/dropped thing.

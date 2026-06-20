# BUILD-MODEL-TODO — Phase A: analysis substrate & project model

Phase A of `ROADMAP_TO_GOAL.md`. Goal: **`goal build ./...` turns a multi-file goal package
into a compilable Go package, and Go-toolchain errors are reported at `.goal` positions.**
This is the keystone — Phase B (`go/types` depth checks) and Phase C (LSP) both sit on it.

This file is loop-ready in the `CHECKER-TODO.md` style: a **thesis** (how it works and why
it's tractable), the **claims that must be proven by spike** before we commit, a
**dependency-ordered unit queue** (one shippable unit per iteration), and the **open
decisions** to resolve at the right unit boundary.

---

## Thesis — how the build model works

Grounded in three facts about the engine as built (verified by reading
`internal/scan/scan.go`, `internal/analyze/analyze.go`, `internal/pipeline/pipeline.go`,
`internal/pass/closed.go`):

### Claim 1 — Cross-file resolution is a table *merge*, not a rearchitecture
`analyze.Tables` is **entirely name-keyed and position-free**: `FuncSignatures`, `Enums`,
`Sealed`, `Structs`, `TypeDecls`, `FromRegistry`, `Interfaces`, `EmbeddedIfaces`, `Methods`
all map a *symbol name* to strings (types, normalized signatures, variant sets). No field
holds a byte offset. Therefore a **package's tables = the union of each file's tables**.

Each pass already runs over *one* source string, re-lexing it and rebuilding spans locally;
it reads cross-pass facts only through `Tables` by name. So a pass running on file B, given
the **merged** tables, resolves a symbol declared in file A (an enum, a struct, a `from
func`) without ever seeing file A's text. ⟹ **Multi-file transpile = build merged tables
once per package, then run the existing per-file pipeline on each file unchanged.**

*Residual risks (the spike must clear):* (a) name collisions across files — but two files
declaring the same package-level name is a genuine Go redeclaration the Go compiler catches,
so the merge rule is "union; the Go compiler is the backstop"; (b) the package-level
injection hazard — see Claim 2.

### Claim 2 — One shared prelude; imports stay per-file
The only package-level **declaration** a pass injects is the closed-E Result preamble in
`internal/pass/closed.go` (`resultPreamble`: `type Result[T,E]`, `Ok`, `Err`, and their
`isResult()` methods). Injected via `injectOffset`, **per file**. If two files in a package
both use closed-E Result, each injects the preamble ⟹ **redeclaration error**.

Injected **imports** are different: `assert.go` injects `import "fmt"`, `doctests.go` emits
`import "testing"`. Go scopes imports per *file*, so duplicate import injection across files
is harmless.

⟹ The fix is narrow: **lift `resultPreamble` out of the per-file pass into a single
package-level file** (a generated `goal_prelude.go`, or an imported runtime package), emitted
**once per package**; leave per-file import injection exactly as is. The `closed` pass keeps
rewriting constructions/matches/`?` but stops injecting the type block.

### Claim 3 — The "source map" is `//line` directives, not a bespoke structure
Passes apply `scan.Splice` internally and **return only the new source string** — the
`[]Replacement{Start,End,Text}` journal is discarded. There is no composed offset map to
thread, and reconstructing one would mean changing all 10 pass signatures and surviving the
final `go/format` reflow.

Instead, lean on the Go toolchain the way the thesis demands: **emit Go `//line
file.goal:line` directives into the generated Go**, and let the Go compiler, `go/types`, and
`go vet` report errors at the mapped `.goal` position natively. This is exactly how `cgo` and
`goyacc` map generated-code errors back to source.

This is sound because of *what* we need to map: the realistic failure is an ordinary Go type
error in **passed-through** code (a function-body expression goal copies verbatim). Passed-
through Go is **not spliced** — its text is identical in source and output, so its goal line
is recoverable. The spliced regions are *our* generated lowerings; an error there is a goal-
compiler bug, not the user's, and gets reported against the prelude/lowering.

⟹ "Source map" = a `//line`-directive emitter keyed to goal positions, plus a thin fallback
("error in generated code from X.goal") for spliced regions. **Must verify `//line`
directives survive `go/format`** (the format-once step) and that the compiler honors them.

### What the thesis buys
Phase A is therefore **mostly plumbing around an unchanged front-end**: a package walker,
a table-merge function, a one-line prelude relocation, a `//line` emitter, and a `goal build`
driver that shells out to the Go toolchain and relays errors. No pass rewrite; no parser; no
offset bookkeeping across passes.

---

## Claims to prove first (SPIKES — do these before the unit queue)

Two load-bearing claims gate the whole phase. Each spike is small, throwaway, and decisive.
**Both ran 2026-06-20 and PASSED** — the thesis is proven; the queue stands.

- [x] **SPIKE-1 — `//line` survival & remap. PASSED.** Hand-wrote a Go file with `//line
  prog.goal:N` directives and a type error (`var x int = "not an int"`) in a passed-through
  region tagged `prog.goal:7`. **Result:** `go build` reported the error at exactly
  `prog.goal:7` — the Go compiler honors `//line` natively, no custom remapping. And
  `go/format` (the literal format-once call) **preserved** both directives intact. ⟹ The
  source map is a `//line` emitter; the Go toolchain does the remapping. No fallback needed.
  *Caveat for U5:* gofmt preserved directives on their own lines; granularity/column tuning
  (per-decl vs per-statement) is still U5's call.

- [x] **SPIKE-2 — merged tables + cross-file lowering + shared prelude. PASSED.** Two files in
  package `demo`: A declares `enum Shape` + error `enum MathErr`; B `match`es `Shape` and
  returns `Result[float64, MathErr]` (E from A). **Results:** (1) per-file `analyze.Build(B)`
  has `Enums[Shape] == nil`; the **union** of the two files' tables has it, variants
  `[Circle Square]`. (2) Running the unchanged `pipeline.Passes` over B *with merged tables*
  lowered the cross-file `match` to a real `switch __gop_v := sh.(type) { case Shape_Circle:
  … }` — no unlowered `match` remained. (3) The closed-E `half()` lowered to `Ok/Err[float64,
  MathErr]`. (4) The two-file package **compiled** with a single emitted `goal_prelude.go`.
  ⟹ Claims 1 & 2 hold: merged name-keyed tables are sufficient for cross-file lowering, and
  one shared prelude makes it compile. (Spike code was throwaway, already removed.)

> Outcome: no claim refuted; the unit queue below stands as written. Spike learnings folded
> into U2 (merge is a literal map-union), U3 (prelude relocation is the only dedup needed),
> and U5 (`//line` emitter, granularity TBD).

---

## Unit queue (dependency-ordered; one per iteration after the spikes)

Each unit follows the loop discipline (`ROADMAP_TO_GOAL.md` §5): implement, prove with
testdata, record decisions in `DECISIONS.md`, verify (`go vet ./...`, `go test -count=1
./...`), check the box, commit one reviewable unit, stop.

- [x] **U1 — Package model & file discovery.** A `package`/workspace type: given a directory
  (or `./...`), find `.goal` files, group them into packages (one package per dir, Go-style).
  Define the `goal.File`/`goal.Package` types the rest of the phase consumes. No transpile yet.
  - **Done:** `internal/project` — `File{Path,Name,Src}`, `Package{Dir,Name,Files}`,
    `Discover(root)` (recursive, groups by dir, sorted), `PackageClause` (lexes the clause).
    Enforces one-package-per-directory; skips `testdata`/hidden/`_` dirs. 6 tests pass; vet clean.
    Cross-package goal imports deferred (DECISIONS Phase A §U1).

- [x] **U2 — Cross-file table merge.** `analyze.BuildPackage([]File) *Tables` (or a `Merge`)
  that unions per-file tables; define and test the collision rule (union; document last-wins vs.
  duplicate-detection, deferring genuine dup-decls to the Go compiler). Prove a cross-file
  reference (enum in A, match in B) resolves. *Depends on SPIKE-2.*
  - **Done:** `analyze.BuildPackage([]string)` + `Tables.Merge` (`maps.Copy` union over every
    name-keyed map), `newTables()` constructor extracted from `Build`. Collision rule:
    last-merged-wins, deterministic via path-sorted input; genuine dup-decls left to the Go
    compiler. 4 tests incl. cross-file enum resolution + last-wins; vet clean. DECISIONS Phase A §U2.

- [x] **U3 — Shared prelude relocation.** Move `resultPreamble` out of `closed.go`'s per-file
  injection into a single package-level emission (`goal_prelude.go` or runtime import). The
  `closed` pass stops emitting the type block; the package driver emits it once iff any file in
  the package uses closed-E Result. Keep per-file import injection unchanged. *Depends on
  SPIKE-2, U2.*
  - **Done:** exported `pass.ResultPreamble` + `pass.NeedsResultPrelude(t)`; inline injection now
    gated by `analyze.Tables.SuppressResultPrelude` (the construction/match/`?` rewrites always
    run). Single-file output byte-identical — full regression suite green. U4 sets the flag and
    emits one `goal_prelude.go`. 3 gate tests; vet clean. DECISIONS Phase A §U3.

- [ ] **U4 — Package transpile driver.** Transpile every file in a package with the merged
  tables (U2) + single prelude (U3); write `.go` outputs. Extends `pipeline.Transpile` (single
  source) to `pipeline.TranspilePackage`. Round-trip test: a multi-file package → a compilable
  Go package. *Depends on U2, U3.*

- [ ] **U5 — `//line` source-map emitter.** Emit `//line file.goal:N` directives per the SPIKE-1
  outcome, so toolchain errors map back to `.goal`. Ships the reusable mapping helper Phase C's
  LSP reuses. *Depends on SPIKE-1, U4.*

- [ ] **U6 — `goal build` / umbrella CLI.** A `goal` command (`build`/`check`/`run`, then
  `fmt`/`new` later) that runs U4 over `./...`, shells out to `go build`/`go vet`, and relays
  errors mapped through U5. `goalc` (single-file) stays as the core. End-to-end: a multi-file
  goal project builds & runs, and a Go error in passed-through code is shown at its `.goal`
  line. *Depends on U4, U5.*

- [ ] **U7 — Cross-file checker.** Extend `check.Analyze` to run over a package with merged
  tables, so the existing 7 guarantees resolve cross-file symbols (closes the 02/06/08
  out-of-file deferrals at the *lexical* level; the `go/types` depth versions are Phase B).
  *Depends on U2.*

**Done when:** a multi-file goal package (≥2 files with a cross-file enum + closed-E Result +
`?`) builds via `goal build`, runs correctly, the checker sees across files, and a deliberate
Go type error in passed-through code is reported at the correct `.goal` location.

---

## Open decisions (resolve at the named unit, not now)

- **Prelude delivery** (U3): generated `goal_prelude.go` per package vs. a versioned imported
  runtime module. Generated-file is simpler and self-contained; imported-runtime eases future
  upgrades. *Lean: generated file for v1.*
- **Output layout** (U4/U6): write `.go` next to `.goal` (sibling, gitignored) vs. a `build/`
  output dir vs. an in-memory module for `go build`. Affects `//line` paths and `go:generate`.
- **`//line` granularity** (U5): per-decl vs. per-statement directives; columns vs. line-only.
  Driven by SPIKE-1's fidelity result.
- **Package scope** (U1): single-package multi-file first (common case); **cross-package goal
  imports** (one goal package importing another) is a later unit, explicitly out of Phase A v1.
- **Collision policy** (U2): silent union (Go compiler backstops dup-decls) vs. an upfront
  located checker diagnostic for same-name decls across files. *Lean: union now, diagnostic later.*

---

## Pointers
- `ROADMAP_TO_GOAL.md` — the phase order and the working loop.
- `internal/analyze/analyze.go` — the name-keyed `Tables` (Claim 1's evidence).
- `internal/pass/closed.go` — `resultPreamble` + `injectOffset` (Claim 2's target).
- `internal/scan/scan.go` — `Replacement`/`Splice` (Claim 3: journal is discarded).
- `internal/pipeline/pipeline.go` — `Transpile` / format-once (extended by U4).

_Status: thesis drafted 2026-06-20; both spikes PASSED 2026-06-20 — thesis proven, queue stands. Next: U1._

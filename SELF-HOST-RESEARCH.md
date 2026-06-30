# Self-Host Research: building the goal compiler in goal, end-to-end

> Research for **Phase 4 (Self-host)** of `REWRITE-ARCHITECTURE.md`. Phases 0–3 are done:
> the AST front-end is the sole fact source and the last `text/scanner` crutch is gone
> (feature `ralph/checker-onto-ast-prune-scan`, US-001…US-010). This document establishes
> feasibility, scope, the blockers, and the bootstrap mechanics. It does **not** yet break
> the work into loop stories — that is the follow-on `prd.json`.

---

## 0. Verdict

**Self-hosting is feasible, and closer than the §5 strategy assumed.** Two findings reframe it:

1. **It is mostly a transpile-the-existing-Go effort, not a rewrite.** goal is a Go *superset*;
   the compiler is plain Go. The goal parser already accepts every ordinary Go construct the
   compiler uses (closures, maps, slices, interfaces, method receivers, type switches,
   recursion, variadics, named returns, struct embedding). So the "port" is largely: copy each
   `internal/<pkg>/*.go` to `.goal`, then fix the handful of constructs that don't round-trip.

2. **Foreign Go imports pass through untouched — empirically confirmed for `go/types` and
   `go/format`.** A goal program that `import "go/types"` / `"go/format"` builds and runs. This
   means the heavy toolchain crutches do **not** need to be shelled out: the goal-written
   compiler imports them as ordinary Go, exactly as the Go version does. The §5 "shell-out
   behind an interface" plan is therefore optional, not required.

The blockers are **a small, finite set of front-end gaps** (one of them silent and gating),
not a language-expressiveness wall.

---

## 1. Scope — what gets ported

**In-scope: the compiler proper (9 packages, ~11,856 LOC).** Topologically ordered (port order):

| # | Package | LOC | Role | Notes |
|---|---------|-----|------|-------|
| 1 | `token` | 361 | token kinds + positions | leaf; **hits the `iota` blocker** (§4.1) |
| 2 | `lexer` | 514 | source → tokens | uses `unicode`, `unicode/utf8` |
| 3 | `ast` | 1,988 | node defs + Walk | `dump.go` uses `reflect` (debug-only — drop or rewrite) |
| 4 | `parser` | 1,765 | tokens → AST (recursive descent) | gnarliest after backend/sema; precedence climbing |
| 5 | `sema` | 2,695 | resolve + check + foreign | `foreign.go` uses `go/parser`/`go/format`/`go/types` |
| 6 | `project` | 147 | package discovery | `io/fs`, `os`, `path/filepath` |
| 7 | `pipeline` | 173 | output types + source map | small |
| 8 | `backend` | 3,441 | AST → Go (emit + lower) | **largest, gnarliest**; stateful emitter, ~100 methods |
| 9 | `typecheck` | 772 | go/types depth checks | `go/types`/`go/importer`; **deferrable to a later milestone** |

Plus the two CLIs: `cmd/goalc` (single-file) and `cmd/goal` (build/check/run; shells out to `go`).

**Out-of-scope for first self-host (~9k LOC):** `lsp`, `fix`, `guide`, `goalfmt`, `interp`
(goscript), `corpus`/`byexample` (test infra), `cap`. These are tooling, not the compiler.

> **SUPERSEDED by the self-host flip (see `DECISIONS.md` → "US-001 — self-host flip:
> adopted layout & trust model").** The flip pulls `goalfmt`, `textedit`, `cap`,
> `guide`, `fix`, `interp`, and `lsp` *into* the self-hosted closure — "self-hosted"
> now means the shipped `goal`+`goalc` library closure, which includes that shipped
> tooling. Only the genuine test/dev infra (`corpus`, `byexample`, the
> `internal/selfhost` harness, `cmd/corpus-gen`, `cmd/build-playground`) stays Go.
> The "tooling, not the compiler ⇒ out of scope" framing above is the pre-flip view.

**Dependency DAG (internal edges only) — clean, acyclic:**
```
token → {lexer, ast} → parser → {sema, project} → pipeline → backend → typecheck
```
Each package's internal deps are ported before it, so the port is strictly leaf-to-root.

**Hardest packages:** `backend` (3,441 LOC; stateful `emitter` struct, ~100 emission methods,
all the goal-specific lowerings) and `sema` (2,695 LOC; multi-pass resolution, `?`-inference
state machine, cross-file/foreign merge). `parser` is standard recursive-descent complexity.

---

## 2. External dependency surface

All stdlib — **no third-party**. 31 distinct imports. Classification, now that passthrough is
empirically confirmed:

- **Import as foreign Go passthrough (the default, and it works):** `fmt`, `io`, `os`, `io/fs`,
  `path/filepath`, `os/exec`, `strings`, `bytes`, `strconv`, `sort`, `unicode`, `unicode/utf8`,
  `errors`, `maps`, `encoding/json`, and the Go-toolchain libs `go/token`, `go/ast`, `go/parser`,
  `go/types`, `go/importer`, `go/format`. The empirical probe built+ran goal programs importing
  `go/types` and `go/format` directly. **No shelling out required.**
- **Reflection:** only `internal/ast/dump.go` (`Sexpr` debug dump). Not on the compile path —
  drop it from the self-hosted build or rewrite as an explicit node switch.
- **No** `unsafe`, `cgo`, `go:embed`, `go:generate`, build tags, goroutines, channels, or
  `select` anywhere in the compiler proper. Single-threaded throughout.

**Seam status (informational, since passthrough removes the requirement to box):**
- `go/format` is already behind `backend.Formatter` (`backend/backend.go:37`); a second direct
  use at `sema/foreign.go:305` (`format.Node`).
- `go/types`/`go/importer` are behind `typecheck.TypeChecker` (`typecheck/checker.go:17`) for the
  depth stage, **but** `backend/arity.go:101-136` uses them directly on the primary transpile
  path (foreign-call `?`-arity resolution). With passthrough this is fine — the ported
  `arity.go` just imports `go/types` like the original.

---

## 3. Language feasibility — goal can express its own compiler

The goal parser/AST accept every ordinary Go construct the compiler uses (verified against
`internal/parser` + `editors/tree-sitter-goal/grammar.js`):

generics (type decls/methods/instantiation), type switches, type assertions, struct embedding,
anonymous structs, closures/func-lits, variadics, multiple+named returns, `iota`/const blocks
*(see blocker)*, struct tags, labeled break/continue, goto, select, channels, goroutines,
defer, maps, slices, range, interface embedding, blank identifier, init funcs, multiple
assignment.

The compiler uses **no** goroutines/channels/select and **no** generic *function* declarations
(grep of `internal/` finds zero; the only `[T any]` is inside an emitted prelude string). So the
intersection of "what goal can't do" and "what the compiler needs" is small — see §4.

---

## 4. Blockers (the actual work) — prioritized

### 4.1 `iota` const-block mangling — SILENT, GATING ⛔
The idiomatic Go enum:
```go
const ( Red Color = iota; Green; Blue )
```
transpiles to broken Go:
```go
const ( Red Color = iota; Green Blue )   // Green parsed as "name type", not a repeat
```
Bare const names that should repeat the prior `= iota` initializer are mis-emitted as
`name type` pairs. `go build` of a transpiled `internal/token` fails (`missing init expr for
EOF`). **The checker does not flag it — it is a silent miscompile.** `token` (port unit #1) and
every idiomatic Go enum depend on this. **This is unit 0; nothing self-hosts until it's fixed.**
Lives in the parser's GenDecl/ValueSpec path + the backend const emission.

### 4.2 Generic function declarations rejected ⚠️ (optional for this compiler)
`func Identity[T any](x T) T` → parser error `expected (, found [`. Localized:
`parser.go:347 parseFuncDecl` never calls the existing `atTypeParams`/`parseTypeParams`
(`parser.go:281,296,307`); also needs a `TypeParams` field on `ast.FuncType` (only `TypeSpec`
has one, `ast.go:261`) and an emit in `funcDecl`/`funcSig` (`backend/emit.go:360,439`). A
~3-spot fix. **The compiler uses zero generic funcs, so this is nice-to-have, not gating** —
but worth doing for language completeness and to keep the port unconstrained.

### 4.3 Comments stripped from generated Go — acceptable ✓
The backend drops `//`, `///`, and doc comments. Cosmetic, and **harmless for the fixpoint**
(both bootstrap stages strip identically). Consequence: the self-hosted compiler emits
comment-free Go. Acceptable for the trust proof; revisit only if generated-Go readability
matters.

### 4.4 CLI ergonomics — minor
`goal run/build` are package/directory-oriented (`goal run .`), not file-oriented
(`goal run main.goal` chdir's into the file and fails). The bootstrap must use the directory
form. Not a blocker, just a contract to honor.

---

## 5. Bootstrap & fixpoint mechanics

> **SUPERSEDED in part by the self-host flip (see `DECISIONS.md` → "US-001 — self-host
> flip: adopted layout & trust model").** Two things below are pre-flip framing: (1)
> the goal compiler no longer lives at a peer `./selfhost` dir — the flip relocates it
> *into* the `internal/` namespace as colocated `<file>.goal` source + committed
> generated `<file>.go`; and (2) the Go build is no longer the *permanent* trust root.
> The flip adopts the **B-commit** bootstrap (committed generated Go, drift-gated),
> and once the hand-written reference Go transpiler is deleted the **corpus behavioral
> tier becomes the primary correctness gate** (the byte-identical `fixpoint` check
> below still holds). The 3-stage mechanics and the fixpoint trust gate themselves are
> unchanged.

No `selfhost/` dir or Taskfile bootstrap target exists yet. The goal-written compiler would be a
`package main` goal program (a goalc-in-goal) under e.g. `./selfhost`. The classic 3-stage
bootstrap with the byte-identity trust gate:

```sh
set -e
task build                                     # stage 0: trusted Go-built ./bin/goal[c]

# stage 1: build the goal-written compiler USING stage 0
./bin/goal build --emit=build/s1 ./selfhost    # goal source -> Go
go build -o bin/goal-c-1 ./build/s1            # Go -> native goal-c-1

# stage 2: build the goal-written compiler USING goal-c-1
./bin/goal-c-1 build --emit=build/s2 ./selfhost
go build -o bin/goal-c-2 ./build/s2            # goal-c-2

# fixpoint: the two goal-built compilers must emit byte-identical Go for their own source
./bin/goal-c-1 build --emit=fix/a ./selfhost
./bin/goal-c-2 build --emit=fix/b ./selfhost
diff -r fix/a fix/b && echo "FIXPOINT OK"
```

The **corpus behavioral conformance tier** (built in Phase 0) is the other control: the
goal-built compiler must pass the same corpus the Go compiler does. Run it after every ported
package so `main` stays green throughout.

Key references: build driver `cmd/goal/main.go` (`cmdBuild:480`, `transpileAll:461`,
`runGo:683`); single-file primitive `cmd/goalc/main.go`; package transpile + prelude
`backend/package.go:36`; prelude trigger `backend/lower.go:115`.

**Prelude story (confirmed clean):** open-E `Result[T,error]` → native `(T,error)` (no prelude);
`Option[T]` → `*T` (no prelude); closed-E `Result[T,E]` → one shared `goal_prelude.go` per
package; non-trivial `Option.Some` → one `goal_options.go` per package. No cross-file
duplication.

---

## 6. Strategy decision: port plain first, dogfood later

§5 implies dogfooding goal's own features (Result/`?`/match/enums) in the compiler. The §5
meta-principle — *never change architecture and language at once* — extends naturally: **never
change language and idiom at once.**

**Recommended:** port the existing Go source to goal as a near-mechanical, plain-Go-superset
translation first (it already nearly round-trips), achieve the fixpoint, lock it with the
corpus — *then* dogfood goal features incrementally (e.g. parser errors via `Result`/`?`, AST
dispatch via `enum`+`match`), each refactor re-proving the fixpoint. This keeps the existing Go
as a line-by-line differential oracle during the risky bootstrap, and isolates "is the language
expressive enough" from "did I introduce a logic bug rewriting in a new idiom."

---

## 7. Risks

- **Silent transpile defects** like §4.1. Mitigation: feed real compiler source through the
  transpiler early and `go build` the output as a gate (the empirical probe is how §4.1 was
  found at all — the checker stayed silent). Build a "transpile + compile every in-scope package"
  smoke test as part of unit 0.
- **`backend`/`sema` size & state.** Mitigation: leaf-to-root order means they're ported last,
  against a fully-working ported front-end, with the corpus as control.
- **Foreign import edge cases.** `go/types` and `go/format` confirmed; the long tail (`io/fs`,
  `encoding/json`, `os/exec`) should each get a one-line passthrough smoke test before the
  package that needs it is ported.
- **`arity.go` go/types on the hot path.** Confirmed it imports cleanly as passthrough; verify
  the goal-built `arity.go` resolves foreign `?`-arity identically (corpus covers this).

---

## 8. Proposed Phase 4 shape (for the follow-on prd)

1. **Unit 0 — close the front-end gaps:** fix `iota` const-block mangling (gating) + add a
   "transpile-then-`go build` every in-scope package" smoke gate; optionally fix generic-func
   decls (§4.2).
2. **Bootstrap harness:** `selfhost/` skeleton (goalc-in-goal `package main`) + Taskfile
   `bootstrap`/`fixpoint` targets running the §5 sequence, even against a stub, so the gate
   exists before code.
3. **Leaf-to-root port**, one package per unit, each gated on transpile-compile + corpus:
   `token → lexer → ast → parser → sema → project → pipeline → backend`, then `typecheck`.
4. **Fixpoint milestone:** goal-c-1 and goal-c-2 byte-identical; goal-built compiler passes the
   full corpus.
5. **(Optional) dogfood pass:** refactor the goal-written compiler to use Result/match/enums,
   re-proving the fixpoint after each step.

Total in-scope: ~11.9k LOC, near-mechanical for most of it once §4.1 is fixed. The gating risk
is concentrated in unit 0 (the silent `iota` bug) and the two big packages (`backend`, `sema`),
both de-risked by the corpus control and the existing Go source as oracle.

## 9. Idiomatic end state — measured (SEAM PRD, SEAM-006 proof)

Phase 5 (the "dogfood pass" of §8.5) ran as a separate **seam** PRD that relaxed the
byte-identical-output gate to a fixpoint-self-consistency + corpus-behavioral gate, so
cross-package idiom changes that alter emitted Go could be re-proven equivalent. The
before/after below is counted from the live `selfhost/` tree (full per-seam tally in
DECISIONS.md "SEAM-006"):

| Idiom | Before (transpiled Go) | After (idiomatic goal) |
|-------|------------------------|------------------------|
| iota classification types | 6 `type X int`+iota | 4 → `enum` (FuncMod, ChanDir, Mode, Severity); 2 kept iota for numeric identity (token.Kind, litClass) |
| AST dispatch | ~36 plain `switch n.(type)` over OPEN interfaces | 27 → exhaustive `match` over a SEALED AST (134 type-pattern arms); 9 documented non-fits |
| Fallible seam API | `(T,error)` + manual `if err!=nil` | 7 Result-returning APIs, 56 `Result.Ok`/`Err`/`?` sites; remainder documented semantic non-fits |

`goal fix` over all 39 selfhost `.goal` files auto-modifies zero of them (12 result-sig
refusals + 14 advisory call-site notes, all mapping to documented non-fits) — the autofixer
agrees the propagating API is idiomatic. `task fixpoint` = FIXPOINT OK on the new source.

**The central result is a META-finding:** goal's deep idioms were blocked not only by
per-package scope but by MISSING compiler features. Reaching this end state required FOUR new
capabilities built in the seam PRD — SEAM-CAP (cross-package enum-match lowering), SEAM-CAP-2
(cross-`.goal`-package enum/sema-fact propagation), and SEAM-CAP-3a–d (sealed-interface
type-pattern match: method-sig preservation, same-package match, cross-`.goal`-package match,
nested hierarchies). Before this work a cross-package enum `match` errored and
sealed-interface match did not exist. The idiomatic self-host was gated on building real
language features, not merely on widening audit scope.

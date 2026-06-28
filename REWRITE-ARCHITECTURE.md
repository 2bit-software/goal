# Rewrite & Architecture Proposal: AST front-end, self-hosting, and the goscript runtime

Status: **proposal / for sign-off**. Date: 2026-06-26.

This document is the output of a research + architecture session. It answers four
questions the owner posed:

1. Are we building the right architecture for what we have *and* for the future
   (self-hosted `goal`, plus an embedded `goscript` runtime that runs inside
   Go/C/Java/TypeScript)?
2. Are we following best practices — interfaces, modularity, separation of concerns,
   no anti-patterns?
3. Is there a better way to do language parsing than the current token-splice approach?
4. Can we keep the golden corpus across the refactor and make it reusable for the
   later self-host / runtime rewrites?

---

## 0. Executive summary

**Recommendation: replace the lexical token-splice front-end with a real
`lexer → parser → goal AST` front-end, built once for *full* Go+ semantics, and make
every tool (transpiler, checker, LSP, `fix`, `fmt`, and the future interpreter) a
back-end over that one AST.**

The reasoning in one line: **both of the owner's future goals independently require an
AST the project has deliberately never built.** A `goscript` interpreter cannot
tree-walk a string rewriter — it needs an AST. A self-hosted `goal` compiler cannot
lean on `go/types`/`go/format`/`text/scanner` the way today's Go implementation does —
it needs its own front-end. The roadmap (`ROADMAP_TO_GOAL.md:222`) said to reopen the
"AST-vs-lexer fork" only on a *concrete ceiling*. **These two goals are that ceiling.**

This is *also* what the design spec already committed to and the project then did out of
order. `goal-design-spec.md` §1.4 specifies the build order: (1) Go+ semantics spec →
(2) goscript restriction diff → (3) **a shared checker/front-end built for full Go+
semantics** → (4) goscript execution engine → (5) Go+ transpile backend. In practice
the transpile backend was built *first* (rational — cheapest path to a working
product), which inverted the order and entangled the checker with the Go-transpile path.
This proposal puts the front-end back at the center where the spec wanted it.

What we keep: the full golden corpus, the name-keyed-facts discipline, the
format-once-at-the-end output model, the "erase the guarantee, keep the runtime,
panic-not-silent" philosophy, and the zero-dependency / stdlib-only constraint.

What we retire: the 11 ordered string-splice passes, the three duplicated arm-splitters,
the token-peeping type-inference engine in `match`, the string-based type parser in
`derive`, the magic `__goal_` hygiene prefix, and `Splice`'s silent overlap-drop.

---

## 1. Where we are today (grounded)

The Go-side toolchain is, by explicit written contract, **parser-free and AST-free**
(`check/check.go:19`, `ROADMAP_TO_GOAL.md:66`). Everything structural rests on two
shared layers:

- **`internal/scan`** — a `text/scanner` wrapper producing a *flat* token stream with
  byte offsets, plus ~15 helpers (`MatchBrace`, `FirstBodyBrace`, `ParamsClose`,
  `TopLevelComma`, `ScanFuncs`) that re-derive nesting on demand. No tree is built.
- **`internal/analyze`** — *name-keyed* symbol tables (`Enums`, `Structs`, `Sealed`,
  `FuncSignatures`, `FromRegistry`, `Methods`) built once by token-pattern scanning.
  Keyed by name, not offset, precisely so they survive splicing.

On top of these, **11 ordered passes** (`internal/pass`) each re-lex the current source,
locate a construct by token-scanning + balanced-delimiter matching, accumulate byte-span
`Replacement`s, and `Splice` once; the driver `go/format`s once at the very end
(`pipeline.go:91`).

There are, right now, **three independent structural models of goal source plus
pervasive ad-hoc re-derivation**:

1. The lexical substrate (`scan` + `analyze`) — shared by passes, `check`, `fix`, and the
   LSP's diagnostics, but *flat*, so every consumer re-lexes and re-matches braces to
   rebuild local structure. `lsp/symbols.go:scanDecls` even adds its own top-level walk.
2. `internal/typecheck` — a *real* `go/types` analysis, but over the **lowered Go**, not
   goal source (`typecheck.go:64`). This is the "transpile then ask the Go compiler"
   crutch. It is elegant and works *only because the target is Go*.
3. `editors/tree-sitter-goal/grammar.js` — a near-**complete** CST grammar for goal
   (enums, match, `?`, implements, derive/from, assert, doctests, labeled args, spread
   elements). Editor-only today, but a credible blueprint/cross-check for an AST.

### 1.1 The lexical ceiling, with evidence

The audit found the string-rewriting design has **passed the point where it pays for
itself on the four complex passes** (`match`, `result`, `closed`, `derive`) while still
being pleasant on the trivial ones (`assert`, `implements`, `defaults`). Concrete tells:

- **A type-inference engine built from token-peeping.** `match.go:191-237`
  (`inferMatchType`) recognizes only "lone string literal → string", "lone bool → bool",
  "whole-body `Enum.Variant(...)` → enum"; numeric bodies are "too many sub-types to pin
  lexically" (`match.go:212`) and it refuses the general `name := match` case. This is a
  type checker painfully reinvented through tokens.
- **The same arm-splitter written three times** — `parseMatchArms` (`match.go:240`),
  `parseResultArms` (`result.go:190`), `parseOptionArms` (`option.go:147`).
- **The same constructor lowered twice** — `resultReturnReps`/`optionArmBody` re-lower
  `return Result.Ok/Err` and `Option.Some/None` inside match arms *purely to dodge*
  `Splice`'s silent overlap-drop (`scan.go:60`).
- **A type-syntax mini-parser operating on `string`** — `derive.go`'s `ptrInner`,
  `arrElem`, `mapKV`, `derefType` bracket-count inside `map[K]V` etc.; one `types.Type`
  walk replaces all of it.
- **Hygiene by magic prefix** — `result` emits `__goal_ok/_err` and `question` consumes
  them by formatting the same string constants (`pass.go:22`); collision safety rests
  entirely on `__goal_` being unused in user code. No gensym, no scope.
- **Known correctness bugs from no parser** — `analyze.parseStructBody`'s comma-split
  mis-splits embedded-comma field types (`DECISIONS.md:1571`); B3 had to read fields from
  `go/types` to sidestep it. `implements.go:90` splits the interface list comma-naively
  and would mis-split `implements Map[K, V]`.

The recurring shape: with no types and no expression tree, the safe move is to **refuse**
(located error) whenever a lexical heuristic can't be sure. The language's expressible
surface is bounded by what token-peeping can recover.

### 1.2 What is genuinely worth preserving

A rewrite must not throw these away:

- **Name-keyed facts that survive transformation.** This made cross-file package merge a
  trivial `maps.Copy` union (`analyze.Merge`). Keep it as the symbol-table discipline.
- **Format once, at the end.** Only the driver formats. Keep this for the Go backend.
- **Per-construct isolation & incrementalism.** Each feature shipped as a unit. The AST
  design must keep "add a feature" cheap (one desugaring rule, not a cross-cutting edit).
- **The "erase the guarantee, preserve the runtime, panic-not-silent" stance** (the
  default arm of a proven-exhaustive match is a loud `panic("unreachable")`).
- **Zero dependency / stdlib-only.** No tree-sitter C, no parser generators.

---

## 2. Decision: the parsing approach

**Adopt a hand-written, Go-grammar-shaped recursive-descent parser producing a goal AST.
Reuse Go as the *back-end* (`go/printer` for codegen, `go/types` over lowered Go for
depth checks during the Go-transpile era) — never as the *parser*.**

Why not the alternatives:

- **Keep the lexical splice.** Rejected: no path to an interpreter (§4) and no path to
  self-hosting without the Go-stdlib crutches (§3). The ceiling in §1.1 is already here.
- **Reuse `go/parser`/`go/scanner`.** Rejected: goal is **not a clean superset of Go**.
  `go/scanner` emits `ILLEGAL` for `?`, `=>`, `field:` call args, and `...ident` literal
  elements; `go/parser` cannot parse `enum`, `match … =>`, `sealed`, `implements`, or
  `from/derive func`, and is hand-written recursive descent with no extension points.
  Forking it is larger and more brittle than writing ~8 new productions on a Go-shaped
  skeleton.
- **Embed tree-sitter.** Rejected: compiles to C, violates zero-dependency. But
  `grammar.js` is the **executable grammar spec** — use it as a differential oracle (parse
  the whole corpus through both, diff the structure) to validate the hand-written parser.
- **Hybrid (AST only for the 4 complex constructs, splice for the rest).** Viable as an
  *interim*, and the migration in §6 effectively passes through it — but not the
  end-state, because the interpreter and LSP both want *one* tree for the whole file.

The genuinely hard parsing problems are **context-sensitivity, not new keywords**, and
they are exactly what an AST resolves structurally:

- `Enum.Variant(x)` is syntactically a Go call but means **construct / destructure-bind /
  ordinary call** depending on type and grammatical position. Today this forces
  Match-before-Enums pass ordering and a `matchSpans` skip-list; with an AST a
  `MatchArm` pattern, a `VariantLit`, and a `CallExpr` are simply different node types.
- `match` is both an **expression** (`var x = match …`) and a **statement**. The evaluator
  and value-position lowering both need it as an expression node — today value-position
  match is refused outright (`result.go:139`).
- `?` is a **postfix unary** in the expression precedence table (use Pratt/precedence-
  climbing for expressions so `f(x)?`, `a.b?`, `g()?` fall out naturally).
- `implements` / `sealed` / `from` / `derive` are **contextual keywords** — lex them as
  identifiers, treat positionally, so user code using those as names still works.

---

## 3. Target architecture

One front-end, many back-ends. The front-end is **engine-neutral** (it knows nothing
about Go codegen vs interpretation); back-ends are pluggable behind interfaces.

```
  source ─▶ token ─▶ lexer ─▶ parser ─▶ AST ─▶ check (typed, full Go+ semantics)
                                               │
                                               ├─▶ backend/go      (transpile → Go)      ← today's product
                                               ├─▶ backend/interp  (tree-walk → goscript)← future runtime
                                               └─▶ backend/vm      (bytecode for C/Java/TS hosts) ← later
  tools (lsp, fix, fmt, guide) all consume the same AST + check results
```

Proposed package layout (names indicative; Go during bootstrap, ported to goal at
self-host):

| Package | Responsibility | Notes |
|---|---|---|
| `token` | token kinds + `Pos` (offset/line/col) | positions are first-class on the AST, unlike today |
| `lexer` | source → tokens; knows `?`, `=>`, `field:`, `...`, `///`, contextual keywords | replaces `scan.Lex`; reuse for the differential test vs tree-sitter |
| `ast` | node types + `Visitor`/`Walk`; declarations, statements, expressions, **patterns** | the one structural model; absorbs `scan` helpers + most of `analyze` |
| `parser` | tokens → AST, Go-shaped recursive descent + Pratt expressions | validated against the corpus and `grammar.js` |
| `types` | type representation + the **shared checker for full Go+ semantics** | see §3.2; may delegate depth to `go/types` initially |
| `sema` (check) | correctness checks over the typed AST: exhaustiveness, no-zero-value, must-use, implements, `?`-arity | folds in today's `check` + `typecheck` |
| `lower`/`ir` | desugar goal constructs to a core form (optional middle layer) | makes backends thin; one place defines "Result = Ok/Err sum" |
| `backend/go` | typed AST/IR → Go source (`go/printer`) | the current transpiler, rebuilt |
| `backend/interp` | typed AST/IR → tree-walking evaluator | the goscript runtime |
| `cap` | capability/effect model (shape now, surfaced later) | §4; spec §4.4 says design the *shape* in early |
| `driver`/`pipeline` | orchestration, file/package discovery, formatter selection | thin |
| tools: `lsp`, `fix`, `fmt`, `guide` | consume `ast` + `sema` | LSP finally gets go-to-def/hover/rename/semantic-tokens |

### 3.1 Best-practice interfaces (directly answers the "interfaces / SoC / DI" ask)

The current code is concrete-to-concrete (passes call `scan`/`analyze` directly; the
pipeline hardcodes `go/format`). Introduce seams where they buy reuse or testability —
without over-abstracting:

- **`Backend`** — `interface { Emit(pkg *ast.Package, info *sema.Info) (Artifacts, error) }`.
  The Go transpiler, the interpreter, and the future VM are three implementations. This is
  *the* seam that makes the runtime a drop-in.
- **`Formatter`** — `interface { Format([]byte) ([]byte, error) }`. Go backend uses
  `go/format`; the self-hosted build may shell out to `gofmt`. Decouples codegen from the
  Go stdlib so the goal port isn't blocked on reimplementing `go/format`.
- **`TypeChecker`** — lets the bootstrap keep the `go/types`-over-lowered-Go crutch behind
  an interface and swap in a native checker when the runtime forces it (§3.2). No caller
  changes when the implementation flips.
- **`FileSystem` / source loader** — abstract `project.Discover`'s disk I/O (per the user's
  DI standard: all I/O behind an interface), so the driver, LSP (in-memory buffers), and
  WASM playground share one driver.
- **Host capability interface** (`cap`) — how the interpreter is granted authority (I/O,
  time, concurrency). Shape it now even if v1 grants everything (spec §4.4).

Keep value semantics for the AST nodes and table facts (the user's Go standard), inject a
`Clock` where time is needed, and **do not** add interfaces the codebase has only one
implementation of and no test seam for — that would be its own anti-pattern.

### 3.2 The type-checking question (the load-bearing decision under the decision)

Today's depth checks transpile to Go and ask `go/types` (`typecheck.go`). This is the
"crutch" — and it is *fine for the Go-transpile path even after self-hosting*, because the
self-hosted compiler still shells out to `go build`, which runs `go/types` anyway.

It is **not** fine for the interpreter: goscript runs in a host with no Go toolchain at
runtime, so exhaustiveness / must-use / implements / no-zero-value must be provable from a
**native** checker over the goal AST. The spec is explicit: build the shared checker once
for full Go+ semantics; goscript *restricts* it, Go+ *extends* it; **never widen the
checker** (`goal-design-spec.md` §1.3-1.4).

**Recommendation:** keep the crutch behind the `TypeChecker` interface for the
Go-transpile era; build the native checker only when the runtime forces it, and design the
`sema` API now so both implementations satisfy it. Do **not** block the front-end rewrite
on a full native type system.

---

## 4. The goscript runtime — what the architecture must reserve for it

The owner's "runtime that runs inside Go/C/Java/TypeScript" is the spec's **`goscript`**
(§1.1-1.3): an embeddable, in-process, capability-sandboxed, **statically typed**,
**semantically exact subset** of Go+, with a frictionless script→binary upgrade path. It
is fully specified and entirely unbuilt; the multi-host (C/Java/TS) ambition is owner
intent beyond the written spec, which today says only "engine we own, in-process, no host
toolchain at runtime."

Architecture implications, all satisfiable by §3:

- **Same front-end, different back-end.** goscript = the shared `lexer→parser→ast→sema`
  front-end + a capability restriction + `backend/interp`. The `Backend` interface is what
  makes this a clean addition rather than a fork.
- **Uniform runtime value model.** The interpreter ignores the Go-codegen optimizations
  (Result→`(T,error)`, Option→`*T`) and uses the **universal tagged-union** representation
  for enums/Result/Option: `{typeId, variantTag, fields}`. Construction builds a tag+payload;
  `match` dispatches on the tag; the proven-exhaustive default arm panics. Types are
  checked statically and **erased at runtime** (spec §2.x).
- **Only two genuinely non-Go runtime mechanics:** value-position `match` (an expression
  that yields a value) and `?` early-return unwinding (non-local control flow threaded
  through `eval`). Everything else is ordinary eager evaluation.
- **Capabilities are authority, not language difference** (spec §4.4): goscript "removes"
  concurrency/I/O by the host not granting the capability, not by a different grammar.
  Reserve the `cap` seam now.
- **The restriction diff is still an open spec item** (`goal-design-spec.md` §6, open
  questions): "enumerate exactly what goscript removes." It doesn't block the front-end,
  but it must be written before `backend/interp`.

The strategic payoff: building the AST front-end now is *not* speculative work for the
runtime — it is the **step 3 the spec said to do first**, finally done, and it is shared
1:1 between the transpiler and the runtime.

---

## 5. Self-hosting (dogfooding) strategy

Self-hosting appears in **no** prior planning doc — it is a greenfield decision, and the
main obstacle is exactly the Go-stdlib reliance (`text/scanner`, `go/format`, `go/types`,
`go/importer`). The §3 architecture removes the first (we write our own lexer/parser) and
boxes the rest behind interfaces (`Formatter`, `TypeChecker`) that the goal port can
satisfy by shelling out to the Go toolchain the compiler already drives.

**Do the architecture change in Go first, prove it, then port to goal.** Do not change the
architecture *and* the implementation language at the same time — that compounds risk and
destroys the golden corpus as a control.

Classic three-stage bootstrap, with a trust gate:

- **Stage 0** — the rebuilt Go implementation (AST front-end + Go backend). Known-good,
  validated against the full corpus.
- **Stage 1** — the compiler rewritten *in goal*. Build it with Stage 0
  (goal source → Go → `go build`) to get `goal-c-1`.
- **Stage 2** — compile the goal-written compiler *with itself* (`goal-c-1` builds it) to
  get `goal-c-2`. **Fixpoint check:** the Go that `goal-c-1` and `goal-c-2` emit for the
  compiler's own source must be byte-identical. That equality is the self-host trust proof.

goal is expressively sufficient to write a compiler (it is a Go dialect); the bottleneck is
purely the stdlib seams above, which the interfaces resolve.

---

## 6. The golden corpus — preserve and make reusable

This is the safety rail for the entire effort, so it comes **first**, not last.

Inventory (audited): **51 active transpile golden pairs** (40 `features/NN/examples/*.goal`
+ `*.go.expected`, 11 `testdata/*.goal`) + **50 checker cases** (`testdata/check/**`, 34
inline `// want` markers) + the doctest sidecars. The per-feature
`features/NN/transpiler/` dirs are **legacy standalone prototypes** in their own
`go.mod`, excluded from `go test ./...` — archive them.

Coupling is low: the `.goal`/`.go.expected` files reference **no** internal package; only
~3 harness files are coupled (hardcoded dir list + `../..` paths + the `Transpile`/
`Analyze` call sites). A restructure breaks the *harness wiring*, not the corpus.

The one real hazard: **the transpile goldens are encoding-specific** — they assert exact
generated names (`__goal_ok`) and exact Go encodings. A fresh AST-based backend will emit
*different but equivalent* Go and break exact-match goldens even when behavior is
identical. So we need two tiers:

1. **Manifest + exact tier.** Extract `corpus/` — a JSON/TOML index over the existing files
   (no need to move them) with three case kinds: `transpile` (gofmt-normalize both sides,
   then compare), `check` (inline `// want`, already manifest-shaped), `doctest`
   (`Output.Test` exact). Define "normalize = gofmt both sides" *in the manifest spec* so any
   runner — Go-hosted, goal-hosted, or interpreter — uses the same comparison. Regenerate the
   exact `.go.expected` from the new backend once, and keep them as a within-implementation
   regression lock.
2. **Behavioral conformance tier (new).** For each input, compile the generated Go and run
   its doctests / a behavioral assertion. This tier is **implementation-independent**: it
   validates "this goal program behaves thus" regardless of the Go encoding chosen, so it is
   the conformance suite the self-hosted compiler *and* the interpreter both run. The
   feature-11 doctest corpus already hints at this tier.

Net: the corpus survives the refactor essentially intact (rewire 3 harness files, regenerate
exact goldens once), and the new behavioral tier is what makes it reusable across the
self-host and runtime rewrites.

---

## 7. Phased migration plan (with verification gates)

Each phase ends at a green gate; no big-bang.

- **Phase 0 — Corpus hardening (do first).** Extract the `corpus/` manifest; add the
  behavioral conformance tier; archive the legacy per-feature transpilers. Gate: old
  pipeline passes the manifest + behavioral tier unchanged. *This is the control for
  everything after.*
- **Phase 1 — AST front-end in Go, no behavior change.** Build `token`/`lexer`/`ast`/
  `parser`. Gate: parse 100% of corpus inputs; differential-check structure against
  `grammar.js`; round-trip `ast → source` for fidelity spot-checks. The old splice pipeline
  still produces all output.
- **Phase 2 — Go backend off the AST, behind a flag.** Build `sema` (folding `check` +
  `typecheck` behind the `TypeChecker` interface) and `backend/go`. Run old and new side by
  side through the behavioral tier until new is green; then regenerate exact goldens from new
  and flip the default. Retire `internal/pass` + `Splice`. Gate: behavioral tier green on new
  backend; exact goldens regenerated; `goal build/run/check` unchanged externally.
- **Phase 3 — Consolidate tools.** Point `lsp`, `fix`, `fmt`, `guide` at the shared AST; add
  the LSP features the AST unlocks (go-to-def, hover, rename, semantic tokens). Gate: LSP +
  fix test suites green.
- **Phase 4 — Self-host.** Port the front-end + Go backend to goal; bootstrap; fixpoint
  check (§5). Gate: stage-1 and stage-2 emit byte-identical Go for the compiler's own source;
  goal-built compiler passes the full corpus.
- **Phase 5 — Runtime.** Write the goscript restriction diff; add `cap` + `backend/interp`;
  run the behavioral conformance tier through the interpreter. Gate: behavioral tier green
  under interpretation; a sample script graduates to a Go+ module as ~no-op.

Phases 0-2 are the load-bearing rewrite; 3-5 are the payoff the architecture was chosen for.

---

## 8. Risks & mitigations

- **Comment/whitespace fidelity loss.** Splice preserved every untouched byte; `ast →
  printer` normalizes. Mitigation: the *generated Go* is a throwaway artifact (users read
  `.goal`), so normalization there is acceptable; for `goal fmt` over goal *source*, use
  position-carrying nodes + comment attachment, the one place fidelity matters.
- **Parser scope creep (must understand all of Go).** A full parser must handle generics,
  type params, the lot — 90% of which the splice approach ignored. Mitigation: Go-shaped
  skeleton + the `grammar.js` oracle bounds the work; the corpus is the acceptance test.
- **Encoding-specific goldens churn.** Mitigation: the two-tier corpus (§6) — behavioral tier
  is stable, exact tier is regenerated once per backend.
- **Doing too much at once.** Mitigation: architecture-in-Go-first, then port; phase gates;
  the corpus as a continuous control.
- **Native type checker underestimated.** Mitigation: keep the `go/types` crutch behind the
  interface; only build native when the runtime (Phase 5) forces it.

## 9. Open decisions for sign-off

1. **Commit to the AST front-end** (vs. hybrid-for-now vs. status quo)? *(Recommend: yes,
   full AST.)*
2. **Sequencing:** architecture-in-Go-first then port to goal (recommended), or attempt the
   self-hosted rewrite directly?
3. **Type checking:** keep the `go/types`-over-lowered-Go crutch behind an interface for now
   (recommended), or invest in a native goal type checker up front?
4. **Scope of this initiative now:** all of Phases 0-2, or start with Phase 0 (corpus
   hardening) as a standalone, low-risk first step that de-risks everything else?

### Decisions taken (2026-06-26)

1. **Full AST front-end** — commit to `lexer → parser → AST → checker`, engine-neutral,
   with pluggable backends. (Not hybrid, not status-quo.)
2. **Architecture in Go first, then port to goal.** Rebuild in Go, prove against the
   corpus, *then* self-host. Never change architecture and language at once.
3. **Scope now: Phases 0–2** — corpus hardening, AST front-end in Go, new Go backend that
   retires the splice passes. Phases 3–5 (tool consolidation, self-host, runtime) are
   planned but out of this initiative's execution scope.
4. **Keep the `go/types`-over-lowered-Go crutch behind a `TypeChecker` interface.** No
   native goal type checker until the runtime forces it.

---

## 10. Detailed plan for Phases 0–2 (approved scope)

Principle throughout: **the corpus is the continuous control.** Nothing in Phase 1–2
ships until the behavioral conformance tier (built in Phase 0) is green. The old splice
pipeline stays the product until Phase 2's flip, so `main` is always shippable.

### Phase 0 — Corpus hardening (the control)

Goal: make the golden corpus a runner-independent manifest with an implementation-
independent behavioral tier, so both the old and new front-ends are judged by the same
yardstick. No transpiler behavior changes in this phase.

- **0.1 `corpus` package + manifest.** Add `internal/corpus` defining a `Case` model and a
  loader. Manifest is a generated JSON/TOML index over the *existing* files (do not move
  them): per case `{id, kind: transpile|check|doctest, input, expected, mode: file|package,
  normalize: gofmt}`. Generate it by walking the current `features/NN/examples` +
  `testdata` + `testdata/check` trees so it captures today's 51 transpile pairs, 50 check
  cases, and the doctest sidecars exactly.
- **0.2 Runner behind an interface.** Define `type Transpiler interface { Transpile(src
  string) (Output, error) }` and `type Checker interface { Analyze(src string)
  []Diagnostic }`. The runner consumes the manifest and a `Transpiler`/`Checker`, applies
  gofmt-normalize-both-sides for `transpile`, inline-`// want` matching for `check`, and
  `Output.Test` exact-match for `doctest`. This removes the hardcoded dir lists and `../..`
  paths from `pipeline_test.go`/`check_test.go` — they become thin adapters that pass
  `pipeline.Transpile` to the runner.
- **0.3 Behavioral conformance tier.** For each `transpile` case, compile the generated Go
  in a temp module (reuse the `goal build` temp-dir machinery) and, where a doctest sidecar
  exists, run it; otherwise assert "compiles + `go vet` clean." This tier asserts *behavior*,
  not Go spelling, so it is the implementation-independent suite the new backend (and later
  the self-host/runtime) must satisfy.
- **0.4 Reify package-mode cases.** Convert the two inline package-mode tests
  (`pipeline/foreign_test.go`, `pipeline_package_test.go`) into on-disk multi-file fixtures
  with a declared import map so they live in the manifest, not in Go string literals.
- **0.5 Archive legacy prototypes.** Move `features/NN/transpiler/` (separate-`go.mod`
  standalone reference transpilers, already excluded from `go test ./...`) to `attic/` and
  note them as historical.
- **Gate 0:** the *current* pipeline passes the manifest exact tier + the behavioral tier,
  with the old harness wiring deleted in favor of the runner.

### Phase 1 — AST front-end in Go (no behavior change)

Goal: a real parsed tree for the whole file, validated by the corpus and the tree-sitter
grammar. The old splice pipeline still produces all shipped output during this phase.

- **1.1 `token`.** Token kinds + `Pos{Offset, Line, Col}`. Positions are first-class
  (today's offsets are thrown away every pass).
- **1.2 `lexer`.** Source → `[]Token`. Must produce the goal-specific lexemes the splice
  approach faked: `?` as a token, `=>` as one token (not `=` then `>`), `...` as one token,
  `///` doc-comment content retained (not skipped — doctests + `fmt` need it), and
  `implements`/`sealed`/`from`/`derive` lexed as *identifiers* (contextual keywords, decided
  by the parser positionally). Carry comments as trivia attached to positions for `fmt`.
- **1.3 `ast`.** Node types + `Walk`/`Visitor`. Must cover: goal decls (`FuncDecl` incl.
  `from`/`derive` modifiers, `EnumDecl`/`Variant`/`PayloadField`, `StructDecl` with
  `ImplementsClause`, `SealedInterfaceDecl`, plus ordinary `TypeDecl`/`Import`/`var`/`const`);
  statements (incl. `AssertStmt`, statement-`MatchExpr`); expressions (`MatchExpr` as an
  *expression*, `MatchArm`, `VariantPattern`/`RestPattern`, `VariantLit` with `LabeledArg`,
  `UnwrapExpr` for postfix `?`, `SpreadElement` for `...defaults`/`...derive`); and the
  ordinary Go expression/statement/type forms goal passes through. Resolve the three
  meanings of `Enum.Variant(x)` structurally (construct vs. destructure-bind vs. call) — the
  problem that forces Match-before-Enums today.
- **1.4 `parser`.** Recursive descent for declarations/statements, Pratt/precedence-climbing
  for expressions (so postfix `?` and value-position `match` fall out of the precedence
  table). Scope note: goal is mostly-Go, so the parser must handle the Go grammar subset
  goal actually uses (generics/type params, composite literals, all statement forms) — sized
  like `go/parser` but smaller, since we target only goal's surface. This is the largest
  single work item.
- **1.5 Differential oracle.** Parse every corpus `.goal` input through both the new parser
  and `editors/tree-sitter-goal/grammar.js`, and diff the structure; treat divergences as
  bugs in one or the other. Plus a `parse → print → parse` round-trip fidelity check.
- **Gate 1:** new parser parses 100% of corpus inputs; structure agrees with the tree-sitter
  grammar; round-trip is stable. Old pipeline unchanged.

### Phase 2 — Go backend off the AST (the flip)

Goal: replace the 11 splice passes with `sema` + `lower` + `backend/go`, and make the new
backend the default once it is behaviorally green.

- **2.1 `sema`.** Symbol resolution + the correctness checks, over the typed tree. Derive
  today's name-keyed facts (enums, structs, signatures, from-registry, methods) by *walking
  the AST* instead of token-scanning, eliminating `analyze`'s comma-split bugs. Fold
  `internal/check` + `internal/typecheck` here; keep the `go/types`-over-lowered-Go depth
  checks behind the `TypeChecker` interface (decision 4) so nothing native is required yet.
- **2.2 `lower` (IR/desugar).** One desugaring rule per construct, replacing one splice pass
  each: enum/sealed → sum encoding; Result(open-E) → `(T,error)`; Result(closed-E) → sum;
  Option → `*T`; `match` → type-switch (incl. value position, which the splice approach
  refuses today); `?` → propagation with *real gensym* (retire the magic `__goal_` prefix);
  `implements` → assertion/marker; `...defaults` → explicit zeros; `assert` → if-panic;
  `derive` → field-by-field (type structure from resolved types, not string parsing);
  doctests → `_test.go`. Output is a backend-agnostic core the interpreter will later share.
- **2.3 `backend/go`.** Walk the lowered core and emit Go *source text* via a structured
  printer (a code-gen visitor — not string surgery on input), then `Formatter.Format` once.
  Constructing `go/ast` + `go/printer` is an option where precision matters (e.g. the
  synthesized Result prelude), but text emission from a clean tree preserves the
  format-once model with less ceremony.
- **2.4 Interface wiring.** Land `Backend`, `Formatter`, `TypeChecker`, and the source-loader
  interface in the driver (§3.1). Go backend = `go/format` formatter + the new emitter.
- **2.5 Dual-run, regenerate, flip.** Run old and new backends side by side through the
  behavioral tier until new is green on every case. Then regenerate the exact `.go.expected`
  goldens from the new backend (they *will* differ — new gensym names, different but
  equivalent encodings) and commit them as the within-implementation regression lock. Flip
  the default; delete `internal/pass` + `internal/scan`'s splice machinery + `analyze`'s
  token-scan tables that `sema` now subsumes.
- **Gate 2:** behavioral tier green on the new backend; exact goldens regenerated from it;
  `goal build/run/check/fix` unchanged externally; LSP still green (it keeps consuming
  `check`/`fix`, now AST-backed — Phase 3 adds the new capabilities).

### Sequencing notes

- 0.1–0.3 are the critical path; 0.4–0.5 can trail. Don't start Phase 1 before Gate 0.
- Phase 1 has no user-visible output — it can proceed in parallel with unrelated feature
  work on the old pipeline, since nothing depends on it until Phase 2.
- The Phase 2 flip is the only risky moment; the dual-run gate (2.5) makes it reversible
  (keep the old backend behind a flag for one release before deleting).

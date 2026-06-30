# Self-Host Flip Plan — make goal the compiler's only source

Status: **proposal / planning only — no files moved yet.**
Goal of this doc: a phased plan to move the goal-written compiler (`./selfhost`,
`.goal`) into the `internal/` namespace and retire the hand-written Go transpiler,
so goal source becomes the single source of truth for the toolchain.

This is a deliberate **departure** from the current documented design. See
"Why this is a departure" below before committing to it.

---

## 1. The two hard constraints

Everything in this plan exists to work around these.

**C1 — `.goal` is not `go build`-able.** `cmd/goal` and `cmd/goalc` are Go
programs compiled by the Go toolchain. They import `goal/internal/{...}` — real
Go. The current `selfhost/*` packages are `.goal`; they only become Go when the
trusted stage-0 binary transpiles them during `task bootstrap`. You cannot drop
`.goal` files into `internal/` and have `go build ./cmd/goal` work. Some
Go-buildable form of the compiler must always exist.

**C2 — `selfhost/` is not a complete replacement.** It ports the compiler proper:

```
token  lexer  ast  parser  sema  project  pipeline  backend  typecheck   (+ main.goal)
```

`cmd/goal` *also* imports these legacy Go packages that have **no goal port**:

| package    | Go LOC | role                         | port difficulty |
|------------|-------:|------------------------------|-----------------|
| interp     |   3722 | goscript interpreter         | large, no exotic Go |
| lsp        |   2292 | language server (JSON-RPC)   | **hardest** — `sync.Mutex`, `any`, stdio streaming |
| fix        |   1060 | autofixer                    | medium (deps `textedit`) |
| guide      |    400 | guide/docs surface           | small |
| goalfmt    |    143 | formatter                    | small |
| textedit   |    168 | edit primitives (`fix` dep)  | small |
| cap        |    110 | capabilities (`interp` dep)  | small |

`corpus` (1057) and `byexample` (270) are **test/dev infrastructure**
(`cmd/corpus-gen`, `cmd/build-playground`, golden harnesses), not part of the
shipped compiler. They can stay Go — see §6.

So the genuinely-must-port set for a self-hosted **shipped** toolchain
(`goal` + `goalc`) is: `goalfmt, guide, textedit, cap, fix, interp, lsp`
≈ **7.9k LOC**, dominated by `interp` and `lsp`.

---

## 2. Why this is a departure (read before committing)

Per `SELF-HOST-RESEARCH.md` §1/§5 and `REWRITE-ARCHITECTURE.md` §7, the existing
design intends:

- `internal/` (hand-written Go) and `selfhost/` (goal port) **coexist as peers**.
- The **Go build is the permanent trust root** (stage-0). The 3-stage bootstrap
  + byte-identical fixpoint derives trust *from* that Go root each build.
- There is **deliberately no committed generated-Go bootstrap**.
- `lsp/fix/guide/goalfmt/interp` are **explicitly out of scope** for self-host —
  "tooling, not the compiler."

A full flip changes all four of those decisions. The benefit is real (one source
of truth; goal dogfoods itself end-to-end; no hand-maintained Go drift). The cost
is a new trust model (§3) and the ~8k LOC port (§5). This is worth doing only if
"goal is the only source" is an actual project goal, not just tidiness. If the
aim is just to stop `selfhost/` looking like a second-class peer, **Phase 1 alone**
(relocation + committed bootstrap, legacy Go retained behind it) gets 80% of the
ergonomic win at ~10% of the cost.

---

## 3. The new bootstrap trust model (the core design decision)

Once the hand-written Go is gone, what does a clean `git clone && go build`
compile? Three options:

- **B-commit (recommended): committed generated Go.** Transpile the goal source
  to Go and **commit that output**. A clean checkout builds the committed Go →
  stage-0 → re-transpiles the goal source → verifies the regenerated Go is
  byte-identical to what's committed (drift gate) → fixpoint. This is the
  standard self-hosting bootstrap (rustc stage0, many compilers). Reproducible,
  reviewable, no prebuilt binaries.
- **B-binary: shipped prebuilt stage-0 binary.** Worse: opaque, platform-bound,
  not reviewable. Rejected.
- **B-emit-only: no commit, generate on every build.** Chicken-and-egg — needs a
  bootstrap binary to generate, which is B-binary. Rejected.

**Recommendation: B-commit.** Everything below assumes it.

### Resulting layout (the literal answer to "selfhost into internal")

You cannot put `.goal` source and its generated `.go` at the *same* importable
path. So the flip splits into source vs. generated:

```
internal/compiler/<pkg>/*.goal   <- canonical goal source (was selfhost/*)
internal/<pkg>/*.go              <- COMMITTED generated Go (was hand-written)
```

- `cmd/*` keep importing `goal/internal/<pkg>` unchanged — those paths now hold
  *generated* Go instead of hand-written Go.
- `task generate` transpiles `internal/compiler/**` → `internal/**`.
- During the transition, `internal/<pkg>` is a mix: generated for ported
  packages, still-hand-written for not-yet-ported ones. End-state: all generated
  (plus the Go-only test harnesses of §6).

This literally moves `selfhost/` into the `internal/` namespace
(`internal/compiler/`) and makes the legacy Go paths generated artifacts of it.
(A more aggressive variant colocates `parser.goal` + generated `parser.go` in one
dir, à la `.proto`/`.pb.go`. Cleaner imports, messier dirs. Decide in Phase 0.)

---

## 4. Phases at a glance

| phase | outcome | reversible? | gating risk |
|-------|---------|-------------|-------------|
| 0 | layout + trust model decided, written down | n/a | bikeshedding |
| 1 | B-commit bootstrap proven **with legacy Go still present** | fully | drift gate flakiness |
| 2 | tooling ported to goal (goalfmt→…→lsp), still peer | fully | lsp concurrency, interp size |
| 3 | the flip: cmd/* on generated, hand-written Go deleted | hard | hidden import/test coupling |
| 4 | final layout, Taskfile/CI/docs, peer `selfhost/` removed | easy | docs lag |

Phases 1 and 2 are independent and can run in parallel.

---

## 5. The phases in detail

### Phase 0 — Decide and record (no code)
- Confirm B-commit and the source-vs-generated layout (§3); pick colocated vs.
  split dirs.
- Define "self-hosted" precisely: the shipped `goal`+`goalc` and their import
  closure. Test-only harnesses (§6) are explicitly excluded.
- Write the decision into `DECISIONS.md` and supersede the "coexist as peers"
  language in `SELF-HOST-RESEARCH.md` §1/§5 so the docs stop contradicting it.

### Phase 1 — Stand up the committed-generated bootstrap (legacy Go stays)
De-risk the trust model *before* deleting anything.
- Add `task generate`: transpile the current `selfhost/*` to Go, write under a
  staging path (e.g. `internal/compiler/` source + a generated tree), commit it.
- Add `task verify-generated`: regenerate, `diff` against committed → fail on
  drift. Wire into `task check` and pre-commit.
- Add a clean-room build path that compiles the *generated* tree (not the
  hand-written Go) into a `goal-gen` binary, and prove it passes the full test
  suite + corpus behavioral tier. At this point two compilers exist and agree;
  nothing is deleted.

### Phase 2 — Port the remaining tooling to goal
Order: leaf-to-root, smallest/safest first; **plain-Go-superset port first, then
dogfood goal idioms** (the rule from `SELF-HOST-RESEARCH.md` §6 — never change
architecture and language at once). Each package gated by behavioral parity
(corpus + its own test suite run against the goal-built version).

1. `goalfmt` (143) — smallest, formatter, easy parity check (idempotent format).
2. `textedit` (168) + `cap` (110) — leaf helpers, unblock `fix`/`interp`.
3. `guide` (400) — small, low coupling.
4. `fix` (1060) — depends on `textedit`; autofix golden tests as the oracle.
5. `interp` (3722) — large but no exotic Go; goscript conformance suite is the
   oracle. Budget the most time here.
6. `lsp` (2292) — **last and hardest.** Uses `sync.Mutex`, `any`, JSON-RPC over
   stdio. Spike first: confirm goal can express goroutines + `sync` (the goal
   backend already *emits* channel/`go` constructs, but the goal *source* has
   never used them — verify the front-end accepts them). If goal can't yet
   express the concurrency `lsp` needs, that becomes a compiler-capability story
   (cf. the SEAM-CAP pattern) before the port.

### Phase 3 — The flip
- Point `cmd/goal`, `cmd/goalc` at the generated packages (already the same
  `goal/internal/*` paths under B-commit — this is mostly deleting the
  hand-written `.go` and regenerating).
- Delete the legacy hand-written Go for every ported package.
- Gate: full `go test ./...`, corpus behavioral tier, 3-stage bootstrap +
  fixpoint, all green. This is the irreversible point — do it as one reviewable
  PR per package or one big switch, but only after Phase 2 is 100%.

### Phase 4 — Cleanup and final layout
- Remove the old peer `selfhost/` tree (now living at `internal/compiler/`).
- Rewrite `Taskfile.yml` bootstrap/fixpoint targets to the new paths.
- Update `cmd/build-playground`, `cmd/corpus-gen` if their Go deps moved.
- Update CI, `README.md`, `REWRITE-ARCHITECTURE.md`, `SELF-HOST-RESEARCH.md`.

---

## 6. Explicit non-goals / what stays Go

- `corpus`, `byexample`, the `internal/selfhost` harness itself, and
  `cmd/{corpus-gen,build-playground}` are **test/dev infrastructure**. They do not
  ship in `goal`/`goalc` and need not be goal-sourced for the compiler to be
  self-hosted. Porting them is optional polish, not part of the flip.
- The Go reference compiler can be preserved on a tag/branch as the historical
  trust anchor even after deletion from `main`.

---

## 7. Top risks

1. **`lsp` concurrency may exceed current goal expressiveness** → could turn into
   a compiler-feature project, not just a port. Spike in Phase 2 before
   committing a timeline.
2. **Drift gate friction** — committed generated Go means every compiler change
   touches two trees; the `verify-generated` gate must be fast and reliable or it
   becomes a tax. Mitigate by making `task generate` the *only* way to edit the
   generated tree.
3. **Hidden test coupling** — `internal/{lsp,corpus,interp}` tests import each
   other and `fix`. Map the test-time import graph before Phase 3 so deleting
   hand-written Go doesn't strand a test.
4. **Trust regression** — losing the independent Go implementation removes the
   differential oracle. The fixpoint proves self-*consistency*, not
   *correctness*; the corpus behavioral tier becomes the primary correctness gate
   and must be kept comprehensive.

---

## 8. Recommended next step

If the full flip is the real objective: do **Phase 0 + Phase 1** first as a
self-contained PRD (committed bootstrap with legacy Go retained). That proves the
trust model end-to-end and is fully reversible, before sinking ~8k LOC into the
Phase 2 ports. If "stop selfhost being a second-class peer" is the *actual* itch,
Phase 1 may be the whole project.

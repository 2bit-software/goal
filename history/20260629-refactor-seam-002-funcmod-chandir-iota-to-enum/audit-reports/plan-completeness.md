# Plan Completeness Audit — SEAM-002 (FuncMod & ChanDir iota → goal enum)

Audit of `implementation-plan.md` for **coverage** against `business-spec.md`.
Scope: does every FR/AC map to a plan element; any plan element with no requirement
(scope creep); any AC with no testing strategy. All file paths and line references
were verified against the repo at `/Users/morgan/Projects/personal/goal`.

## Verdict

Coverage is **complete**. Every FR and every AC traces to at least one plan element.
No CRITICAL or MAJOR gaps. No genuine scope creep. Three MINOR findings, the
highest being an under-specified ChanDir row that drops two qualifiers the FuncMod
row carries.

## Requirement → Plan Traceability

### Functional Requirements

| Req | Plan element(s) | Status |
|-----|-----------------|--------|
| **FR-1** FuncMod is a goal enum (newline-separated tag variants, doc preserved) | Plan line 13: `selfhost/ast/goal_decl.goal` "`enum FuncMod {…}` (newline-separated tag variants; doc text preserved)" | Covered |
| **FR-2** ChanDir is a goal enum (same form, doc preserved) | Plan line 14: `selfhost/ast/ast.goal` "`enum ChanDir {…}`" | Covered, but see MINOR-1 (qualifiers dropped) |
| **FR-3** All consumers (cross- & same-pkg) use exhaustive match, no `_` arm; incl. ast.goal:170; tagless bool-switch stays | Plan covers all 7 consumer sites: ast.goal:170 (line 14), question.goal:210 (line 15), resolve.goal:218 tagless + resolve.goal:458 `x.Dir` (line 16), convert.goal:34 (line 17), emit.goal:361 + emit.goal:2309 (line 18). Tagless switch preserved per spec. | Covered |
| **FR-4** Construction uses qualified variant syntax; parser.goal:365 explicit FuncPlain | Plan line 19: parser.goal qualified spelling + `&ast.FuncDecl{Mod: ast.FuncMod.FuncPlain}`; Interface Contracts (lines 60-61) | Covered |
| **FR-5** token.Kind stays iota (documented refusal) | Plan line 21 (DECISIONS.md token.Kind refusal) + line 26 ("token.Kind stays iota") | Covered |
| **FR-6** Behavior preserved + zero-value invariant (both construction owners explicit) | Plan line 19 (parser.goal:365) + Testing Strategy `task fixpoint`; verified only two construction owners exist (parser.goal:365, :511) | Covered |

### Acceptance Criteria

| AC | Plan element(s) | Testing strategy present? |
|----|-----------------|--------------------------|
| AC: FuncMod/ChanDir declared as enums | goal_decl.goal + ast.goal rows | Yes — `task check`/`task build` compile |
| AC: no plain switch/`==`/`!=` over either type | all consumer rows; Plan lines 50-52 note §9 plain-switch-over-sealed-enum is a compile error | Yes — enforced by compiler via `task check` |
| AC: plain fn resolves FuncPlain + every ChanType has explicit Dir; end-to-end, no new unit test | parser.goal:365 + :511 rows; Testing Strategy | Yes — `task fixpoint` + parser/sema/backend port gates |
| AC: token.Kind stays iota; DECISIONS records refusal | Plan line 21 | Docs AC — no automated test (expected) |
| AC: `task check` green (after internal/ast relocation) | funcmod_test.go (new) + ast_test.go split (lines 20, 47); Testing Strategy | Yes — `task check` |
| AC: `task build` green | Testing Strategy gates (line 78) | Yes |
| AC: `task fixpoint` FIXPOINT OK | Testing Strategy gates | Yes |
| AC: corpus behavioral tier unchanged | Plan line 79 (`TestASTEngineWholeCorpusBehavioralGate` unchanged) | Yes |
| AC: DECISIONS records conversion + refusal | Plan line 21 | Docs AC — no automated test (expected) |

## Scope-Creep Check (plan elements with no requirement)

- `internal/ast/funcmod_test.go` (new) and the `internal/ast/ast_test.go` FuncMod-block
  removal — **NOT scope creep.** Traces to the AC "`task check` is green (after the
  internal/ast FuncMod-oracle relocation)". Verified the constraint is real:
  `internal/selfhost/port_test.go:142` compiles `../ast/ast_test.go` against the
  transpiled (enum) selfhost/ast, so the bare-iota FuncMod assertions
  (`Mod: FuncFrom`, lines 250-280) cannot survive there once selfhost/ast is an
  enum — they must move to an internal-only (Go-iota) test. Well-reasoned and in-spec.
- `prd.json` / `progress.txt` edits (plan lines 22-23) — house loop bookkeeping; no
  FR/AC. Standard process, flagged as informational only (MINOR-3).

## Findings

### MINOR-1 — ChanDir plan row drops the "newline-separated / doc-preserved" qualifiers
Plan line 13 (FuncMod) reads: `enum FuncMod { FuncPlain; FuncFrom; FuncDerive }`
**"(newline-separated tag variants; doc text preserved)"**. Plan line 14 (ChanDir)
reads only: `enum ChanDir { SendRecv; SendOnly; RecvOnly }` — with **no** newline
note and **no** doc-preservation note. FR-2 requires ChanDir use the "same declaration
form as FR-1, doc text preserved," and DECISIONS.md §01-enums mandates "Variant
separator: newline-separated, no trailing punctuation." The semicolon-joined inline
form shown for ChanDir, taken literally and without the FuncMod row's qualifiers,
risks an implementer dropping the `SendRecv`/`SendOnly`/`RecvOnly` doc comments
(verified present at ast.goal:675/677/679) and/or using the wrong separator.
Low risk because the FuncMod row models the correct form, but the asymmetry is a real
coverage gap for FR-2's doc-preservation clause.

### MINOR-2 — Documentation ACs have no verification step
The two DECISIONS.md ACs (token.Kind refusal recorded; FuncMod/ChanDir conversion
recorded) are produced by plan line 21 but have no entry in the Testing Strategy.
This is expected for documentation deliverables (no automated test is possible), noted
for completeness only.

### MINOR-3 — prd.json / progress.txt edits trace to no requirement
Plan lines 22-23 edit `prd.json` (`passes: true`) and `progress.txt`. These map to no
FR/AC. They are standard loop/house bookkeeping rather than spec scope, so not a defect;
flagged so the trace is exhaustive.

## Assumptions

- "Coverage" was assessed as **textual traceability** between spec requirements and
  plan elements, plus existence/accuracy of every file and line the plan cites — not
  whether the planned edits are themselves correct goal syntax or will compile.
- I assumed the spec's claim that `parser.goal` is the *sole* owner of FuncDecl/ChanType
  construction; I verified it by grep (only parser.goal:365 and :511 construct them in
  `selfhost/`), so the FR-6 zero-value invariant is fully bounded by the plan.
- I assumed `internal/ast/{ast.go,goal_decl.go}` and `selfhost/token/*` are
  intentionally untouched (plan lines 26-27, spec Out of Scope) and did not treat their
  absence from the change set as a gap.
- Line numbers in the spec/plan were checked against the current tree; the spec's
  "resolve.goal:216" is actually 217-218 and the ast_test.go FuncMod block is 250-280
  (spec/plan say ~247-284) — these are immaterial drifts, not findings, since the plan
  identifies sites descriptively, not by exact line.
- Documentation-only ACs are assumed not to require an automated testing strategy.

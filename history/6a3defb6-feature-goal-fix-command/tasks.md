# Tasks: `goal fix`

Complexity: **Medium-Complex** — ~6 Go source files, 1 enabling refactor, CLI wiring, plus
golden `testdata/fix/**`. Sequenced A→F (see implementation-plan.md). `[P]` = parallelizable
with siblings in the same phase. Each task is atomic and startable from fresh context.

## Phase A — Enabling refactor + CLI skeleton

- [ ] T001 [A1] Relocate `funcSpans`/`sigAt` (`internal/pass/pass.go`) → exported
  `analyze.FuncSpans`/`analyze.SigAt` and `zeroLit` (`internal/pass/defaults.go`) →
  `analyze.ZeroLit`; repoint all `internal/pass` callers. No behavior change.
  **AC**: `go test ./internal/...` green; no remaining unexported refs in `pass`.
- [ ] T002 [A2] Create `internal/fix/fix.go`: types `Change`, `Report`, `Level`
  (Suggest|Warn|Skip), and `File(src string, t *analyze.Tables) (out string, changes
  []Change, reports []Report, err error)` returning `src` unchanged for now.
  **AC**: package compiles; unit test: idiomatic input returns byte-identical output.
- [ ] T003 [A3] CLI wiring in `cmd/goal/main.go`: add `fix` to `guideCommands`, `case
  "fix"` in `run()`, `parseFixFlags(args) (paths []string, inplace bool, err error)`
  (`-inplace`, file-or-dir, default `.`), and `cmdFix` (Discover → `fix.File` → stdout
  default / `-inplace` writes changed files only / reports → stderr). Depends: T002.
  **AC**: `goal fix <dir>` prints discovered files; `goal fix --help`/usage lists `fix`.
- [ ] T004 [A4] CLI tests in `cmd/goal/main_test.go` (`goalModule`+`run`): no-op file
  prints unchanged & writes nothing; `-inplace` on unchanged file does not rewrite; bad
  path → non-zero exit. Depends: T003. **AC**: tests pass; FR-003/004/013/014 covered.

## Phase B — Propagation collapse (`internal/fix/propagate.go`) — [US1]

- [ ] T005 [B1] Result keep/discard collapse (DR-1/DR-2): in `ModeResult` spans, match
  `v, err := g(); if err != nil { return <zeros>, err | Result.Err(err) }` → `v := g()?`
  (and `_ := g()?`), using `analyze.SigAt` + `analyze.ZeroLit`. Strict match; deviations →
  skip+Report. Depends: T002.
- [ ] T006 [B2] Option collapse + `*o`→`v` rewrite (DR-4) in `ModeOption` spans; escaping
  `o` → skip+Report. Depends: T005 (shared matcher helpers).
- [ ] T007 [P] [B3] Safety guards: DR-5 (comment in edited span via raw-source scan → skip)
  and DR-6 (multi-line RHS → skip). Depends: T005.
- [ ] T008 [B4] Golden tests `internal/fix/testdata/fix/propagate/{before,after}.goal`
  (keep/discard/option) + negatives (wrapped, decorated, comment, multi-line, escaping
  pointer); idempotence + transpile-oracle (`lower(after)` valid). Depends: T005–T007.
  **AC**: FR-005/010/011 + SC-001 covered; all golden + negative cases pass.

## Phase C — Signature conversion (`internal/fix/resultsig.go`) — [US2]

- [ ] T009 [C1] Detect convertible `(T, error)` functions (DR-3 whole-function check: every
  early return is a bare-propagation zero per `analyze.ZeroLit`; success return single-
  valued). Non-conforming/multi-value/error-only → Skip+Report `file:line`. Depends: T001.
- [ ] T010 [C2] Rewrite signature `(T,error)`→`Result[T,error]` (bracket-aware `T`
  extraction) and `return v, nil`→`return Result.Ok(v)`; set `signatureChanged`. Depends:
  T009.
- [ ] T011 [C3] Fixed-point loop in `fix.File`: re-lex + rebuild tables when
  `signatureChanged`, bounded iterations; orders B-after-C. Depends: T005, T010.
  **AC**: FR-011 fixed point; converted body's props collapse via Phase B.
- [ ] T012 [C4] Golden + CLI: spike `before.goal`→`after.goal` asserting `lower(after) ==
  lower(before)`; exported-fn signature change → stderr warning (DR-9/FR-008). Depends:
  T011. **AC**: FR-006/008 + SC-003 covered; user headline example works end-to-end.

## Phase D — Call-site reporting (`internal/fix/callsite.go`) — [US2]

- [ ] T013 [D1] Enumerate call sites of converted functions; legal in-package
  Result/Option callers collapse via Phase B; illegal sites (non-Result caller,
  closure/`defer`/`go`) → Suggest "manual follow-up" (DR-7/FR-007). Depends: T011.
- [ ] T014 [D2] Tests: same-package Result-fn caller collapses to `?`; non-Result caller
  left unchanged + reported. Depends: T013. **AC**: FR-007 covered.

## Phase E — switch→match (`internal/fix/match.go`) — [US3, stretch]

- [ ] T015 [P] [E1] `switch` over in-file enum (`t.Enums` via `scan.LeadIdent`/case
  qualifier) → `match`, preserve arm bodies, drop `break` (DR-8); non-enum switch
  untouched. Depends: T002.
- [ ] T016 [P] [E2] Golden tests for switch→match + non-enum negative. Depends: T015.
  **AC**: FR-009 covered.

## Phase F — Polish & docs

- [ ] T017 [F1] Cross-rule idempotence/fixed-point test on a kitchen-sink fixture; verify
  SC-002/SC-005 (default writes nothing; `-inplace` writes only changed files). Depends:
  T008, T012, T014.
- [ ] T018 [P] [F2] Docs: README `goal fix` subcommand entry + `goal ai` guide line +
  `--help` usage text. Depends: T003.
- [ ] T019 [F3] Full `go test ./...` + `goal check` on fixtures; confirm zero-dependency and
  no `internal/pass` regression. Depends: all.

## Dependency summary / critical path

Critical path: **T001 → T002 → T003 → T005 → T010 → T011 → T012 → T017 → T019**.
Parallelizable: T007 with T006; T015/T016 (Phase E) anytime after T002; T018 after T003.

## Traceability (every FR/SC mapped)

- FR-001/002→T003 · FR-003/013/014→T003/T004 · FR-004→T003/T004/T017
- FR-005→T005-T008 · FR-006→T009-T012 · FR-007→T013/T014 · FR-008→T012
- FR-009→T015/T016 · FR-010→T008 · FR-011→T011/T017 · FR-012→T005/T009 (skip+report)
- SC-001→T008 · SC-002→T017 · SC-003→T012 · SC-004→all skip+report tasks · SC-005→T017
- US1→T005-T008 · US2→T009-T014 · US3→T015/T016

No orphan tasks; every task traces to a plan phase (A–F).

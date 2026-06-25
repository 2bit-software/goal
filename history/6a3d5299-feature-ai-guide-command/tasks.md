# Tasks — `goal ai`

Traces to `implementation-plan.md` (Phases 1–6), `technical-spec.md`, and `spec.md`
(FR-1…FR-13 / US1–US4).

## Complexity

- Files affected: ~16 (new: `assets.go`, `internal/byexample/*`, `internal/guide/*`,
  `docs/ai/*`; modified: `cmd/goal/main.go`, `cmd/goal/main_test.go`,
  `cmd/build-playground/main.go`, `AI-KNOWLEDGE-BOOTSTRAP.md`).
- Cross-module deps: byexample ← (build-playground, guide); guide ← (root `goal`, pipeline,
  check); cmd/goal ← guide.
- **Classification: Medium–Complex (boundary).** Single list (≤20 tasks), staged by phase.

## Critical path

T01 (parser lift) → T02 (build-playground refactor + features.json guard) → T05 (root embed)
→ T06 (guide assembler) → T07–T08 (catalog + feedback sample) → T11 (golden) → T15 (regen
bootstrap). Authored fragments (T09) and CLI registry (T03–T04) run in parallel with the
parser/guide spine.

## Phase 1 — Shared parser

- [ ] T01 [US1] Create `internal/byexample/byexample.go`: move the parse layer out of
  `cmd/build-playground/main.go` (`splitSections`, `section`, `hasNamedGoalBlock`,
  `collectDescription`, `readGoalExample`, `readOutputBlock`, `readFenceBody`, `slugify`,
  `stripLeadingNumber`, `capitalizeFirst`, fence regexes). Expose `Parse(doc, docPath string)
  (Doc, error)` returning `Feature` with **raw-markdown** `DescriptionMD`/`LoweringMD`
  fields + `Source`, `SourceName`, `OutputKind`, `LockedExpected` per technical-spec.
  *Accept*: package builds; `Parse` returns categories+features from `docs/by-example.md`.

- [ ] T02 [US1] Refactor `cmd/build-playground/main.go` to consume `byexample.Parse`, keeping
  its `renderMarkdown`/`renderInline` (fed `DescriptionMD`/`LoweringMD`) and `verify(Source,
  LockedExpected, OutputKind)`. *Depends*: T01. *Accept*: `go run ./cmd/build-playground`
  produces a `site/features.json` **byte-identical** to the committed one (diff is empty).

- [ ] T03 [P] [US1] `internal/byexample/byexample_test.go` (stdlib `testing`): parse the
  repo's `docs/by-example.md`; assert feature count ≥ 13, and each feature has non-empty
  `Source` + `SourceName`. *Depends*: T01.

## Phase 2 — CLI registry + entry points

- [ ] T04 [US2] In `cmd/goal/main.go`, introduce the `command`/`flagDoc` registry
  (`commands = []command{build,run,check,ai}`) and refactor `run()` to dispatch via it,
  preserving current `build`/`run`/`check` behavior. *Accept*: existing
  `cmd/goal/main_test.go` still passes.

- [ ] T05 [US1] Add `aiCmd` to the registry and the `--ai` alias: at the top of `run()`,
  rewrite a leading `--ai` to the `ai` command; `ai` takes an optional section-name arg (not
  a path). `aiCmd.Run` calls `guide.Render(out, section)`. Update top-level usage to list
  `ai`. *Depends*: T04, T06. *Accept*: `goal ai` and `goal --ai` both run `guide.Render`.

## Phase 3 — Root embed + assembler

- [ ] T06 [US1] Create root `./assets.go` (`package goal`) with
  `//go:embed docs/by-example.md docs/overview.md docs/ai` → `var Docs embed.FS`. *Accept*:
  `go build ./...` succeeds; `goal.Docs` readable.

- [ ] T07 [US1] Create `internal/guide/guide.go`: `Render(w io.Writer, section string) error`.
  Read assets from `goal.Docs`, parse via `byexample.Parse`, and render sections in fixed
  order (intro←overview.md, toolchain←registry+`docs/ai/toolchain.md`, features←parsed+live
  `pipeline.Transpile(Source)`, diagnostics, starter, conventions, authoring, pointers).
  Error-kind features show the real `Transpile` error. *Depends*: T01, T06. *Accept*:
  `Render(&buf, "")` returns non-empty markdown covering every feature with live lowerings.

- [ ] T08 [US1] Section selection: `section != ""` renders only that named section; unknown
  name returns an error listing valid names (FR-12). *Depends*: T07.

## Phase 4 — Diagnostics + authored fragments

- [ ] T09 [US2] Create `internal/guide/catalog.go`: `diagnosticCatalog []diagDoc{Code,Feature,
  Meaning,Severity}` with one authored entry per live code; render sorted by `Code`.
  *Source of codes*: the ~25 `Code:` literals in `internal/check` + `internal/typecheck`.

- [ ] T10 [US2] Live feedback sample: add `internal/guide/testdata/feedback_sample.goal` (a
  snippet that trips e.g. `non-exhaustive-match` + `missing-field`), embed it, and in the
  diagnostics section run `check.AnalyzePackage([]string{sample})` rendering real findings as
  `file:line:col: severity: [code] message` (reuse the render format from
  `cmd/goal/main.go`). *Depends*: T07.

- [ ] T11 [P] [US1] Authored fragments under `docs/ai/`: `toolchain.md`, `conventions.md`,
  `authoring.md`, `pointers.md`, and `starter.goal` (or decide to reuse the composition
  example). Source prose from current `AI-KNOWLEDGE-BOOTSTRAP.md` §2/§6, corrected to use
  `__gop_` (not `__goal_`). *Depends*: T06 (embed paths).

## Phase 5 — Drift guards

- [ ] T12 [P] [US1] `internal/guide/guide_test.go::TestLiveLowerings`: for every parsed
  feature, assert `pipeline.Transpile(Source)` succeeds (or fails for error-kind). *Depends*:
  T07.

- [ ] T13 [P] [US2] `...::TestDiagnosticCatalogMatchesSource`: regex-scan
  `internal/check` + `internal/typecheck` `.go` for `Code:\s*"([a-z-]+)"`; assert literal set
  == `diagnosticCatalog` key set; report missing/extra (FR-7). *Depends*: T09.

- [ ] T14 [P] [US2] `...::TestToolchainListsAllCommands`: render the toolchain section; assert
  each `commands[i].Name` appears and the count matches (FR-6). *Depends*: T05, T07.

- [ ] T15 [P] [US1] Add to `cmd/goal/main_test.go`: `goal ai` and `goal --ai` exit 0, emit
  non-empty markdown to stdout, write nothing to the tree; unknown `goal ai <bad>` errors
  with the valid-name list. *Depends*: T05, T08.

## Phase 6 — Golden artifact

- [ ] T16 [US3] Regenerate `AI-KNOWLEDGE-BOOTSTRAP.md` via
  `go run ./cmd/goal ai > AI-KNOWLEDGE-BOOTSTRAP.md`; add a header line noting it is generated
  and how to regenerate. *Depends*: T07–T11.

- [ ] T17 [US3] `internal/guide/guide_test.go::TestBootstrapGolden`: `Render(&buf, "")` must be
  byte-equal to the committed `AI-KNOWLEDGE-BOOTSTRAP.md`; on mismatch fail with the regen
  command (FR-11). *Depends*: T16.

## Phase 7 — Final validation

- [ ] T18 [US1] Full-suite + manual smoke: `go test ./... -count=1`; `go run ./cmd/goal ai`
  renders cleanly; build a tiny module per the guide's starter section and confirm
  `goal run` executes it. *Depends*: all.

## Traceability (spec → tasks)

| Requirement | Tasks |
|---|---|
| FR-1/FR-2 (command + alias, stdout, no writes) | T05, T15 |
| FR-3 (by-example source, shared parser) | T01, T02, T07 |
| FR-4/FR-5 (live lowering / live error) | T07, T12 |
| FR-6 (toolchain from registry) | T04, T05, T14 |
| FR-7 (diagnostics catalog drift guard) | T09, T13 |
| FR-8 (live feedback sample) | T10 |
| FR-9 (embedded, no runtime disk reads) | T06, T07 |
| FR-10 (feature count derived) | T07, T03 |
| FR-11 (golden bootstrap) | T16, T17 |
| FR-12 (section access) | T08, T15 |
| FR-13 (zero-dep / stdlib tests) | all (no new deps); T03/T12–T17 use stdlib `testing` |

No orphan tasks; every task traces to a plan phase and ≥1 requirement.

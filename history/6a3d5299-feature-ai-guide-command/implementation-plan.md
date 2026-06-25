# Implementation Plan — `goal ai`

Traces to `spec.md` (FR-1…FR-13) and `technical-requirements-research.md`. Decisions on the
three open questions are resolved per user sign-off: **OQ-1 include section access**,
**OQ-2 `docs/ai/` fragments**, **OQ-3 lexical-stage feedback sample for v1**.

## Inline reuse audit (no duplication)

| Planned item | Existing code | Decision |
|---|---|---|
| Doc → feature parser | `cmd/build-playground/main.go` parse layer (`splitSections`, `parseFeature`, `readGoalExample`, `readOutputBlock`, helpers) | **Extract** to `internal/byexample`; both consumers import it |
| Markdown→HTML renderer | `renderMarkdown`/`renderInline` in build-playground | **Leave** in build-playground (HTML is playground-only); guide renders Markdown directly from raw fields |
| Live transpile | `pipeline.Transpile` | **Reuse** as-is |
| Checker (lexical) + diagnostic render | `check.AnalyzePackage`, `checkDiag.render` (`cmd/goal/main.go:255`) | **Reuse**; lift `render` format into shared helper if needed |
| Intro prose | `docs/overview.md` | **Reuse** (embed; already correct re: checker) |
| Per-feature content | `docs/by-example.md` | **Reuse** (embed) |
| Drift-guard test pattern | build-playground `verify()` | **Mirror** for the diagnostics catalog + AI-bootstrap golden |

No new third-party dependency (FR-13). All new tests use stdlib `testing`.

## Work breakdown (dependency-ordered)

### Phase 1 — Extract the shared parser (`internal/byexample`)
1. Create `internal/byexample/byexample.go`. Move the *parse* layer out of
   `cmd/build-playground/main.go`: `splitSections`, `section`, `hasNamedGoalBlock`,
   `collectDescription`, `readGoalExample`, `readOutputBlock`, `readFenceBody`, `slugify`,
   `stripLeadingNumber`, `capitalizeFirst`, and the fence regexes.
2. Define the exported result type carrying **raw markdown**, not HTML:
   ```go
   type Feature struct {
       Anchor, Title       string
       Category            string
       DescriptionMD       []string // raw doc lines before the example
       Source, SourceName  string
       LoweringMD          []string // raw doc lines after the output block
       OutputKind          string   // "go" | "test" | "error"
       LockedExpected      string   // the doc's locked block (advisory for the guide)
   }
   type Doc struct { Title, GeneratedFrom string; Categories []Category }
   ```
   Expose `Parse(doc, docPath string) (Doc, error)`.
3. **Refactor `cmd/build-playground` to consume it**: it keeps `renderMarkdown`/`renderInline`
   and `verify()`, now feeding them `Feature.DescriptionMD`/`LoweringMD`. Output
   `site/features.json` must be **byte-identical** before/after (regression guard below).
4. Test `internal/byexample/byexample_test.go`: parse the embedded `docs/by-example.md`,
   assert ≥ the known feature count and that each feature has a non-empty `Source` and
   `SourceName`.

### Phase 2 — Command registry + `ai` command in `cmd/goal`
5. Introduce a command registry in `cmd/goal`:
   ```go
   type flagDoc struct{ Name, Summary string }
   type command struct{ Name, Summary, Usage string; Flags []flagDoc; Run func(...) error }
   var commands = []command{ build, run, check, ai }
   ```
   Refactor `run()` (`main.go:38`) to dispatch via the registry instead of the hardcoded
   `switch` (FR-6). Keep behavior identical for build/run/check (covered by `main_test.go`).
6. Wire the `ai` command and the `--ai` alias: in `run()`, if `args[0] == "--ai"`, dispatch
   `ai` (FR-2). `ai` accepts an optional section name argument (FR-12).
7. Update the top-level usage string to include `ai`.

### Phase 3 — The guide assembler (`internal/guide`)
8. Create `internal/guide/guide.go` with `Render(w io.Writer, section string) error`
   (`section == ""` → full guide). Embed assets with `go:embed`:
   - `docs/by-example.md`, `docs/overview.md`
   - `docs/ai/*.md` (authored fragments: toolchain, conventions, authoring, deeper-pointers)
   - `internal/guide/testdata/feedback_sample.goal` (the embedded bad snippet)
   (Note: `go:embed` paths are package-relative; copy/symlink the needed docs under the
   package dir, or place the assembler in `cmd/goal` where the embed root can reach `docs/`.
   Resolve embed-root mechanics in technical-spec.)
9. Assemble sections in fixed order (deterministic for the golden test, FR-11):
   1. **Intro** ← `docs/overview.md` (rendered as Markdown, lightly massaged for CLI).
   2. **Toolchain** ← command registry (commands, flags, iteration loop). Iteration-loop
      prose ← `docs/ai/toolchain.md`.
   3. **Features** ← `byexample.Parse`; for each feature emit prose (`DescriptionMD`), the
      `Source`, and **the live `pipeline.Transpile(Source)` output** (FR-4). Error-kind
      features show the real `Transpile` error message (FR-5). Feature count/order derived
      from parse (FR-10).
   4. **Diagnostics** ← authored catalog (`docs/ai/` or a Go table) rendered + the **live
      checker sample**: `check.AnalyzePackage([]string{embeddedBad})` rendered in
      `file:line:col: severity: [code] message` (FR-8).
   5. **Starter program** ← a `name=`-tagged example (reuse the composition or a dedicated
      `docs/ai/starter.goal`), shown with live transpile + the exact CLI invocations.
   6. **Locked conventions** ← `docs/ai/conventions.md`.
   7. **Authoring do/don'ts** ← `docs/ai/authoring.md`.
   8. **Deeper pointers** ← `docs/ai/pointers.md` (links to by-example, design-spec, DECISIONS).
10. Diagnostics catalog: a sorted Go table `var diagnosticCatalog = []diagDoc{{Code, Feature,
    Meaning, Severity}}` embedded in `internal/guide`. Rendered deterministically (sorted by
    code).

### Phase 4 — Authored fragments
11. Write `docs/ai/toolchain.md`, `conventions.md`, `authoring.md`, `pointers.md`,
    `starter.goal`, and `internal/guide/testdata/feedback_sample.goal`. Source the prose from
    the current `AI-KNOWLEDGE-BOOTSTRAP.md` §2/§6 (corrected: `__gop_` not `__goal_`).

### Phase 5 — Drift guards + golden artifact
12. **Live-lowering test** (FR-4 acceptance): for each parsed feature, assert
    `pipeline.Transpile(Source).Go` (or `.Test`/error) equals what the guide renders — i.e.
    the guide's feature rendering is itself driven by `Transpile`, and a test asserts no
    feature example fails to transpile. (This also re-proves the doc's locked blocks via the
    retained build-playground `verify`.)
13. **Diagnostics-catalog drift test** (FR-7): scan `internal/check` + `internal/typecheck`
    source for `Code:\s*"..."` literals; assert the set equals `diagnosticCatalog`'s key set.
    Fail with a message naming missing/extra codes.
14. **Toolchain drift** (FR-6): the toolchain section renders from `commands`; a test asserts
    every `commands[i].Name` appears in the rendered output and the count matches.
15. **AI-bootstrap golden** (FR-11): regenerate `AI-KNOWLEDGE-BOOTSTRAP.md` via
    `go run ./cmd/goal ai > AI-KNOWLEDGE-BOOTSTRAP.md`; a test runs `guide.Render` to a buffer
    and asserts byte-equality with the committed file. Document the regen command in
    `docs/ai/` and the file header.
16. **features.json regression** (Phase 1 safety): a test (or manual check noted in tasks)
    that build-playground still emits identical `site/features.json`.

### Phase 6 — Retire the hand-maintained file
17. Replace `AI-KNOWLEDGE-BOOTSTRAP.md` content with generated output; add a header line
    noting it is generated by `goal ai` and how to regenerate. Keep it committed (US3).

## Sequencing & dependencies
- Phase 1 blocks Phase 3 (parser) and must keep `features.json` stable.
- Phase 2 blocks the toolchain section (registry) and the `--ai`/`ai` entry points.
- Phases 3–4 are co-developed (assembler needs the fragments).
- Phase 5 tests land with the code they guard.
- Phase 6 is last (depends on stable output for the golden test).

## Risks / mitigations
- **`go:embed` root reach** (RESOLVED in technical-spec): `embed` can't climb above its
  package dir, and `docs/` is at the module root. Resolution: a **root-level `package goal`**
  (`./assets.go`) embeds `docs/...` and exposes an `embed.FS`; `internal/guide` imports it. No
  staged copies, no duplication. Verified: no root `.go` files exist, module path is `goal`.
- **build-playground output drift during refactor**: lock with a before/after diff of
  `site/features.json`.
- **Determinism**: sort the diagnostics catalog; fixed section order; no maps in render paths.

## Open items for tasks step
- Decide whether the starter program reuses the composition example or a dedicated one.
- Whether to add a `go generate` directive for both the staged assets and the bootstrap file.

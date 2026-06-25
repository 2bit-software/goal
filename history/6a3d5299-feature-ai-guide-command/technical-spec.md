# Technical Spec — `goal ai`

Implementation design for `implementation-plan.md`. Resolves the embed-root risk and pins
package layout, types, and test shapes. Zero third-party deps; stdlib `testing` only.

## Embed strategy (resolved)

`go:embed` patterns are relative to the embedding file's directory and **cannot use `..`**.
`docs/` lives at the module root; `cmd/goal` and `internal/guide` are subdirectories and
cannot reach it. Resolution: **a root-level package** at the module root (`package goal`,
where `go.mod` is) — it can embed anything under the repo root with no `..`.

Create `./assets.go`:
```go
// Package goal exposes embedded repo assets to internal tooling (the AI guide).
package goal

import "embed"

//go:embed docs/by-example.md docs/overview.md docs/ai
var Docs embed.FS
```
`internal/guide` imports the module-root package `goal` and reads `goal.Docs`. (An internal
package may import its module-root parent.) No staged copies, no duplication, single source.

Verified preconditions: no existing root `.go` files; module path is `goal`; nothing imports
bare `"goal"` yet — so adding the root package is non-breaking.

## Package layout

```
./assets.go                       // package goal — embed.FS of docs/
internal/byexample/byexample.go   // lifted parser, raw-markdown Feature records
internal/byexample/byexample_test.go
internal/guide/guide.go           // assembler/renderer: Render(w, section)
internal/guide/catalog.go         // diagnosticCatalog table + render
internal/guide/guide_test.go      // live-lowering, golden, toolchain, catalog-drift tests
internal/guide/testdata/feedback_sample.goal
docs/ai/toolchain.md              // authored: iteration loop / toolchain prose
docs/ai/conventions.md            // authored: locked conventions
docs/ai/authoring.md              // authored: do/don'ts
docs/ai/pointers.md               // authored: deeper references
docs/ai/starter.goal              // starter program example (or reuse composition)
cmd/goal/main.go                  // command registry + ai command + --ai alias
cmd/build-playground/main.go      // now imports internal/byexample for parsing
AI-KNOWLEDGE-BOOTSTRAP.md         // generated golden artifact
```

## `internal/byexample` — lifted parser

Moves the parse layer verbatim from `cmd/build-playground/main.go` (sections, fence reading,
slug/number helpers, regexes). Key change: expose **raw markdown lines**, not HTML.

```go
type Feature struct {
    Anchor, Title, Category string
    DescriptionMD []string   // doc lines before the example (raw)
    Source, SourceName string
    LoweringMD    []string   // doc lines after the output block (raw)
    OutputKind    string     // "go" | "test" | "error"
    LockedExpected string    // the doc's locked block (advisory for the guide; still used by build-playground.verify)
}
type Category struct { Name string; Features []Feature }
type Doc struct { Title, GeneratedFrom string; Categories []Category }

func Parse(doc, docPath string) (Doc, error)
```

`cmd/build-playground` refactor: replace its inline parse with `byexample.Parse`, then run its
existing `renderMarkdown(DescriptionMD)` / `renderMarkdown(LoweringMD)` and `verify(Source,
LockedExpected, OutputKind)` to produce the unchanged `features.json`. **Acceptance: byte-for-
byte identical `site/features.json` before/after** (diff in tasks/verify).

## `internal/guide` — assembler

```go
func Render(w io.Writer, section string) error // section=="" → full guide
```
- Reads `goal.Docs` for `docs/overview.md`, `docs/by-example.md`, `docs/ai/*`.
- Parses by-example via `byexample.Parse`.
- For each feature: prints `DescriptionMD` (as-is markdown), a fenced `goal` block of
  `Source`, then a fenced `go` block of **`pipeline.Transpile(Source).Go`** (or `.Test` for a
  doctest example, or the `Transpile` error text for `OutputKind=="error"`). The shown
  lowering is therefore produced live (FR-4/FR-5).
- Section order is fixed (intro, toolchain, features, diagnostics, starter, conventions,
  authoring, pointers) for deterministic output (FR-11). Named sections map to these keys
  (FR-12); unknown name → error listing valid keys.

### Diagnostics catalog (`catalog.go`)
```go
type diagDoc struct{ Code, Feature, Meaning string; Severity string }
var diagnosticCatalog = []diagDoc{ /* one per live code, authored Meaning */ }
```
Rendered sorted by `Code`. The live feedback sample runs
`check.AnalyzePackage([]string{string(feedbackSampleGoal)})` and renders each finding as
`file:line:col: severity: [code] message` (reuse the format from `cmd/goal/main.go`
`checkDiag.render`; extract to a shared helper if convenient). Lexical stage only for v1
(OQ-3) — note in prose that the typed depth stage adds more on a real package.

## `cmd/goal` — command registry + entry points

```go
type flagDoc struct{ Name, Summary string }
type command struct {
    Name, Summary, Usage string
    Flags []flagDoc
    Run   func(root string, emit bool, emitDir string, out, errOut io.Writer) error
}
var commands = []command{buildCmd, runCmd, checkCmd, aiCmd}
```
- `run()` looks up `args[0]` in `commands`; preserves current `build`/`run`/`check` behavior
  (guarded by existing `cmd/goal/main_test.go`).
- `--ai` alias: at the top of `run()`, if `args[0] == "--ai"`, rewrite to the `ai` command
  (with the rest as section arg).
- `aiCmd.Run` ignores emit flags and calls `guide.Render(out, section)`. Optional section
  comes from the path-position arg (reuse/relax `parseFlags`, or parse `ai`'s args separately
  since it takes a section name, not a path).
- Usage strings (top-level and per-command) are rendered by the toolchain section, so they are
  the single source the guide reads (FR-6).

## Tests (stdlib `testing`)

1. `byexample_test.go` — parse embedded doc; assert feature count ≥ known and required fields
   present.
2. `guide_test.go::TestLiveLowerings` — for every parsed feature, `pipeline.Transpile(Source)`
   must succeed (or, for error-kind, fail), proving no guide example is broken (FR-4 accept).
3. `guide_test.go::TestDiagnosticCatalogMatchesSource` — regex-scan `internal/check` and
   `internal/typecheck` `.go` files for `Code:\s*"([a-z-]+)"`; assert the literal set equals
   the catalog key set; report missing/extra (FR-7).
4. `guide_test.go::TestToolchainListsAllCommands` — render toolchain section; assert each
   `commands[i].Name` present and count matches (FR-6).
5. `guide_test.go::TestBootstrapGolden` — `guide.Render(&buf, "")`; assert byte-equal to
   committed `AI-KNOWLEDGE-BOOTSTRAP.md`; on mismatch, fail with the regen command (FR-11).
6. `cmd/goal/main_test.go` additions — `goal ai` and `goal --ai` exit 0, emit non-empty
   markdown to stdout, write nothing; unknown section errors (FR-1/2/12).
7. Manual/CI check — `site/features.json` unchanged after the parser lift (Phase 1 guard).

## Determinism rules (for the golden test)
- No maps in any render path; catalog sorted by code; section order fixed.
- Live transpile output is deterministic (already true — `pipeline.Transpile` is pure).
- Bootstrap regen command, documented in the file header: `go run ./cmd/goal ai > AI-KNOWLEDGE-BOOTSTRAP.md`.

## Notes
- Prefix truth (CORRECTED during impl): the live transpiler emits **`__goal_`** (see
  `internal/pass/pass.go` constants `okName="__goal_ok"` etc.). The committed `site/features.json`
  showing `__gop_` was STALE; regenerating it via the refactored build-playground restored
  `__goal_`. Authored fragments and the bootstrap correctly use `__goal_`.
- build-playground keeps `verify()` so the doc's locked blocks still can't rot, even though
  the guide ignores them in favor of live output.

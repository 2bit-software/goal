# Technical Requirements & Research — `goal ai`

This captures the codebase facts the spec depends on and the technical shape of the
implementation, so planning starts from ground truth rather than inference. Verified against
the checkout on 2026-06-25.

## Existing assets the design reuses

### 1. `docs/by-example.md` is already a live-verified per-feature source
`cmd/build-playground/main.go` parses it into per-feature records (prose, a `name=`-tagged
```goal``` example, a "Transpiles to" ```go``` block, or a "Rejected with" ```error``` block)
**and** re-runs `pipeline.Transpile` on every example, asserting the locked output matches
(`verify()`, `cmd/build-playground/main.go:323`). This is the exact "generated from real
capabilities" property requested — it already exists, pointed at `site/features.json`.

**Implication**: the parser must be **lifted out of `cmd/build-playground` into a shared
internal package** (proposed `internal/byexample`) exposing parsed `Feature` records. Both
`cmd/build-playground` (HTML + manifest) and the new `goal ai` renderer (Markdown + live
transpile) consume it. This satisfies the "one parser, not two" acceptance criterion.

The parser, markdown renderer, and helpers to lift live in `cmd/build-playground/main.go`
lines ~102–507 (section splitting, `parseFeature`, `readGoalExample`, `readOutputBlock`,
`renderMarkdown`, `slugify`, etc.). HTML rendering (`renderMarkdown`/`renderInline`) stays
with the playground; the *parse* layer moves.

### 2. The live transpiler entry point
`pipeline.Transpile(src string) (Output, error)` (`internal/pipeline/pipeline.go:78`) returns
`Output{Go, Test}`. For an intentional-error feature, `Transpile` returns an error whose
`.Error()` is the located message (see `verify`'s error branch). `goal ai` calls this per
example at output time — N≈15 transpiles, in-process, milliseconds. No module/disk needed for
single-snippet lowering (matches how `goalc` and the playground WASM operate).

### 3. Feature/pass registry (a second live signal)
`pipeline.Passes` (`internal/pipeline/pipeline.go:52`) is an ordered `[]Pass{Name,Run}`.
Note it is **not 1:1 with user-facing features**: e.g. `storedresult` is a guard, `match` and
`enums` are two passes for what users see as "enums + match". So the authoritative
*user-facing* feature list for the guide is the parsed `by-example.md` categories/features
(FR-10), optionally cross-checked against passes in a loose table-driven test, not equated.

### 4. Diagnostic codes are string literals across two packages
`grep 'Code:' internal/check internal/typecheck` yields ~25 stable codes (e.g.
`non-exhaustive-match`, `missing-field`, `discarded-result-error`, `unimplemented-method`,
`dropped-stored-result`, …). They are declared inline at emit sites, not in a registry.

**Chosen approach (lighter, matches codebase philosophy)**: an authored catalog
(`code → feature, one-line meaning, severity`) embedded for rendering, plus a **set-equality
test** that scans `internal/check` + `internal/typecheck` source for `Code:` literals and
asserts the catalog's key set equals the live set. Adding a code without cataloguing it (or
removing one) fails the test. This mirrors the existing build-playground drift guard.
(The heavier alternative — a central `internal/diag` registry every emit site references — is
out of scope per the spec.)

### 5. CLI structure today
`cmd/goal/main.go` dispatches via a hardcoded `switch cmd` over `build|run|check`
(`main.go:47`) and a separate `parseFlags` (`main.go:62`) recognizing `--emit[=dir]`.
There is **no command registry and no global-flag handling** yet.

**Implication**: introduce a small in-package command registry (slice of
`{Name, Summary, Usage, Flags []flagDoc}`) that `run()`'s dispatch reads (so the registry is
the single source the toolchain section renders from, FR-6). Add the `ai` command and wire
the `--ai` alias (`run` should treat a leading `--ai` as the `ai` command). Keep it
hand-rolled — **no `urfave/cli`** (project is zero-dependency; the global Go standard
suggesting urfave/cli is overridden by the project's [[no-testify-goal]] zero-dep rule).

### 6. `goalc` single-file path (reference for live transpile + checker sample)
`cmd/goalc/main.go` runs checker (lexical) then lowering on one file to stdout. The "what
feedback looks like" sample (FR-8) follows the same single-file lexical-stage path:
`check.AnalyzePackage([]string{badSrc})` over an embedded bad snippet, rendered in the
`file:line:col: severity: [code] message` format already implemented by `checkDiag.render`
(`cmd/goal/main.go:255`). The typed depth stage needs a real package/module, so v1 uses the
lexical stage for the sample (OQ-3).

## Proposed component shape (for planning, not binding)

- `internal/byexample/` — lifted parser; exposes `Parse(doc string) ([]Category, error)` with
  `Feature{Anchor,Title,DescriptionMD,Source,SourceName,LoweringMD,ExpectedKind,...}` holding
  **markdown/raw** (not HTML). build-playground adds its HTML rendering on top.
- `internal/guide/` (or `cmd/goal/ai.go`) — the assembler/renderer: embeds the doc + authored
  fragments via `go:embed`, parses, live-transpiles each example, renders Markdown sections in
  order (intro → toolchain → features → diagnostics → starter → conventions → authoring →
  deeper-pointers). Pure string output to an `io.Writer`.
- `cmd/goal/main.go` — command registry; `ai` command; `--ai` alias; `goal ai <section>`.
- Embedded assets: `docs/by-example.md`, `docs/overview.md` (intro), `docs/ai/*.md`
  (authored sections), embedded bad snippet for the feedback sample.
- Generator + golden test for `AI-KNOWLEDGE-BOOTSTRAP.md` (FR-11), e.g. a test that runs the
  assembler and compares to the committed file (regenerate via a documented command).

## Constraints (from project memory and conventions)

- **Zero third-party dependencies**; tests use stdlib `testing` only ([[no-testify-goal]]).
  Overrides the global "use testify/urfave/cli" Go standard.
- Go **1.26+**, module-based. `go:embed` is available and already idiomatic here.
- Synthesized-temp prefix is `__gop_` in current output (note: the bootstrap file's prose says
  `__goal_`; the live output in `features.json` uses `__gop_`). This *is* an example of doc
  drift the live approach fixes — the guide will show `__gop_` because it transpiles live.

## Risks / watch-items

- **Markdown-from-doc fidelity**: the playground parser currently renders straight to HTML.
  Lifting must expose the *raw markdown / structured fields* so the guide can emit markdown;
  don't route through HTML and back. Plan the `byexample.Feature` fields as raw text.
- **Locked output blocks become advisory for `goal ai`** (it shows live output) but remain
  load-bearing for the playground + its drift test — keep build-playground's `verify` intact
  so the locked blocks can't silently rot.
- **Diagnostic-code scan test** must tolerate formatting variance (`Code:     "x"` vs
  `Code: "x"`); use a tolerant regex like the grep used during research.
- **Determinism**: `goal ai` output must be byte-stable (sorted catalog, fixed section order)
  for the golden test (FR-11) to be reliable.

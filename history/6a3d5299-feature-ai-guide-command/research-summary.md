# Research Summary — `goal ai`

## Headline finding

The repo **already implements the "generated from real capabilities" pattern** the user
wants — it is just pointed at the website. `cmd/build-playground` parses `docs/by-example.md`
into per-feature records and re-runs the live `pipeline.Transpile` on every example,
asserting the locked Go output matches (`verify()`). So we are not inventing a mechanism;
we are adding a second consumer (a Markdown AI guide) of an existing, proven, drift-proof
source.

## What is genuinely "live" (derivable from the binary)

| Guide section | Live source | Mechanism |
|---|---|---|
| Per-feature syntax + lowering | `docs/by-example.md` + `pipeline.Transpile` | parse doc, transpile each example at output time |
| Intentional-error features | `pipeline.Transpile` error | show the real located message |
| Toolchain (commands/flags) | new command registry in `cmd/goal` | dispatch + guide read the same registry |
| Diagnostics catalog | `Code:` literals in `internal/check` + `internal/typecheck` | authored catalog + set-equality drift test |
| "What feedback looks like" | `check.AnalyzePackage` on embedded bad snippet | run checker live, render real diagnostics |
| Feature count/order | parsed doc categories | derived, never hardcoded |

## What stays authored (judgment prose, embedded, lightly guarded)

- Intro / why goal exists (can reuse `docs/overview.md`, already embedded-consumed).
- Locked conventions (qualified construction, `=>`, `_`, `?`, `__gop_` reserved prefix).
- Authoring do/don'ts for an AI agent.
- The iteration loop narrative (check → run → emit+test).

## Key drift the live approach fixes immediately

- The bootstrap prose says synthesized temporaries are `__goal_`; the live transpiler emits
  `__gop_`. The guide will show `__gop_` because it transpiles live.
- README / `docs/overview.md` claim "checker not started"; it is implemented and on by
  default. The guide renders the truth (and runs the checker live in its sample).

## Decisions confirmed with the user

1. Re-transpile examples at output time (literal live output).
2. Reuse `docs/by-example.md` + derive CLI/diagnostics from code; authored prose embedded.
3. `goal ai` subcommand, full Markdown dump to stdout; `goal --ai` alias; optional
   `goal ai <section>`.
4. `AI-KNOWLEDGE-BOOTSTRAP.md` becomes a generated golden artifact guarded by a test.

## Constraints

- Zero third-party dependencies; stdlib `testing` only (project rule overrides global Go
  standard suggesting testify/urfave/cli).
- Output must be byte-deterministic for the golden test.
- Lift the doc parser into a shared internal package so build-playground and `goal ai` use
  one parser (not two) — and expose raw markdown/structured fields, not HTML, so the guide
  can render Markdown directly.

## Open questions carried into planning

- OQ-1: ship `goal ai <section>` in v1 or defer (P3).
- OQ-2: authored sections in a new `docs/ai/` fragment dir vs. appended to by-example.md.
- OQ-3: feedback sample uses lexical checker stage only (single-file); confirm representative
  or embed a tiny module fixture for the depth stage.

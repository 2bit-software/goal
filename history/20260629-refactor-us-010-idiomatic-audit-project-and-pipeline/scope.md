# Scope — US-010 idiomatic audit: project and pipeline

## What is being audited and why

The two smallest selfhost packages, combined into one story:

- `selfhost/project/project.goal` — package discovery (File/Package, Discover,
  packageName, PackageClause, skipDir).
- `selfhost/pipeline/pipeline.goal` — pure output type defs (Output, GoFile,
  PackageOutput).
- `selfhost/pipeline/sourcemap.goal` — `//line` directive insertion
  (AddLineDirectives, declSites, declLines, declName, declSite).

Goal: apply goal idioms (Result/Option + `?`, enum/match) where behavior-
preserving, otherwise record refusals in DECISIONS.md, and prove the machine
check (`goal fix` reports no remaining auto-convertible sites) holds.

## Current state (the (T,error)/fallible surface)

`goal fix` report (report-only, before any change):
- project.goal:53 `Discover` — skipped [result-sig]: exported, callers fix
  cannot see (NOT auto-convertible).
- project.goal:100 `packageName` — skipped [result-sig]: non-propagating return
  (builds errors via fmt.Errorf; NOT auto-convertible).
- project.goal:76 `Discover` — suggestion [call-site] (advisory; no source diff).
- pipeline: no findings at all (no fallible surface).

So zero auto-convertible propagation sites already — AC-2 holds for both
packages before any edit.

## Constraints (what must NOT change)

- `Discover` and `PackageClause` are oracle-pinned: internal/project/
  project_test.go (TestDiscover*, TestPackageClause*) and the self-host port
  gate (internal/selfhost/port_test.go uses project.Discover heavily). Their
  public signatures are frozen.
- task fixpoint must stay byte-identical (goal-c-1 == goal-c-2).
- No cross-package edits.

## Audit conclusion (refusal, no source change)

Per the US-009 safety rule, an open-E (T,error)->Result conversion is only safe
on an exported fn with NO in-tree callers AND NO oracle test. Evaluating every
fallible function:

- `Discover` (exported (T,error)) — has in-tree callers (port gate + main) AND
  oracle tests pin it. REFUSE.
- `PackageClause` (exported) — returns plain `string`, swallows the parse error
  with `_`; not a (T,error) function. Also oracle-pinned. Nothing to convert.
- `packageName` (unexported (string,error)) — sole caller is Discover, which is
  not Result-returning and cannot become one (oracle-pinned), so `?` has no
  valid host; converting would force manual Result-unpacking at the caller —
  not a behavior-preserving idiom gain. `goal fix` also reports it as
  non-propagating. REFUSE.
- pipeline (Output/GoFile/PackageOutput, AddLineDirectives, declSites,
  declLines, declName) — no error-returning function exists; declSites swallows
  the parse error with `_`. No Result/Option/`?` surface. Nothing to convert.
- switch->match: `declName` type-switches over `ast.Decl` (a category interface
  that cannot be sealed per US-007 §9) with no in-file enum. No match candidate.
- enum/Option: no closed unordered variant set, no comma-ok value helper. N/A.

Outcome: documented refusal for both packages (mirrors US-005..US-008), no
.goal source change. DECISIONS.md gains a "US-010 (project + pipeline)" section.

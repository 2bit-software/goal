# Scaffold notes — US-010

## What was created

This audit is a documented refusal: no `.goal` source changes. The "new
implementation" is the analysis recorded in DECISIONS.md.

- DECISIONS.md — new section "self-host idiomatic audit — US-010 (project +
  pipeline)" classifying every fallible function and recording the refusals:
  - `Discover` — refused (exported, oracle-pinned by internal/project tests + the
    self-host port gate; no Result host for `?`).
  - `packageName` — refused (sole caller is non-Result `Discover`; non-propagating
    return per `goal fix`).
  - `PackageClause` — N/A (returns plain string, swallows parse error).
  - pipeline (pipeline.goal + sourcemap.goal) — N/A (no fallible surface).
  - switch->match — refused (no in-file enum; declName's type-switch is over the
    unsealable ast.Decl interface).
  - enum / Option — N/A.

## How it differs from old code

It does not differ — the `.goal` sources are unchanged by design. The machine
check (`goal fix` reports no auto-convertible sites) already held before this
story; the audit confirms there is no behavior-preserving conversion to make.

## How to verify independently

- `goal fix selfhost/project/*.goal` and `goal fix selfhost/pipeline/*.goal` —
  no source diff; only deliberate SKIPs + one advisory suggestion.
- `go test ./internal/selfhost -run 'TestPortedProjectPackage|TestPortedPipelinePackage'`
  — both ported packages transpile, compile, and pass behavioral suites.
- `task check`, `task build`, `task fixpoint` — all green; fixpoint byte-identical.

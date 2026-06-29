# US-008 Idiomatic audit: parser

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop runs linear history; no branch per loop-runner)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | done | 2026-06-29 |
| plan | done | 2026-06-29 |
| tasks | done | 2026-06-29 |
| implement | done | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

Idiomatic audit of `selfhost/parser` (step 3 of the self-host idiomatic plan).
Genuinely look for intra-package, behavior-preserving idiomatic conversions:
package-internal helpers returning `(T, error)` that can become `Result`/`Option`
+ `?`, and internal `switch`-over-in-file-`enum` that can become `match`. Convert
the real intra-package surface where it exists; record genuine refusals-with-reason
in DECISIONS.md. Never change the public `ParseFile` signature the US-003 verbatim
self-host oracle pins; parser tests must pass against the transpiled package; and
`task fixpoint` must stay byte-identical. Machine check: `goal fix` reports no
remaining auto-convertible propagation sites.

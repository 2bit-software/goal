# Completeness Audit — US-004

Scope: behavior-preserving rewire of `goal check`'s lexical-equivalent stage
from `internal/check` to `internal/sema`.

## Findings

- MINOR: FR-2 carves out the DECISIONS.md divergences (US-003). The corpus
  e2e test must therefore assert against the sema behavior for those cases, not
  the pre-rewire legacy text. Mitigated by reusing/choosing a corpus case whose
  finding is NOT one of the 4 documented divergences, or by asserting the sema
  outcome explicitly.
- MINOR: Severity reconciliation between the depth stage (`check.Severity`) and
  the sema stage (`sema.Severity`) is an implementation detail; spec correctly
  leaves it out. Both are int-based, Error=0/Warning=1, so conversion is total.

No CRITICAL or MAJOR findings. The change is mechanical and guarded by the
existing US-003 parity gate.

## Assumptions

- The corpus check cases referenced by the e2e test exist under the corpus
  manifest (KindCheck), as relied on by US-003's parity gate.
- `sema.AnalyzePackageInDir` is the intended driver (per US-002 notes).

# Verification — US-003 differential parity gate

## Test suite

- `task check` (go vet + `go test ./... -count=1`): PASS — all packages green,
  including `goal/internal/corpus` (which now hosts the parity gate).
- `task build`: PASS — `bin/goal` and `bin/goalc` build.
- Focused: `go test ./internal/corpus/ -run TestSemaLegacyParity -v`: PASS.

## Acceptance criteria

- [x] A test runs both the sema checker (`SemaCheck`) and the legacy
      `internal/check` (`check.Analyze`) over every `KindCheck` case under
      `testdata/check/**` and compares by (file, line, feature, code, severity).
      → `TestSemaLegacyParity` in `internal/corpus/parity_test.go`.
- [x] Passes only when findings are identical except for divergences explicitly
      recorded in `DECISIONS.md`. → `knownDivergences` allowlist, each entry backed
      by the new DECISIONS.md "US-003" section; undocumented divergences and stale
      allowlist entries both fail the gate.
- [x] AST-fires-Error / legacy-deferred divergences recorded as documented
      improvements with `// want` markers reflecting sema behavior. → three
      12-derive-convert cases documented; their existing markers already contain
      the sema Error substrings (verified), so they reflect sema behavior.
- [x] `task check` and `task build` green.

## Notes

The gate is self-policing: it fails loudly on an empty check corpus, on any new
undocumented divergence, and on any documented divergence that stops reproducing.

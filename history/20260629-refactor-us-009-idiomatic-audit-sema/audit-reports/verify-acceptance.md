# Verify: Acceptance — US-009 sema

All acceptance criteria verified GREEN.

| Criterion | Result |
|-----------|--------|
| Fallible functions use Result/Option + `?` where it fits | `Analyze` converted to `Result[[]Diagnostic, error]` with `parser.ParseFile(src)?`; this is the complete set of fitting in-package sites. PASS |
| Diagnostic/mode kinds as enums + match where they fit, else recorded | Mode/Severity (ordered iota ints, cross-package ==/conversions) recorded as refusals in DECISIONS.md; Diagnostic.Code/Feature are strings; no in-file enum so no switch->match. PASS |
| `goal fix` reports no remaining auto-convertible propagation sites | Per-file `goal fix selfhost/sema/*.goal` produces NO source diff; remaining output is only documented skips + advisory suggestions (non-auto-convertible). `Analyze` no longer appears. PASS |
| sema tests pass against transpiled package | `go test ./internal/selfhost -run TestPortedSemaPackage` → ok. PASS |
| task check / build / fixpoint green, fixpoint byte-identical | task check ok (all pkgs incl. internal/sema + port gate); task build ok; task fixpoint → FIXPOINT OK. PASS |

Emitted Go confirms behavior preservation: `func Analyze(src string) (ok
[]Diagnostic, err error) { file, err := parser.ParseFile(src); if err != nil {
return ok, err }; ...; return Check(file, info), nil }` — identical two-value
signature and propagation.

No CRITICAL, MAJOR, or MINOR findings.

## Assumptions

- An exported function with no in-tree callers and no oracle test may have its
  goal-source idiom changed because the emitted Go signature is preserved (open-E
  Result lowers to (T,error)). Validated by FIXPOINT OK + green port gate.

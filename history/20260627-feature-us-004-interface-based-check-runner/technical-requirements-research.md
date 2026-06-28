# Technical Requirements — US-004

- Mirror the existing US-003 transpile-runner pattern: define a Checker
  interface + CheckerFunc adapter so the free func check.Analyze
  (func(string)([]check.Diagnostic,error)) satisfies it.
- Reuse the // want marker semantics already proven in
  internal/check/check_test.go (regexp `//\s*want\s+"([^"]*)"`, line-keyed,
  OffsetToPosition for diagnostic line, unclaimed Error fails).
- No import cycle: internal/check imports only internal/analyze; corpus may
  import check.
- Test loads ../../corpus/manifest.json (manifestPath const already exists),
  iterates KindCheck cases, runs via CheckerFunc(check.Analyze), fails loudly if
  zero check cases.
- Zero-dependency: stdlib testing only, no testify.

# Verify — Quality (US-002)

- Error handling: parse error returned (TestAnalyzePackageInDirParseErrorReturned);
  foreign-resolution errors non-fatal and surfaced via the `...With` slice
  (control branch asserts len(ferrs)==1 and no crash).
- Edge: empty/nil srcs returns an empty result + nil error (per-file loop falls
  through) — not separately tested but trivially correct.
- The foreign-dependent test asserts the REAL dependency by contrasting
  Warning-vs-Error across two resolver behaviors, so it cannot pass vacuously.
- No code contradicts the spec; the driver mirrors check.AnalyzePackageInDirWith.
- Zero-dependency / stdlib testing honored (no testify).

No findings. Recommend pass.

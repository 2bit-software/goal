# Verification — Quality

- Error handling: deferral path (unknown keyed type → Warning) and the
  skip paths (non-Ident type, unknown enum, non-variant tag, data-less variant)
  are all exercised by the golden corpus; no silent false "complete".
- Edge cases covered by goldens: spread opt-out (both `...defaults` and
  `...derive`), struct tags, nested literals (complete.goal Addr), data-less
  variant construction (Shape.Dot as SelectorExpr), match-arm binding.
- No contradiction with spec: messages mirror internal/check/fields.go so marker
  parity holds; severities match (Error rejects, Warning defers).
- Tests assert real behavior: each `// want` marker is matched against a
  diagnostic on the same line by corpus.RunCheck, which also fails on any
  unclaimed Error; no-marker cases fail on any Error at all.
- Scope discipline: only sema (fields.go + one-line Check wiring) and a new
  corpus test touched; no change to the lexical check, the manifest, or any
  golden — the AST checker plugs in through the existing SemaCheck adapter.

No quality findings. The check is correct by construction where the lexical
checker must reconstruct shape heuristically.

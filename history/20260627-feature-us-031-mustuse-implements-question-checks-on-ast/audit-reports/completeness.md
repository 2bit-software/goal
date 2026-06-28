# Audit — Completeness

## Verdict: ready to implement

The spec's behavior is fully pinned by the existing `testdata/check` `// want`
markers (03-result, 06-error-e, 07-implements) and by the lexical checker
(`internal/check/{mustuse,implements,question,closed}.go`) as a line-by-line oracle.

## Findings

- **MINOR** — FR-3 (open-E `?`) has no Error-producing fixture in the corpus; only a
  clean pass (`consumed_clean.goal` `questionUse`). The Error branches are implemented
  for parity but are exercised only by intent, not by a corpus fixture. Acceptable:
  US-031's gate is the corpus cases that exist; the Error branches mirror the lexical
  checker verbatim. Covered additionally by a sema unit test.
- **MINOR** — The `_ :=` discard (`defer_underscore_discard.goal`) has no `// want`
  marker; both "emit advisory Warning" and "emit nothing" satisfy the test (unclaimed
  Warnings are allowed). Spec chooses the Warning for parity. No ambiguity for the gate.
- **MINOR** — Closed-E `Result.Err` passthrough (`defer_err_value.goal`, bare-ident
  arg) is specified as deferred (Warning). The lexical checker has a richer
  "passthrough suppression"; the AST check uses the simpler "non-`E.Variant` arg ⇒
  defer", which produces no Error and satisfies the fixture (no marker). Equivalent for
  the gate.

No CRITICAL or MAJOR findings. No contradictions. No unresolved open questions.

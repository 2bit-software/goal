# AI-Consumer Readiness Audit — US-007

## Findings

- Implementable without guessing: the port pattern is fully specified by the two
  prior ports (US-005 token, US-006 lexer) and the existing
  internal/selfhost.{BuildTranspiled,BuildAndTest} harness.
- Acceptance criteria map 1:1 to test assertions (Discover -> package name,
  BuildTranspiled -> compile, BuildAndTest -> behavioral).

No CRITICAL or MAJOR findings.

## Assumptions

- The copied internal/ast sources need zero edits beyond dropping dump.go,
  because the only reserved-word grep hit is the string literal `len("assert")`.
- The package clause stays `package ast`; project.Discover resolves the goal
  source under selfhost/ast as one package named "ast".

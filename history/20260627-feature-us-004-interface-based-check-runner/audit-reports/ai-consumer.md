# AI-Consumer Readiness Audit — US-004

An AI agent can implement this without guessing: the Checker interface signature,
marker regexp, line-keying, and unclaimed-Error rule are all fixed by the existing
internal/check/check_test.go harness and the US-003 runner pattern.

No CRITICAL or MAJOR findings.

## Assumptions
- Checker.Check returns []check.Diagnostic (the project's existing diagnostic
  type), so the runner can read Severity/Message and compute the line via
  check.OffsetToPosition — no new diagnostic abstraction is introduced.

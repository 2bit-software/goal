# AI-Consumer Readiness Audit — US-017 (iteration 2)

Re-audit after the spec revision.

## Resolution of iteration-1 findings

- **C-1 (CRITICAL) — `from` fixture verifies vacuously.** RESOLVED. The spec's
  Test harness section directs the implementer to a closed-E program whose callee
  genuinely returns `Result.Err(ParseError.Empty)` so `toApp` actually fires;
  same-E coverage uses `qclosed_prop_same`'s erring `parse`.
- **M-1 — test oracle/data format unspecified.** RESOLVED. The Test harness
  section names the convention: inline `const program` strings + `newInterp` /
  `evalFn` + `Value`-shape assertions; `.go.expected` is NOT an oracle.
- **M-2 — conversion trigger runtime vs static.** RESOLVED. Error Handling states
  the trigger is resolved statically off the direct-call operand's callee E vs
  caller E, with a located refusal when it cannot be resolved.
- **M-3 — cross-container `?` undefined.** RESOLVED. The interpreter trusts sema
  well-typedness; a non-variant or mismatched operand is a located refusal.

## Verdict

An AI agent can implement this without clarifying questions. No CRITICAL/MAJOR
remain.

## Assumptions

- The current-function signature is threaded via an interpreter-held stack pushed
  in `callFunc` (callee sig) and `callMethod` (none sig), popped via defer.
- `?` is handled in `evalExpr` as a single `case *ast.UnwrapExpr` seam, reached
  by all statement positions through existing callers.

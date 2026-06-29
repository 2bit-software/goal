# Tasks — SEAM-003

Atomic seam: the tree is red until all source tasks (T1-T4) land together
(§9 / undefined-symbol). Commit only after T5 is green.

## T1 — Enum definitions (foundation)
- selfhost/sema/sema.goal: `Mode` iota -> `enum Mode { ModeNone ModeResult ModeResultClosed ModeOption }`.
- selfhost/sema/check.goal: `Severity` iota -> `enum Severity { Error Warning }`; convert `String()` to `match s { Severity.Warning => "warning"  _ => "error" }`.
- Depends on: nothing.

## T2 — Same-package sema consumers
- selfhost/sema/foreign.goal: add `Mode: Mode.ModeNone` to the foreign `FuncSig{}` (nil-fault fix).
- selfhost/sema/resolve.goal: construction `Mode.ModeX`; switch over `sig.Mode` -> two value-matches (Arity, EndsInError).
- selfhost/sema/question.goal: `caller.Mode != ModeResult` -> bool match (x2); appendQuestionResolved switch-true Mode cases -> precomputed bool matches; closedQuestionDiags guarded bool match.
- selfhost/sema/mustuse.goal: `ok && (sig.Mode == ...)` -> `ok &&` + guarded bool match.
- selfhost/sema/{implements,convert,fields,assert,check}.goal: requalify `Severity: Error/Warning` -> `Severity: Severity.Error/Warning`.
- Depends on: T1.

## T3 — Cross-package backend consumers
- selfhost/backend/lower.goal: needsResultPrelude loop -> bool match.
- selfhost/backend/emit.goal: 426/1772 guarded bool match; construction `sema.Mode.ModeResult` (1901/1904); calleeMode guard + 2129 match.
- Depends on: T1.

## T4 — Cross-package typecheck consumers
- selfhost/typecheck/mustuse.goal: guarded bool match for `sig.Mode != sema.ModeResult`; `Severity: sema.Severity.Error/Warning`.
- selfhost/typecheck/implements.goal, nozero.goal: `Severity: sema.Severity.Error`.
- Depends on: T1.

## T5 — Verify + docs
- Run: `task check`, `task build`, `task fixpoint`. Fix any miss (grep for residual `== ModeX`, `switch .*Mode`, `Severity: Error`).
- DECISIONS.md: record conversion, supersede US-011.
- Depends on: T1-T4.

# Implementation Tasks — US-008 idiomatic audit: parser

## Task 1: Confirm gates green on unchanged parser source
**Status**: completed
**Files**: (none — verification only)
**Depends on**: (none)
**Spec coverage**: AC machine check, AC tests-against-transpiled, AC build/fixpoint
**Verify**:
- `goal fix selfhost/parser/*.goal` → no content diff; only the result-sig SKIP on
  exported `ParseFile` (no auto-convertible sites)
- `task check` (includes `internal/selfhost` port gate + `internal/parser` tests)
- `task build`
- `task fixpoint` → FIXPOINT OK, byte-identical

### Instructions
Build `bin/goal`, run `goal fix` on each parser file and diff against source, then
run the three prd `verifyCommands`. All must be green before recording the decision.

## Task 2: Record the US-008 (parser) audit decision in DECISIONS.md
**Status**: completed
**Files**: `DECISIONS.md`
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3 (records the refusals-with-reason)
**Verify**: section "self-host idiomatic audit — US-008 (parser)" exists; `task
check`/`task build`/`task fixpoint` remain green (doc-only change cannot affect them).

### Instructions
Append a new section mirroring the US-005/006/007 format. Record three classified
candidates:
1. Result/`?`: refused — parser is an error-accumulator (`parser.errs` + `errorf`);
   no intra-package `(T,error)` propagation surface; the lone `(T,error)` is the
   exported, oracle-pinned `ParseFile` (goal fix result-sig SKIPs it).
2. switch→match: refused — no in-file `enum`; switches are over `token.Kind` (int),
   tokens, `ast` interface type-switches (can't seal per US-007 §9), and boolean
   `switch {}`. No closed-enum scrutinee.
3. Option for predicates: not applicable — bool funcs are pure predicates.
Note the machine check result and that the outcome is no `.goal` source change.

## Task 3: Flip prd.json + append progress.txt (loop bookkeeping)
**Status**: pending
**Files**: `prd.json`, `progress.txt`
**Depends on**: Task 2 + green verify
**Spec coverage**: loop-runner finalize steps
**Verify**: `prd.json` US-008 `passes:true`; progress.txt has the new US-008 entry.

### Instructions
After `/mc.complete` and the commit land green, set US-008 `passes:true` and append
the progress.txt iteration entry (done by the loop runner, not inside the workflow
implement step).

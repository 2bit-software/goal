# Tasks Audit

## Coverage
- FR-1: Tasks 1, 2, 3. FR-2: Tasks 1, 2, 3. FR-3: Tasks 1, 2. FR-4: Task 3.
  FR-5: Task 2. All FRs covered.
- All plan files covered: emit.go (T1), emit.goal (T2), sealed_methods_test.go (T3).
  lower.go/lower.goal intentionally unchanged.
- No scope creep.

## Ordering
- DAG: T1 -> {T2, T3}. Valid; no cycles. Codebase compiles after each task
  (T1 alone is self-consistent; T2 mirror keeps selfhost valid; T3 adds a test).

## Executability
- Each task has concrete instructions referencing exact functions/line ranges and a
  runnable verify command. Each touches 1 file. T2's verify (`task fixpoint`)
  depends on T1+T2 both landing — acceptable since T2 depends on T1.

## Sizing
- All three tasks are right-sized for a single turn; none trivial or oversized.

No CRITICAL/MAJOR findings.

## Assumptions
- Behavioral proof of callability is done via a temp-module `go test` rather than an
  in-process check, matching existing backend test patterns.
- Empty-body sealed interfaces are deliberately excluded from the new multi-line
  path to preserve byte-identity / fixpoint.

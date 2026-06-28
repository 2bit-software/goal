# Task Breakdown — US-004 interface-based check runner

## Task 1: Add Checker interface + RunCheck runner
- File: internal/corpus/check_runner.go (new)
- Define Checker interface, CheckerFunc adapter, RunCheck(root, Case, Checker).
- RunCheck: read case Input relative to root; run ck.Check; parse `// want`
  markers (regexp `//\s*want\s+"([^"]*)"`, per-line); each marker satisfied by a
  diagnostic on the same line (check.OffsetToPosition) whose Message contains the
  substr; any Error-severity diagnostic on a marker-less line => fail. Warnings
  may go unclaimed. Return case-identified errors.
- Covers FR-1, FR-2, FR-3.
- Verify: go build ./internal/corpus.

## Task 2: Whole-corpus check test
- File: internal/corpus/check_runner_test.go (new)
- TestCheckRunner: Load(manifestPath); iterate cases; skip non-KindCheck; per
  case t.Run(c.ID) calling RunCheck(repoRoot, c, CheckerFunc(check.Analyze));
  t.Fatalf if zero check cases ran.
- Covers FR-4.
- Verify: go test ./internal/corpus -run TestCheckRunner -count=1.

## Coverage
- FR-1/2/3 -> Task 1 (check_runner.go)
- FR-4 -> Task 2 (check_runner_test.go)
- Verify gates -> both tasks + full go build/vet/test ./....

# Verification Report — US-004

## verifyCommands (prd.json)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages ok)

## Acceptance criteria
- Checker interface + RunCheck matching `// want "substr"` markers per-line,
  failing on unclaimed Error-severity diagnostics — DONE (check_runner.go).
- A test runs every check case in the manifest against check.Analyze and all
  pass — DONE: TestCheckRunner runs 50/50 check cases, all PASS.

## Commit
5802780 feat(corpus): add interface-based check runner (US-004)

# Verification — US-003

## Gates (verifyCommands)

- `task check` — PASS (go vet + full `go test ./...`; includes goal/internal/corpus
  transpile/behavioral/check tiers and the self-host port gates goal/internal/selfhost).
- `task build` — PASS (bin/goal, bin/goalc).
- `task fixpoint` — PASS (FIXPOINT OK; `diff -r _bootstrap/fa _bootstrap/fb` empty).

## Acceptance Criteria

- AC #1 — `selfhost/main.goal` imports `goal/selfhost/{backend,pipeline,project}`,
  no `internal/*`. VERIFIED (grep).
- AC #2 — no `selfhost/*.goal` imports `goal/internal/*`. VERIFIED
  (`grep -rn 'goal/internal/' selfhost/` returns only a doc comment phrase, no imports).
- AC #3 — `task fixpoint` exits 0 with goal-c-1/goal-c-2 built from the
  self-contained tree (nested `module goal`, `go build -C`). VERIFIED.
- AC #4 — `task check` green (corpus tiers + port gates). VERIFIED.
- AC #5 — `task build` green. VERIFIED.

## Notes

- The fixpoint is now a genuine differential proof: goal-c-1 is built from the
  ported selfhost packages (not goal/internal/*), and goal-c-1's emit of the
  compiler equals goal-c-2's byte-for-byte.
- The corpus transpile + behavioral tiers run under `task check` via the trusted
  compiler; the genuine fixpoint establishes that goal-c reproduces that compiler
  byte-identically, satisfying the corpus criterion without new infrastructure.

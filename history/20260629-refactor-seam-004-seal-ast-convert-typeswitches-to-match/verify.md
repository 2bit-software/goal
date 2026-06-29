# Verify — SEAM-004

## Gates (all green)
- `task check` — exit 0; full `go test ./...` + `go vet` pass (incl. internal/corpus
  behavioral/interp tiers and internal/selfhost port gates building each sealed,
  match-converted selfhost/<pkg> and running the ported internal/<pkg> tests).
- `task build` — clean (bin/goal, bin/goalc).
- `task fixpoint` — FIXPOINT OK (goal-c-1 and goal-c-2 byte-identical over ./selfhost
  on the new sealed-AST + match source).
- Full `go test ./...` run twice: no failures (confirms the fixed sealedInterfaceOf
  nondeterminism is gone; previously ~50%/run flake).
- Nested-sealed exhaustiveness tests stress-run (-count=40): stable.

## Acceptance criteria
1. Node/Expr/Stmt/Decl/Spec are sealed interfaces (§8.1) — done.
2. AST-family type-switches over a single sealed interface converted to `match` with
   exhaustiveness across sema (check, resolve, question, fields, mustuse, assert),
   backend (lower, emit), parser (parser, goal_construct) — done.
3. go/ast-mirror oracle resolved: mirror retained unchanged (tests reference no
   marker methods, so they pass against both open internal/ast and sealed selfhost
   transpile) — recorded in DECISIONS.md.
4. task check (no golden regen needed) + build + fixpoint green; corpus tiers
   unchanged — done.
5. Walk's switch exclusion justified (grouped multi-type + no-op clauses; AC-permitted)
   — documented in walk.goal and DECISIONS.md. emit.armBody (sibling-category) and
   foreign ×3 (go/ast stdlib) also kept plain with documented reasons.

## Prerequisite fix
sealedInterfaceOf now resolves to the most specific sealed interface containing all
covered arm types (deterministic), fixing a latent CAP-3d nondeterminism that the
real nested AST exposed. Mirrored in internal/ + selfhost/.

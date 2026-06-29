# Audit: Completeness

## CRITICAL
None.

## MAJOR
None. All five pipeline steps (scope guard, rename, autofix, manual idioms,
verify+report) have explicit functional requirements with reasons enumerated for
each non-fit (numeric identity, accumulator, multi-value, comma-ok, cross-package
consumer not in scope).

## MINOR
- MINOR: "warn/refuse" on a multi-package module leaves the choice between warn
  and refuse to the implementer. Resolution: skill SHALL refuse by default and
  state the one-file/one-package rule, since proceeding on a multi-package module
  is exactly the unsupported case. Documented in the skill.
- MINOR: build-tag handling is named but the SEAM stories found no build tags in
  the selfhost tree. Resolution: skill documents the rule (preserve `//go:build`
  / `// +build` lines verbatim; they are comments and pass through) and flags any
  encountered for manual review.

## Assumptions
- Skill location is `.claude/skills/go-to-goal-upgrade/` (standard Claude Code
  project skill dir; repo has no pre-existing project skill).
- The skill is procedural guidance + helper scripts for an agent/human to run,
  not a compiled binary — matching how the global skills (bootstrap-project) are
  structured.
- Dogfood runs on a COPY in a scratch dir; the example is small and self-contained.

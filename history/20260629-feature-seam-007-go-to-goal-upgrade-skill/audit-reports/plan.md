# Audit: Implementation Plan

## CRITICAL
None.

## MAJOR
None. Every spec FR maps to a plan file: FR-1 scope-guard.sh; FR-2 rename.sh;
FR-3/FR-4 SKILL.md + idiom-catalogue.md; FR-5 SKILL.md report template + verify;
FR-6 SKILL.md carry-forward section. Dogfood = examples/before + after +
walkthrough. Dependency order is a valid topological sort.

## MINOR
- MINOR: plan should pin the dogfood example concretely. Resolution: use a small
  purpose-made Go package exercising an iota const block, a type-switch over a
  closed set, a method on the iota type, and a fallible (T,error) function — so
  all four manual idioms are demonstrated, plus a `goal fix` candidate.

## Assumptions
- Skill is procedural (markdown + shell helpers), not compiled code; matches the
  global skill house style.
- Paths verified: `.claude/` exists and is empty of skills; no conflict.

# Tasks Audit — US-046

## Coverage
- FR-1/2/3 -> Task 1. FR-4 -> Task 2. AC "full LSP suite passes" / "scanDecls
  removed" -> Task 3 (and Task 1). Every plan file (symbols.go, diagnostics.go)
  appears. No scope creep.

## Ordering
- Tasks 1 and 2 are independent and each leaves the package compiling (signatures
  preserved). Task 3 depends on both. Valid DAG, no forward refs.

## Executability
- Each task names exact files, exact node types/fields, and a concrete verify
  command. Task 1 references ast node types and rangeOf; Task 2 references
  lexer.Tokens and token.Token fields. Unambiguous.
- Each task touches <= 1 file (well under 5).

## Sizing
- Task 1 is the substantive rewrite (single file, single function + helper).
  Tasks 2 and 3 are small but non-trivial (import swap + gates/grep guards) and
  each carries a real verification. Acceptable; could be merged but kept distinct
  for clear verification per AC.

## Findings
- None CRITICAL/MAJOR/MINOR that block.

## Assumptions
- Splitting verify (Task 3) from the edits keeps each task's verification crisp;
  all three land in one commit (one story).

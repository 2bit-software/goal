# Implementation Tasks

## Task 1: Implement CheckFields over the AST and wire it into Check
**Status**: completed
**Files**: `internal/sema/fields.go` (new), `internal/sema/check.go` (modify)
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4, FR-5, FR-6
**Verify**: `go build ./... && go vet ./...`

### Instructions
- Create `internal/sema/fields.go` (package sema) with
  `CheckFields(file *ast.File, info *Info) []Diagnostic`.
- Walk the file using the existing `visitorFunc` adapter (in check.go). Collect
  and check:
  - `*ast.CompositeLit` with `Type` an `*ast.Ident`:
    - If any elt is `*ast.SpreadElement` → complete; skip (FR-3).
    - present = set of `*ast.KeyValueExpr` keys whose Key is an `*ast.Ident`.
    - If `info.Structs[name]` known → compute missing declared fields in order;
      Error `missing-field` if any (FR-1).
    - Else if keyed (≥1 KeyValueExpr) → Warning `unresolved-literal-type` (FR-4).
    - Else skip.
  - `*ast.VariantLit`: resolve enum via `exprName(vl.Enum)`; `info.Enums[name]`;
    nil → skip. Find the variant by `vl.Variant.Name`; absent → skip; data-less →
    clean. present = `*ast.LabeledArg` labels; missing declared fields in order →
    Error `missing-field` (FR-2). (Match-arm bindings are VariantPattern, not
    VariantLit, so never reached — FR-5.)
- Messages mirror internal/check/fields.go verbatim (FR-6). Feature string
  `08-no-zero-value`. Reuse `plural`/`pronoun`/`exprName` from check.go; add a
  `quoteJoin(names []string) string` (backtick-join).
- In `internal/sema/check.go`, append `CheckFields(file, info)...` to `Check`.

## Task 2: Corpus runner test for 08-no-zero-value
**Status**: completed
**Files**: `internal/corpus/sema_fields_test.go` (new)
**Depends on**: Task 1
**Spec coverage**: all acceptance criteria (the 9 golden cases)
**Verify**: `go test ./internal/corpus/ -run TestSemaFields -count=1`

### Instructions
- Mirror `internal/corpus/sema_checker_test.go` (`TestSemaExhaustiveRunner`):
  iterate manifest `KindCheck` cases whose `Input` has prefix
  `testdata/check/08-no-zero-value/`, run each through
  `RunCheck(repoRoot, c, CheckerFunc(SemaCheck))`, `t.Fatalf` if zero cases ran.
- Then run the full prd verify gates: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`.

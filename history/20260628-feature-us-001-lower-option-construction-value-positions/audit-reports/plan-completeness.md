# Plan Audit: Coverage

## Findings

No CRITICAL findings. No MAJOR findings.

Every acceptance criterion traces to a plan element:
- FR-1/FR-2 (all value positions) -> `tryOptionValue` interception at the top of
  `emitter.expr`, which is the single seam all value positions route through.
- FR-3 (nil / &x / boxed) -> `optionConstruction` kinds + `optionPrelude` helper.
- FR-4 (no regression) -> existing `optionValueExpr`/return path untouched.
- Test criterion -> `TestASTEngineLowersOptionInValuePositions` in the testing
  strategy.

No scope creep: package-mode `goal_options.go` injection is required for
correctness of the existing whole-corpus verify gate, not new scope.

## Assumptions
- Single-pass emitter cannot hoist a temp in pure-expression position, so a generic
  helper is the boxed-temporary realization.

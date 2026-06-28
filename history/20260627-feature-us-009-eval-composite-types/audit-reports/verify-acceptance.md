# Verify — Acceptance Coverage — US-009

Full suite green: `go build ./...`, `go vet ./...`, `go test ./... -count=1`
all pass. `internal/interp` dependency check confirms no go/types / backend /
typecheck import (US-022 invariant preserved).

## AC → evidence

| Acceptance criterion | Test evidence |
|---|---|
| Struct literal builds a struct; fields read back via field access | `TestCompositeProgram/structFields` (=7), `structFieldAssign` (=17) |
| Slice literal builds a slice; elements read by index | `TestCompositeProgram/sliceIndex` (=60), `sliceElementAssign` (=31) |
| Map literal builds a map; key assignment updates it | `TestCompositeProgram/mapIndexAndAssign` (=105) |
| Ranging a slice yields ascending indices + elements | `rangeSliceValues` (=10), `rangeSliceIndices` (=20), `rangeBreakContinue` (=8) |
| Ranging a map visits each key + value | `rangeMapValues` (=6), `rangeMapKeys` (="abc", sorted) |
| Combined: builds+reads structs/slices/maps, ranges slice and map | `TestAcceptanceCombined` (=26) |
| Out-of-range index / unsupported target/key → descriptive error | `TestCompositeErrors` (outOfRange, nonStringMapKey, fieldOnNonStruct, indexNonCollection) |

Every acceptance criterion maps to an asserting test. No uncovered criteria.

## Assumptions
- Maps string-keyed; keyed struct literals; Go reference semantics for in-place
  mutation; map-range order sorted for determinism. All recorded in the spec.

# Verification — US-002 interp value model

## Acceptance Criteria (spec is source of truth)

- [x] AC-1: internal/interp defines a Value type covering int, float, string,
      bool, nil, struct, slice, map, and function, plus a universal tagged-union
      Variant{TypeID, Tag, Fields} used uniformly for enum/Result/Option.
      → value.go: Kind enum (KindNil/Int/Float/String/Bool/Struct/Slice/Map/Func/
      Variant), Value struct, Variant{TypeID,Tag,Fields}, VariantVal used for all
      three sum types (TestVariantUniformForSumTypes).
- [x] AC-2: A unit test constructs a Variant and each primitive/composite Value,
      reads a Variant field back by name, and asserts Value equality and String()
      rendering.
      → value_test.go: TestConstructEachKind (all 10 kinds), TestFieldReadBack
      (Field("value") round-trip + absent/non-variant negatives), TestEqual +
      TestEqualFuncByIdentity, TestString.

## Verify commands (prd.json verifyCommands)

- `go build ./...` → BUILD OK
- `go vet ./...` → VET OK
- `go test ./... -count=1` → all packages ok (goal/internal/interp ok)

## Findings

None. No CRITICAL/MAJOR/MINOR. The feature works exactly as specified and the
full project test suite is green.

## Assumptions (carried from plan/audit, validated)

- TypeID/Tag/field keys are strings; v1 maps are string-keyed.
- Function values compare by identity (name-only carrier; call wiring is US-004+).
- A single int64/float64 covers all int/float widths in v1.

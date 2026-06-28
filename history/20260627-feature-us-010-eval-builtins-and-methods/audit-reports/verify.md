# Verify — US-010 Eval builtins and methods

## verifyCommands (from prd.json) — all green

- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (all packages pass; internal/interp green)

## Acceptance criteria

1. Builtins `len`, `append`, `make`, `panic` implemented in
   internal/interp/eval.go (evalBuiltin + builtinLen/Append/Make/Panic);
   value- and pointer-receiver methods dispatched via the method registry
   (registerMethods) + tryMethodCall/callMethod in interp.go. ✓
2. internal/interp/builtins_test.go asserts:
   - `len` over slice/string/map and `make([]int,3)` length (TestBuiltinLen),
   - `append` element read-back (TestBuiltinAppendValue),
   - `make(map[string]int)` write + read-back (TestBuiltinMakeMapReadBack),
   - a recovered panic carrying its value (TestBuiltinPanicRecovered),
   - pointer-receiver mutation visible to caller (TestPointerReceiverMethodMutates),
   - value-receiver read + non-leak (TestValueReceiverMethodReads /
     TestValueReceiverMethodDoesNotLeak),
   - a descriptive refusal for len of a non-container (TestBuiltinLenUndefinedOperand). ✓

## Dependency discipline (US-022 forward-guard)

`go list -deps goal/internal/interp` shows no go/types, internal/backend, or
internal/typecheck dependency. ✓

# Implementation Verification — US-016

## Gates (prd.json verifyCommands)

- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages green, incl. internal/interp)

## Dependency envelope (US-022 guard)

`go list -deps ./internal/interp` contains no go/types, internal/backend, or
internal/typecheck — DEPS_CLEAN.

## Acceptance Criteria

| Criterion | Evidence |
|-----------|----------|
| Option.Some constructs tagged "Some"/TypeID "Option" with one payload | TestOptionSomeConstruction |
| Option.None constructs tagged "None" with no payload | TestOptionNoneConstruction |
| match Some runs Some arm | TestOptionMatchArms (present) |
| match None runs None arm | TestOptionMatchArms (absent) |
| Some arm binds the UNWRAPPED inner value | TestOptionSomeArmBindsUnwrappedValue |
| Unit test over a 04-option shape | internal/interp/option_test.go (all of the above) |
| (error stance) unknown ctor / wrong arity refused | TestOptionUnknownCtorIsRefused, TestOptionSomeArityIsRefused |

All acceptance criteria satisfied. No `*T` optimization — Option uses the same
universal Variant encoding as enums/Result.

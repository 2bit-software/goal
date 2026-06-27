# Implementation Tasks — US-003 Environment and scopes

## Task 1: Implement Env scope chain
**Status**: completed
**Files**: internal/interp/env.go (new)
**Depends on**: none (Value exists)

Create `env.go`:
- `NotFoundError{Name string}` with `Error() string` ("undefined: " + Name).
- `Env{vars map[string]Value; parent *Env}`.
- `NewEnv() *Env` — root scope.
- `(*Env) NewChild() *Env` — child with parent = receiver.
- `(*Env) Define(name string, v Value)` — bind in this scope.
- `(*Env) Lookup(name string) (Value, error)` — walk this scope then parents;
  not-found returns zero Value + *NotFoundError.

**Acceptance**: `go build ./...` and `go vet ./...` clean.

## Task 2: Unit tests
**Status**: completed
**Files**: internal/interp/env_test.go (new)
**Depends on**: Task 1

Tests (stdlib testing, no testify):
- TestDefineAndLookupSameScope
- TestParentFallThrough
- TestShadowing (non-destructive: parent still resolves outer)
- TestLookupUndefinedReturnsNotFound (errors.As to *NotFoundError, Name matches)
- TestDefineOverwriteSameScope

**Acceptance**: `go test ./internal/interp/ -count=1` passes; full
`go test ./... -count=1` green.

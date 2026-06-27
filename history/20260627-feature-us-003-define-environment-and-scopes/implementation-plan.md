# Implementation Plan — US-003 Environment and scopes

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/env.go` | The `Env` lexical scope chain: NewEnv root constructor, Define, Lookup, NewChild, plus the not-found error. |
| `internal/interp/env_test.go` | Stdlib `testing` unit tests for define/lookup, parent fall-through, shadowing, and not-found error. |

### Modified Files
None. This is purely additive to internal/interp.

## Package Structure

```
internal/interp/
  value.go        (existing, US-002)
  value_test.go   (existing, US-002)
  env.go          (new)
  env_test.go     (new)
```

## Dependency Graph

1. `internal/interp/value.go` — already exists (provides `Value`).
2. `internal/interp/env.go` — depends on 1 (stores `Value`).
3. `internal/interp/env_test.go` — depends on 2.

## Design

`Env` is a parent-linked scope:

```go
type Env struct {
    vars   map[string]Value
    parent *Env
}
```

- `NewEnv() *Env` — root scope (parent nil), empty vars.
- `(*Env) NewChild() *Env` — child scope whose parent is the receiver.
- `(*Env) Define(name string, v Value)` — bind in THIS scope (overwrites a
  same-scope binding).
- `(*Env) Lookup(name string) (Value, error)` — walk this scope then parents;
  first hit wins; on exhaustion return zero Value + a not-found error naming the
  symbol.

Not-found error: a typed `*NotFoundError{Name string}` (Error() = "undefined: "
+ Name) so callers can detect it via errors.As and the missing name is
reported. Keep it stdlib-only, no testify.

## Test Plan (maps to acceptance criteria)

- TestDefineAndLookupSameScope — Define then Lookup returns the value.
- TestParentFallThrough — outer Define, Lookup from child finds it.
- TestShadowing — child redefines a name; child Lookup returns inner, parent
  Lookup still returns outer (non-destructive).
- TestLookupUndefinedReturnsNotFound — Lookup of an unbound name returns a
  not-found error naming the symbol (errors.As to *NotFoundError, Name matches).
- TestDefineOverwriteSameScope — re-Define in same scope replaces the value.

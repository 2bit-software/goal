# Feature Specification: Statement-context error passthrough in `goal fix`

**Feature Branch**: `n/a (no branching — applied on main)`
**Created**: 2026-06-25
**Status**: Draft
**Input**: User description: extend `goal fix` to convert the statement-context error
guard `if err := doThing(); err != nil { return Result.Err(err) }` into `doThing()?`
when the call's only output is the error, inside a `Result[T, error]` function.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Collapse a pure error passthrough written as an if-init guard (Priority: P1)

A goal author has a function returning `Result[T, error]` whose body calls a side-effecting
helper that returns only `error` (e.g. `os.MkdirAll`, `json.Unmarshal`, `protojson.Unmarshal`,
`toml.Unmarshal`, `os.WriteFile`). They wrote it in idiomatic Go statement-context style:

```goal
if err := os.MkdirAll(dir, 0o755); err != nil {
    return Result.Err(err)
}
```

Running `goal fix` should rewrite that block to the equivalent one-liner:

```goal
os.MkdirAll(dir, 0o755)?
```

**Why this priority**: This is the entire feature. It is the single largest blind spot in the
fixer today — `goal fix` already collapses the value-binding form (`x, err := f(); if err …`)
but silently leaves the statement-context form untouched, so a file can end up with a `?`
collapse two lines above an identical-shaped guard the tool ignored. Closing the gap makes the
fixer's behavior consistent and predictable.

**Independent Test**: Feed a single `Result[T, error]` function containing one if-init error
guard to `fix.File` and assert the guard becomes `call(...)?`, the change is recorded, and a
second pass is a no-op (idempotent).

**Acceptance Scenarios**:

1. **Given** a `Result[T, error]` function with `if err := f(args); err != nil { return Result.Err(err) }`, **When** `goal fix` runs, **Then** the block becomes `f(args)?` at the same indentation.
2. **Given** the same function after one fix pass, **When** `goal fix` runs again, **Then** nothing changes.
3. **Given** a file mixing the value-binding form and the statement-context form, **When** `goal fix` runs, **Then** both collapse to `?`.

---

### User Story 2 - Leave anything that is not a provably-equivalent passthrough untouched (Priority: P1)

The author has guards that *look* similar but are not pure passthroughs: the body wraps the
error into a domain type, decorates it with `fmt.Errorf`, returns a non-zero value, has an
`else`, or carries a comment that would be dropped. `goal fix` must leave every one of these
exactly as written.

**Why this priority**: The fixer's core contract is that it never emits behavior-changing code.
A false rewrite here (e.g. dropping an error-wrapping layer) is strictly worse than the missed
collapse the feature set out to fix.

**Independent Test**: Feed each non-conforming shape and assert the source is unchanged.

**Acceptance Scenarios**:

1. **Given** `if err := f(); err != nil { return Result.Err(WrapError(err)) }`, **When** `goal fix` runs, **Then** the block is unchanged.
2. **Given** an if-init guard whose body contains a comment, **When** `goal fix` runs, **Then** the block is unchanged and a Skip report is recorded.
3. **Given** an if-init guard with an `else` clause, **When** `goal fix` runs, **Then** the block is unchanged.
4. **Given** the same shape inside a function that does **not** return `Result[T, error]`, **When** `goal fix` runs, **Then** the block is unchanged (`?` would be illegal there).

---

### Edge Cases

- **Init clause binds a value as well as the error** (`if v, err := f(); err != nil { … }`): out
  of scope — a bare `f()?` statement discards the unwrapped value, and the value cannot be used
  after an if-init binding (Go scoping). Only a single-variable LHS equal to the condition
  variable is collapsed.
- **Bare `return err`** (not `Result.Err(err)`): not a valid propagation inside a Result
  function, so it is not matched (consistent with the value-binding rule).
- **Composite-literal init** (`if x := T{}; …`): no top-level `;` is found before the body brace
  (the brace belongs to the literal), so the candidate is skipped.
- **Closed-E `Result[T, E]` functions** (`ModeResultClosed`): out of scope, matching the existing
  `fixPropagate` which only collapses open-E `ModeResult`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `goal fix` MUST collapse `if condVar := CALL; condVar != nil { <propagation return> }`
  to `CALL?` when the enclosing function returns `Result[T, error]` (`ModeResult`) and the init
  clause binds exactly the condition variable (the call's only output is the error).
- **FR-002**: The `<propagation return>` body MUST be a provably-equivalent passthrough — the same
  shapes the existing `validPropagationReturn` accepts for Result mode (`return Result.Err(err)`
  or `return zero, err`). Any other body leaves the block untouched.
- **FR-003**: The rewrite MUST preserve the original line's indentation.
- **FR-004**: A guard carrying a comment MUST be left untouched and recorded as a `propagate` Skip
  report (consistent with the value-binding rule).
- **FR-005**: A guard followed by `else`, or inside a non-`Result` function, MUST be left untouched.
- **FR-006**: The rewrite MUST be idempotent — running `goal fix` on already-fixed output produces
  no further changes.
- **FR-007**: The new rule MUST NOT alter the existing value-binding collapse, signature
  conversion, switch-to-match, or call-site reporting behavior.

### Key Entities

- **Init-clause error guard**: an `if` statement whose init clause is `err := CALL` and whose
  condition is `err != nil`, with a body that is a single propagation return.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All six classes of call described in the bug report (`protojson.Unmarshal`,
  `toml.Unmarshal`, `os.MkdirAll`, `os.WriteFile`, `json.Unmarshal`, and any single-error-output
  call) collapse to `CALL?`.
- **SC-002**: Zero behavior-changing rewrites: every error-wrapping / decorating / non-zero-value
  / commented / else-bearing guard is left byte-for-byte unchanged.
- **SC-003**: The full existing test suite (`go test ./...`) continues to pass.

## Testing Requirements *(mandatory)*

### Test Strategy

Unit tests in `internal/fix/fix_test.go`, the package's existing convention (stdlib `testing`
only — the goal project is zero-dependency, no testify). Each test feeds a small source string
to `fix.File` and asserts on the returned source / changes / reports, mirroring the existing
`TestCollapseInsideResultFunc` style. Idempotence is asserted by a second `File` call.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Unit | if-init guard in a Result function collapses to `CALL?` |
| FR-002 | Unit | `Result.Err(Wrap(err))` body is left untouched |
| FR-003 | Unit | collapsed line keeps the original indentation (asserted via exact-match `want`) |
| FR-004 | Unit | commented guard is unchanged + `propagate` Skip report |
| FR-005 | Unit | guard inside a non-Result function is unchanged |
| FR-006 | Unit | second `File` pass on fixed output yields no changes |
| FR-007 | Unit | existing suite unchanged (regression) |

### Edge Case Coverage

- Value+error LHS in init clause → not collapsed (covered by single-var LHS guard).
- `else` clause present → not collapsed.

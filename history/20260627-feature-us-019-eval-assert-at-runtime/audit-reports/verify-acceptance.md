# Verify — Acceptance

## Verify gates (prd.json verifyCommands) — all green

- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (all packages pass, incl. internal/interp)

## Acceptance criteria

AC1 — "a false condition panics with the located assertion message; a true
condition is a no-op": satisfied by `execAssert` (internal/interp/assert.go).
True -> returns nil; false -> `panicSignal{StrVal("<pos>: assertion failed:
<cond>[: <fmt msg>]")}` over the unrecovered panic channel.

AC2 — "a unit test over a 10-assert shape asserts the panic on a false assertion
and normal completion on a true one": internal/interp/assert_test.go, modeled on
features/10-assert/examples/{bank,message}.goal:
- TestAssertTrueIsNoOp / TestAssertMessageFormTrueIsNoOp — normal completion.
- TestAssertFalseBarePanicsLocated — panic, "assertion failed" + condition +
  located.
- TestAssertFalseMessageFormFormatsMessage — formatted message in the panic.
- TestAssertNonBoolConditionIsRefused — descriptive refusal (FR-4).

All 5 pass (`go test ./internal/interp -run Assert -v`).

## US-022 envelope

internal/interp gains no new dependency (assert.go imports only fmt, strings,
goal/internal/ast). It stays clear of internal/backend / internal/typecheck /
go/types.

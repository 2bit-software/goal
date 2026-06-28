# Implementation Tasks

## Task 1: Capability-denied error + enforce in emitStdout
**Status**: completed
**Files**: `internal/interp/interp.go`, `internal/interp/host.go`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4, FR-5
**Verify**: `go build ./... && go vet ./...`

### Instructions
- In `internal/interp/interp.go`, add a typed error:
  ```go
  type CapabilityError struct {
      Cap cap.Capability
      Pos token.Pos
  }
  func (e CapabilityError) Error() string {
      return fmt.Sprintf("interp: %s: capability denied: %s not granted", e.Pos.String(), e.Cap)
  }
  ```
  (`cap`, `token`, `fmt` are already imported.)
- Change `emitStdout` to take the effect's source position and turn the
  not-granted branch into a refusal that performs NO write:
  ```go
  func (ip *Interp) emitStdout(pos token.Pos, write func(io.Writer) error) error {
      if !ip.caps.Has(cap.Stdout) {
          return CapabilityError{Cap: cap.Stdout, Pos: pos}
      }
      w := ip.stdout
      if w == nil { w = os.Stdout }
      return write(w)
  }
  ```
  Update the doc comment to state the not-granted branch is now a located, named
  refusal (US-024 done), not a silent skip.
- In `internal/interp/host.go::evalHostCall`, update the lone `emitStdout` call
  (the `fmt.Println` interception) to pass `sel.Pos()` as the new first arg.

## Task 2: Denial tests
**Status**: completed
**Files**: `internal/interp/cap_deny_test.go` (new)
**Depends on**: Task 1
**Spec coverage**: all acceptance criteria
**Verify**: `go test ./internal/interp/... -count=1`

### Instructions
- New file `internal/interp/cap_deny_test.go`, `package interp`, stdlib
  `testing` only (no testify). Mirror `cap_io_test.go`'s setup
  (parser.ParseFile + sema.Resolve + New(...) with a captured `bytes.Buffer`).
- `TestPrintlnUnderDeniedStdoutIsRefused`: New(file, info,
  WithCapabilities(cap.DenyAll()), WithStdout(&buf)); Run() must return an error
  that `errors.As` into `CapabilityError`; assert `.Cap == cap.Stdout`; assert
  the message contains "Stdout" and a position (non-empty `.Pos.String()`);
  assert `buf.Len() == 0` (nothing written).
- `TestEmitStdoutDeniedReturnsLocatedNamedError`: construct an interp with a
  denied set; call `emitStdout` directly with a sentinel write func that sets a
  flag; assert the flag never set (write skipped) and the returned error is a
  `CapabilityError`.
- `TestPrintlnUnderGrantedStdoutStillPrints`: regression — WithCapabilities(
  cap.GrantAll()) + WithStdout(&buf); Run() returns nil and buf has the expected
  output (FR-5).

## Coverage

- Files: interp.go, host.go (Task 1), cap_deny_test.go (Task 2) — all plan
  files covered.
- Requirements: FR-1..FR-5 covered by Task 1, all acceptance criteria asserted
  by Task 2.

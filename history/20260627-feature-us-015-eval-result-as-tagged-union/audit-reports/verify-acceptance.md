# Verify — Acceptance Coverage (US-015)

Gates (prd.json verifyCommands), all green:
- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (all packages pass, internal/interp included)

US-022 envelope re-checked: `go list -deps ./internal/interp` has no go/types,
internal/backend, or internal/typecheck.

## Criterion -> evidence

- Result.Ok(x) -> tagged-union tag "Ok", payload x:
  `TestResultOkConstruction` (TypeID "Result", tag "Ok", payload Config{Raw:"hi"}).
- Result.Err(e) -> tagged-union tag "Err", payload e:
  `TestResultErrConstruction` (open-E host error) and
  `TestResultErrCarriesEnumValueClosedE` (closed-E enum payload).
- Match binds unwrapped Ok payload / Err error:
  `TestResultMatchOpenE` (Ok -> cfg.Raw == "hello"; Err -> "empty input"),
  `TestResultMatchClosedE` (Ok -> cfg.Raw; Err -> nested match on the enum).
- Same representation for open-E and closed-E:
  open-E and closed-E programs both produce TypeID "Result" variants;
  `TestResultMatchClosedE` + `TestResultErrCarriesEnumValueClosedE` show the Err
  payload is the enum variant under the identical Result encoding.
- Unknown ctor / wrong arity are located refusals:
  `TestResultUnknownCtorIsRefused`, `TestResultCtorArityIsRefused`.

Every acceptance criterion maps to an asserting test. No gaps.

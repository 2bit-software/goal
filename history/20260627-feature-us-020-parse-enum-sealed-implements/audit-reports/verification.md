# Verification — US-020

## Acceptance criteria

- [x] Enum with data-less variants parses (traffic.goal: Red/Yellow/Green, no
      payload) — `TestParseEnumDataless`.
- [x] Enum with payload variants parses with field name+type, incl. multi-field
      (status.goal: Active{since:Time}, Cancelled{reason:string, at:Time}) —
      `TestParseEnumDecl`.
- [x] `sealed interface Shape {}` parses into a SealedInterfaceDecl with an empty
      method set — `TestParseSealedInterface`.
- [x] `type T struct implements Iface { ... }` parses with the implements clause
      and field list — `TestParseSealedInterface` / `TestParseImplements`.
- [x] Qualified implements (`io.Writer`) parses as a SelectorExpr —
      `TestParseImplements` (Discard case).
- [x] Test parses the 01-enums and 07-implements example inputs — done.

## Verify commands

- `go build ./...` — pass
- `go vet ./...` — pass
- `go test ./... -count=1` — pass (all packages green, incl. internal/parser and
  the existing corpus/pipeline suites; no regressions)

## Notes

`status.goal` / `nested.goal` function bodies use labeled-argument construction
(`Status.Active(since: now())`), which is US-022's grammar; the parser is
error-tolerant and still produces the EnumDecl, so the payload test asserts the
declaration structure without requiring a clean whole-file parse. Whole-corpus
clean parsing is the later US-024 gate.

No CRITICAL or MAJOR findings. Recommend pass.

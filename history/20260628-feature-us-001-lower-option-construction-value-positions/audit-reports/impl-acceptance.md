# Implementation Audit: Acceptance Criteria

## Result: PASS — no CRITICAL, no MAJOR

Verified against `go build ./...`, `go vet ./...`, `go test ./... -count=1` (all green).

- AC1 `x := Option.Some(v)` binds to *T pointer form, no `Option.` token — PASS.
  Generated: `x := &v`; whole file has no `Option.` token.
- AC2 call-arg / struct-field / slice / map positions yield valid Go (go/format),
  no `Option.` — PASS. Generated: `sink(&v)`, `sink(nil)`, `opt: &v`,
  `[]*int{&v, nil}`, `map[string]*int{"a": &v, "b": nil}`.
- AC3 None -> nil, Some(addressable) -> &x, otherwise boxed temp — PASS. Boxed:
  `boxed := goalSome(1)` + the `goalSome[T any]` helper.
- AC4 existing direct-return / Result.Ok-payload Option lowering unchanged — PASS.
  `TestASTEngineLowersNestedOptionInResult` and the whole corpus/pipeline tier green;
  the return path (optionValueExpr) is untouched.
- AC5 backend test exercising all four positions exists and passes — PASS:
  `TestASTEngineLowersOptionInValuePositions`.

## Assumptions
- An identifier argument is the canonical "addressable" case (`&x`); any other arg
  (literal/call/selector) is boxed via `goalSome`. A selector like `a.b` would be
  Go-addressable but is conservatively boxed — still valid Go.
- The boxing helper is appended at end-of-file (single) / a shared `goal_options.go`
  (package), relying on Go's order-independent package-level declarations, so it is
  emitted only when actually used (runtime usedOptionHelper flag), never spuriously.

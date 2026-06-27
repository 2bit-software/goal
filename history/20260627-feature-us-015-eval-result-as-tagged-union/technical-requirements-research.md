# Technical Requirements / Research — US-015

## Existing seams (from progress.txt + code reading)

- The universal tagged union already exists: `interp.VariantVal(typeID, tag, fields)`
  in internal/interp/value.go. Enums already construct via this (US-012). Result
  reuses the SAME constructor with `TypeID == "Result"`, `Tag == "Ok" | "Err"`.
- `Result.Ok(x)` / `Result.Err(e)` are bare positional calls, so they parse to
  `*ast.CallExpr` whose `Fun` is `*ast.SelectorExpr{X: Ident "Result", Sel:
  "Ok"|"Err"}` (NOT a `*ast.VariantLit`, which only forms for labeled args).
  Intercept them in `evalCallMulti`, guarded by "Result not shadowed in scope".
- Match dispatch is already tag-keyed and uniform (US-013/US-014): `selectMatchArm`
  compares `pattern.Variant.Name == subj.Variant.Tag`, so an `Ok`/`Err` arm
  dispatches for free over a Result subject.
- The ONE new match concern: binding. An enum arm (`Event.Login(l)`) binds `l` to
  the WHOLE variant (payload read as `l.field`). A Result arm (`Result.Ok(cfg)`)
  binds `cfg` to the UNWRAPPED payload directly (the backend lowers it to the bare
  value `v` / `err`). So `armScopeFor` must unwrap the single payload for Result
  (and, later, Option) variants while keeping enum bindings as the whole variant.
  Distinguish by `TypeID == "Result"` (US-016 adds "Option").

## Plan

1. value.go: canonical Result payload field names + a `resultPayload` helper that
   returns the single payload value of a sum-payload variant.
2. eval.go: intercept `Result.Ok/Result.Err` construction in `evalCallMulti`
   (before host/method paths), build the VariantVal. Exactly one argument each;
   an unknown `Result.X` ctor is a located, descriptive refusal.
3. interp.go: `armScopeFor` unwraps the single payload for Result variants so an
   `Ok`/`Err` arm binds the inner value/error, not the variant.
4. Tests: drive a 03-result-shaped program (open-E, `Result[Config, error]`) and a
   06-error-e-shaped program (closed-E, `Result[Config, ParseError]`) through the
   interpreter, asserting construction tags + match-arm payload/error binding.

## Constraints

- Zero dependency: stdlib `testing` only, no testify.
- internal/interp must NOT gain a dependency on go/types, internal/backend, or
  internal/typecheck (US-022 envelope). This story adds none.
- Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.

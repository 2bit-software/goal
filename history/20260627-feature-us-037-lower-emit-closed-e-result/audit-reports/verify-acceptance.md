# Verify — Acceptance Coverage

Project gates all green: `go build ./...`, `go vet ./...`, `go test ./... -count=1`
(all packages ok, including goal/internal/backend).

| Acceptance criterion | Evidence |
|---|---|
| Closed-E file emits the generic Result/Ok/Err encoding once, ahead of first use | TestASTEngineClosedResultEncoding asserts `type Result[T, E any] interface{ isResult() }` + `type Ok[T, E any] struct{ Value T }`; emit.go file() emits it once after imports. |
| `Result.Ok`/`Result.Err` -> Ok/Err sum constructors carrying the arg | Encoding test asserts `return Ok[Config, ParseError]{Value: Config{Raw: s}}` and `return Err[Config, ParseError]{Value: ParseError(ParseError_Empty{})}`. |
| Closed-E match -> exhaustive Ok/Err dispatch with carried value bound per used arm | Encoding test asserts `case Ok[Config, ParseError]:`, `case Err[Config, ParseError]:`, and the panicking default; closedResultMatch binds `binding := guard.Value`. |
| Closed-E `?` unwraps on success, propagates failure on error | qclosed_prop_same: `var cfg Config`, `case Ok[Config, ParseError]:`, `return Err[Config, ParseError]{Value:` asserted. |
| From-conversion applied across differing error types; conversion emitted & callable | qclosed_prop_from: `func toApp(e ParseError) AppError {` and `return Err[Config, AppError]{Value: toApp(` asserted. |
| All three 06-error-e inputs pass the behavioral tier through the AST backend | TestASTEngineClosedResultBehavioralTier runs all 3 through corpus.RunCompile (temp-module go build + go vet); passes. |

No acceptance criterion is left without a corresponding test.

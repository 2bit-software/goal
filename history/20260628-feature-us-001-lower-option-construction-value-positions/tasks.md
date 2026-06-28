# Tasks — Lower Option construction in value positions

## T1 [completed]: Add Option classifier + boxing prelude (foundation, no deps)
- File: internal/backend/lower.go
- Add `optionConstruction(x ast.Expr) (kind string, arg ast.Expr, ok bool)`.
- Add `const optionPrelude` (`func goalSome[T any](v T) *T { return &v }` + doc).
- Add `needsOptionPrelude(f *ast.File) bool` via `ast.Walk`/`identFinder`.
- Spec coverage: FR-3 (encoding source of truth), supports FR-1/FR-2.

## T2 [completed]: Intercept Option construction in the value-emission path (deps: T1)
- File: internal/backend/emit.go
- Add `func (e *emitter) tryOptionValue(x ast.Expr) bool` using `optionConstruction`:
  `nil` / `&<ident>` / `goalSome(<arg>)`.
- Add `if e.tryOptionValue(x) { return }` at the top of `expr()`.
- Spec coverage: FR-1, FR-2, FR-3.

## T3 [completed]: Inject the boxing helper once per file/package (deps: T1, T2)
- Files: internal/backend/emit.go (`file()`), internal/backend/package.go
  (`TranspilePackage`).
- Single-file: inject `optionPrelude` after imports when
  `!suppressPrelude && needsOptionPrelude(f)`.
- Package: append shared `goal_options.go` when any file `needsOptionPrelude`.
- Spec coverage: FR-2 (boxed temp valid Go), no-regression for package corpus.

## T4 [completed]: Test (deps: T1–T3)
- File: internal/backend/backend_test.go (external `package backend_test`, stdlib
  `testing`, NO testify).
- `TestASTEngineLowersOptionInValuePositions`: var-assignment, call-argument,
  struct-field, slice-literal, map-literal, plus a non-addressable `Some(literal)`.
  Assert `format.Source` ok, no `Option.` token, `nil` and `&v` present.
- Spec coverage: final acceptance-criteria test.

## T5 [completed]: Verify (deps: T1–T4)
- Run `go build ./...`, `go vet ./...`, `go test ./... -count=1`. All green.
- Confirm existing `TestASTEngineLowersNestedOptionInResult` still passes (FR-4).

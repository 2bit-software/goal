# Implementation Tasks — US-031

## Task 1: Extend Info with interface facts + resolve interfaces
**Status**: completed
**Files**: `internal/sema/sema.go`, `internal/sema/resolve.go`
**Depends on**: none
**Spec coverage**: FR-2 (foundation)
**Verify**: `go build ./internal/sema/...`

### Instructions
- Add `Interfaces map[string][]Method` and `EmbeddedIfaces map[string][]string` to `Info`.
- Initialize both maps in `Resolve`.
- In `resolveTypeDecl`, when a `TypeSpec.Type` is an `*ast.InterfaceType`, record its
  methods (Field with `Names` + `*ast.FuncType`) into `Interfaces[name]` and its
  embedded interface names (Field with no `Names`, `Type` an Ident/Selector) into
  `EmbeddedIfaces[name]`. Build each interface `Method` with `Sig =
  joinTypes(paramTypeListFL(params)) + "|" + joinTypes(paramTypeListFL(results))`,
  `Arity`/`EndsInError` from the result list, `Raw` left empty.

## Task 2: must-use check (feature 03)
**Status**: completed
**Files**: `internal/sema/mustuse.go`, `internal/sema/mustuse_test.go`
**Depends on**: none
**Spec coverage**: FR-1
**Verify**: `go test ./internal/sema/ -run MustUse`

### Instructions
- `CheckMustUse(file, info) []Diagnostic`. Walk the file; a `*ast.ExprStmt` whose `X`
  is a `*ast.CallExpr` with an `*ast.Ident` Fun naming a `ModeResult`/`ModeResultClosed`
  func ⇒ Error `dropped-result` (message mirrors `internal/check/mustuse.go`
  `classifyStatementCall`). A `*ast.AssignStmt` with single Lhs Ident `_` and single Rhs
  CallExpr to such a func ⇒ Warning `unresolved-result-discard`.

## Task 3: implements check (feature 07)
**Status**: completed
**Files**: `internal/sema/implements.go`, `internal/sema/implements_test.go`
**Depends on**: Task 1
**Spec coverage**: FR-2
**Verify**: `go test ./internal/sema/ -run Implements`

### Instructions
- `CheckImplements(file, info)`. Walk decls for `type T struct implements I`
  (`GenDecl(TYPE)`→`TypeSpec`→`StructType.Implements`). For the asserted interface:
  sealed ⇒ skip; qualified/`"."` ⇒ defer Warning `unresolved-interface`; resolve
  required methods (own + embedded via `requiredMethods`, defer if unresolvable); for
  each required method missing ⇒ Error `unimplemented-method`, sig-mismatch ⇒ Error
  `method-signature-mismatch`. Mirror `internal/check/implements.go` messages.

## Task 4: question + closed-E checks (features 05/06)
**Status**: completed
**Files**: `internal/sema/question.go`, `internal/sema/question_test.go`
**Depends on**: none
**Spec coverage**: FR-3, FR-4
**Verify**: `go test ./internal/sema/ -run Question`

### Instructions
- `CheckQuestion(file, info)`: for each plain (`Mod==FuncPlain`, `Recv==nil`) func with
  resolved Mode `ModeResult`, collect `?` sites (AssignStmt Rhs / ExprStmt X that is an
  `*ast.UnwrapExpr`), resolve the callee, apply the open-E rules (mirror
  `internal/check/question.go`).
- `CheckClosed(file, info)`: for each plain func with Mode `ModeResultClosed`: at each
  `?` site enforce From-totality (`missing-from-conversion` / `unresolved-question-error`),
  and at each `Result.Err(X)` (CallExpr Fun SelectorExpr Result.Err) enforce closedness
  (`err-outside-closed-enum` / `unknown-error-variant` / `unresolved-err-value` /
  `unresolved-error-enum`). Mirror `internal/check/closed.go` messages.

## Task 5: wire into sema.Check + corpus runner test
**Status**: completed
**Files**: `internal/sema/check.go`, `internal/corpus/sema_question_test.go`
**Depends on**: Task 2, 3, 4
**Spec coverage**: all AC
**Verify**: `go build ./... && go vet ./... && go test ./... -count=1`

### Instructions
- Append `CheckMustUse`, `CheckImplements`, `CheckQuestion`, `CheckClosed` to
  `sema.Check`.
- Add `TestSemaQuestionImplementsRunner` mirroring `sema_fields_test.go`: walk the
  manifest, run every `KindCheck` case under `testdata/check/03-result/`,
  `testdata/check/06-error-e/`, `testdata/check/07-implements/` through
  `CheckerFunc(SemaCheck)` + `RunCheck`; `t.Fatalf` on zero cases.

# Implementation Tasks — US-006

## Task 1: Copy lexer source to goal
**Status**: completed
**Files**: `selfhost/lexer/lexer.goal`
**Depends on**: (none)
**Spec coverage**: FR-1
**Verify**: file exists, `grep -nE '\b(match|enum|assert)\b'` shows only comment/string hits

### Instructions
- Copy `internal/lexer/lexer.go` verbatim into `selfhost/lexer/lexer.goal`
  (Go superset is valid goal). Keep `package lexer` and the
  `goal/internal/token` + `unicode`/`unicode/utf8` imports unchanged.

## Task 2: Extend BuildAndTest to carry dependency packages
**Status**: completed
**Files**: `internal/selfhost/selfhost.go`, `internal/selfhost/port_test.go`
**Depends on**: (none)
**Spec coverage**: FR-2, FR-3 (enables behavioral gate for a package with an in-module dep)
**Verify**: `go test ./internal/selfhost -run TestPortedTokenPackage`

### Instructions
- Add a `deps map[string]*project.Package` parameter to `BuildAndTest`; before
  transpiling the package under test, transpile and `writePackage` each dep into
  the same temp module (mirroring BuildTranspiled's loop).
- Update the existing `TestPortedTokenPackage` call to pass `nil` for deps.

## Task 3: Add ported-lexer port test
**Status**: completed
**Files**: `internal/selfhost/port_test.go`
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-1, FR-2, FR-3 (all acceptance criteria)
**Verify**: `go test ./internal/selfhost -run TestPortedLexerPackage`

### Instructions
- Add `TestPortedLexerPackage`: `project.Discover` both `../../selfhost/token`
  and `../../selfhost/lexer`; assert one package each, names "token"/"lexer".
- Run `BuildTranspiled` over the layout {"internal/token": token, "internal/lexer": lexer}.
- Run `BuildAndTest("internal/lexer", lexerPkg, ["../lexer/lexer_test.go"],
  {"internal/token": tokenPkg})`.

## Task 4: Verify project gates
**Status**: completed
**Files**: (none — verification only)
**Depends on**: Task 3
**Spec coverage**: all
**Verify**: `task check`, `task build`, `task fixpoint`

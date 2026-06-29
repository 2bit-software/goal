# Implementation Tasks — US-008

## Task 1: Port parser source files to selfhost/parser
**Status**: completed
**Files**: selfhost/parser/parser.goal, selfhost/parser/goal_construct.goal,
selfhost/parser/goal_decl.goal, selfhost/parser/goal_match.goal,
selfhost/parser/goal_stmt.goal
**Depends on**: (none — token/lexer/ast already ported)
**Spec coverage**: FR-1
**Verify**: files exist and `grep -nE '\b(match|enum|assert)\b' selfhost/parser/*.goal`
shows only comments / token.MATCH / token.ENUM (no identifier collisions).

### Instructions
Copy each non-test source file from internal/parser verbatim to
selfhost/parser/<base>.goal (Go superset = valid goal). No edits required; the
imports goal/internal/{ast,lexer,token} resolve against the ported packages, and
errors/fmt pass through.

## Task 2: Add TestPortedParserPackage gate
**Status**: completed
**Files**: internal/selfhost/port_test.go
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-3
**Verify**: `go test ./internal/selfhost -run TestPortedParserPackage`

### Instructions
Mirror TestPortedAstPackage. Discover selfhost/{token,lexer,ast,parser}; assert
each package name. Run BuildTranspiled over layout {internal/token, internal/lexer,
internal/ast, internal/parser} (compile gate). Run BuildAndTest("internal/parser",
parserPkg, ["../parser/parser_test.go"], deps={internal/token, internal/lexer,
internal/ast}) (behavioral gate).

## Task 3: Project gates
**Status**: completed
**Files**: (none — verification only)
**Depends on**: Task 2
**Spec coverage**: all acceptance criteria
**Verify**: `task check` and `task build` both green.

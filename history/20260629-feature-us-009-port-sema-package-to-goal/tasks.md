# Implementation Tasks — US-009

## Task 1: Copy internal/sema sources to selfhost/sema/*.goal
**Status**: completed
**Files**: selfhost/sema/{analyze,assert,check,convert,fields,foreign,implements,mustuse,package,question,resolve,sema}.goal (12 new)
**Depends on**: (none)
**Spec coverage**: FR-1, AC1
**Verify**: `ls selfhost/sema/*.goal | wc -l` == 12; `goal/internal/{token,ast,parser}` imports present.

### Instructions
Copy each non-test internal/sema/*.go to selfhost/sema/<name>.goal verbatim
(Go superset is valid goal). No reserved-word identifier collisions exist
(bare match/enum/assert appear only in string literals). Do NOT copy *_test.go.
There is no reflection/dump file to drop.

## Task 2: Add TestPortedSemaPackage to the self-host gate
**Status**: completed
**Files**: internal/selfhost/port_test.go
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-3, AC2, AC3
**Verify**: `go test ./internal/selfhost -run TestPortedSemaPackage`

### Instructions
Mirror TestPortedParserPackage. Discover selfhost/{token,lexer,ast,parser,sema},
assert each package name. COMPILE gate: BuildTranspiled over layout
{internal/token, internal/lexer, internal/ast, internal/parser, internal/sema}
(lexer included because the transpiled parser imports goal/internal/lexer).
BEHAVIORAL gate: BuildAndTest("internal/sema", semaPkg, <self-contained test
files>, deps={token,lexer,ast,parser}). Start with the candidate test set
(assert, check, convert, implements, mustuse, question, resolve, sema_test.go);
exclude foreign_test.go and package_test.go (depend on testdata/extpkg).
Narrow the set if a copied test references an excluded symbol.

## Task 3: Mark story passed and log
**Status**: completed
**Files**: prd.json, progress.txt
**Depends on**: Task 2
**Spec coverage**: closeout
**Verify**: `task check && task build`; `task fixpoint`

### Instructions
After all gates green, set US-009 passes:true in prd.json and append the
progress.txt entry. (The loop-runner / morse-code commit handles the commit.)

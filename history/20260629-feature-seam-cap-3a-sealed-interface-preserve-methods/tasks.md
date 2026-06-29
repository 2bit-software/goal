# Implementation Tasks

## Task 1: Preserve declared methods in the live transpiler (internal/)
**Status**: completed
**Files**: internal/backend/emit.go
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3
**Verify**: `go build ./... && go test ./internal/backend/`

### Instructions
- Extract the interface-method-emit loop body from `interfaceType` (emit.go
  ~L521-540) into a new helper `func (e *emitter) interfaceMethod(m *ast.Field)`
  that emits a single method element (named method via `identList` + `funcSig`,
  embedded interface via `expr`) WITHOUT the trailing newline. Have `interfaceType`
  call it inside its loop, then `e.p("\n")`.
- Rewrite `sealedInterfaceDecl` (emit.go ~L240-246):
  - if `d.Name == nil`: keep the existing `e.fail(...)`.
  - if `d.Methods == nil || len(d.Methods.List) == 0`: keep
    `e.p(genSealedInterface(d.Name.Name))` (compact, byte-identical — FR-3).
  - else: emit `type <Name> interface {\n`, then for each method call
    `interfaceMethod` + `e.p("\n")`, then the marker `is<Name>()\n`, then `}`.
- Output is gofmt-normalized downstream, so exact whitespace is not load-bearing.

## Task 2: Mirror in the self-hosted transpiler (selfhost/)
**Status**: completed
**Files**: selfhost/backend/emit.goal
**Depends on**: Task 1
**Spec coverage**: FR-5 (and FR-1/2/3 in the .goal mirror)
**Verify**: `task fixpoint`
### Instructions
- Apply the identical `interfaceMethod` extraction + `sealedInterfaceDecl` rewrite
  to selfhost/backend/emit.goal (its `interfaceType`/`funcSig`/`identList` are
  line-for-line identical to internal/). The .goal source is a Go superset, so the
  code is the same. Keep `genSealedInterface` in lower.goal unchanged.

## Task 3: Regression test
**Status**: completed
**Files**: internal/backend/sealed_methods_test.go (new)
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-4
**Verify**: `go test ./internal/backend/`
### Instructions
- Shape test (`backend.Transpile`): inline goal source with
  `sealed interface Node { Pos() Position; End() Position }`, a `Position` struct,
  a `Circle struct implements Node` with value-receiver `Pos`/`End`, and a func
  taking `Node`. Assert emitted Go contains `Pos() Position`, `End() Position`, and
  `isNode()`.
- Behavioral test (skipped under `-short`): transpile a package via
  `backend.TranspilePackage`, write the .go to a temp `module goal`, add a test
  that constructs the implementor, assigns it to a `Node` variable, and calls
  `Pos()`/`End()` through it; `go test` must pass. (If the methods were dropped from
  the interface, the call through the interface value would fail to compile.)
  Follow the temp-module pattern in internal/backend/crosspkg_goal_enum_test.go
  (`writeFile`, `exec.Command("go", "test", ...)`).

## Whole-tree gates (run after all tasks)
`task check`, `task build`, `task fixpoint`; corpus behavioral tier unchanged.

# Implementation Tasks

## Task 1: Propagate sealed implementor sets in internal/sema/foreign.go
**Status**: completed
**Files**: internal/sema/foreign.go
**Depends on**: (none)
**Spec coverage**: FR-1, FR-3 (capability), AC-1, AC-2 (internal half)
**Verify**: `go build ./internal/...`

### Instructions
- Add a 6th return value `sealed map[string][]string` to `foreignDecls` and
  `goalForeignDecls`.
- `.go` path of `foreignDecls`: return `nil` for sealed (out of scope; document).
- `goalForeignDecls`: after `info := ResolvePackage(files)`, build `sealed`; for each
  `iface` in `info.Sealed` where `isExportedName(iface)`, set
  `sealed[alias+"."+iface]` = each impl in `info.SealedImpls[iface]` requalified via
  `qualifyForeignType(impl, alias)`.
- `EnrichForeign`: nil-init `info.Sealed`/`info.SealedImpls`; update the call site to
  `structs, funcs, methods, enums, sealed, err := foreignDecls(...)`; merge
  `info.Sealed[iface]=true` and `info.SealedImpls[iface]=impls`.

## Task 2: Mirror in selfhost/sema/foreign.goal
**Status**: completed
**Files**: selfhost/sema/foreign.goal
**Depends on**: Task 1
**Spec coverage**: AC-2 (selfhost half), AC-3 (fixpoint)
**Verify**: `task check` (port gate compiles selfhost/sema)

### Instructions
- Apply the SAME edits line-for-line (the .goal is real Go compiled by the port gate).
  Keep comments/structure identical to internal/sema/foreign.go where they already match.

## Task 3: Sema cross-package enrichment + exhaustiveness test
**Status**: completed
**Files**: internal/sema/testdata/sealedshape/shape.goal, internal/sema/crosspkg_sealed_test.go
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-3, AC-4
**Verify**: `go test ./internal/sema/ -run CrossPkgSealed`

### Instructions
- Fixture: `sealed interface Node {}` + `type Lit struct implements Node { Val int }` +
  `type Neg struct implements Node { Inner Node }`.
- Test: parse a consumer importing the fixture path; `info := Resolve(file)`; run
  `EnrichForeign` with an injected resolver returning the fixture dir; assert
  `info.Sealed["shape.Node"]` and `info.SealedImpls["shape.Node"]` contains `*shape.Lit`,
  `*shape.Neg`; assert `CheckExhaustive` is clean for a complete match and yields one
  `non-exhaustive-match` Error for a match missing `*shape.Neg`.

## Task 4: Backend behavioral cross-package test + fixtures
**Status**: completed
**Files**: internal/backend/testdata/goalsealed/shape/shape.goal,
  internal/backend/testdata/goalsealed/use/use.goal, internal/backend/crosspkg_sealed_test.go
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-4, AC-3
**Verify**: `go test ./internal/backend/ -run CrossPackageGoalSealed`

### Instructions
- Mirror crosspkg_goal_enum_test.go: reuse `transpileGoalPkg`/`writeFile` helpers.
- shape.goal defines the sealed interface + implementors; use.goal has a cross-package
  `match` over `shape.Node` returning per-implementor values.
- Lowering assertion: transpiled use.go contains `.(type) {`, `case *shape.Lit:`,
  `case *shape.Neg:`.
- Behavioral: transpile both per-package, write into temp module, run a reference
  `switch x := n.(type)` test; assert agreement.

## Task 5: Gates + DECISIONS.md
**Status**: completed
**Files**: DECISIONS.md
**Depends on**: Tasks 1-4
**Spec coverage**: AC-5
**Verify**: `task check && task build && task fixpoint`

### Instructions
- Add a "SEAM-CAP-3c" section to DECISIONS.md (mirror CAP-3b's entry style).
- Run all three gates; confirm FIXPOINT OK and corpus behavioral tier unchanged.

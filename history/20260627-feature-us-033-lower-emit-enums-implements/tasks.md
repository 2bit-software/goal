# Implementation Tasks — US-033

## Task 1: Add goal-construct encoders (lower.go)
**Status**: completed
**Files**: internal/backend/lower.go (new)
**Depends on**: (none)
**Spec coverage**: AC1
**Verify**: `go build ./internal/backend/`

### Instructions
Add package-level helpers mirroring internal/pass encoders, operating on sema
types:
- `genEnum(e *sema.Enum) string` — `type Name interface{ isName() }`, per-variant
  structs, per-variant `func (Name_V) isName() {}`.
- `genSealedInterface(name string) string` — `type Name interface{ isName() }`.
- `exported(name string) string` — capitalize first rune.
- `pointerReceiverSet(f *ast.File) map[string]bool` — types with a pointer
  receiver (Recv field Type is *ast.StarExpr, star-strip to Ident name).

## Task 2: Thread sema.Info through the emitter
**Status**: completed
**Files**: internal/backend/backend.go, internal/backend/emit.go
**Depends on**: Task 1
**Spec coverage**: AC1
**Verify**: `go build ./internal/backend/`

### Instructions
- backend.go: Transpile calls `sema.Resolve(file)`; goBackend.Emit passes info to
  `emitFile(file, info)`.
- emit.go: emitter struct gains `info *sema.Info` and `pointerRecv map[string]bool`;
  emitFile(f, info) initializes them (pointerReceiverSet(f)).

## Task 3: Emit enum + sealed declarations and implements marker/assertion
**Files**: internal/backend/emit.go
**Depends on**: Task 2
**Spec coverage**: AC1, AC2 (07-implements, shape/traffic)
**Verify**: `go build ./internal/backend/`

### Instructions
- `decl`: add `*ast.EnumDecl` -> p(genEnum(info.Enums[name])),
  `*ast.SealedInterfaceDecl` -> p(genSealedInterface(name)).
- `structType`: drop the `implements` clause (emit plain struct body; do not fail).
- After a TypeSpec whose StructType has Implements, emit the marker/assertion:
  sealed -> `func (T) is<I>() {}`; else pointerRecv[T] -> `var _ I = (*T)(nil)`;
  else `var _ I = T{}`. Iface name rendered from clause.Type (Ident or Selector).

## Task 4: Lower variant construction
**Files**: internal/backend/emit.go
**Depends on**: Task 2
**Spec coverage**: AC1, AC2 (01-enums status/traffic/nested)
**Verify**: `go build ./internal/backend/`

### Instructions
- `expr` `*ast.VariantLit`: when Enum is *Ident in info.Enums, emit
  `Enum(Enum_V{Label: value, ...})` (labels exported, values via e.expr recursive).
- `expr` `*ast.SelectorExpr`: when X is *Ident in info.Enums and Sel in VSet, emit
  `Enum(Enum_V{})`; else fall through to ordinary selector emission.

## Task 5: Behavioral-tier + focused tests
**Files**: internal/backend/backend_test.go
**Depends on**: Task 3, Task 4
**Spec coverage**: AC2
**Verify**: `go test ./internal/backend/ -run EnumsImplements -count=1`

### Instructions
- TestASTEngineEnumsImplementsBehavioralTier: drive the 4 features/01-enums +
  3 features/07-implements transpile cases through backend.Transpile +
  corpus.RunCompile(repoRoot, c, ...), `-short`-skipped, loud zero-case guard.
- TestASTEngineEnumEncoding / implements: focused transpile+assert on the
  emitted encoding text (interface, construction, sealed marker, (*T)(nil)).

## Task 6: Full verify gates
**Depends on**: Task 5
**Verify**: `go build ./...`, `go vet ./...`, `go test ./... -count=1`

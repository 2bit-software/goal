# Plan Audit — Buildability

- Dependency order valid: sema.Resolve + Info already exist (US-027/US-031);
  the emitter only reads Info. No forward references.
- Interface contracts agree: Backend.Emit(file, info) signature is unchanged;
  only the body of Transpile (New -> Resolve) and emitFile's arity change, both
  internal to package backend.
- File paths verified: internal/backend/{backend.go,emit.go,backend_test.go}
  exist; lower.go is new in the same package.
- Compiles at each step: emit.go new cases consult info maps that Resolve
  initializes (never nil), so plain-Go files keep compiling.
- Integration point specific: Transpile calls sema.Resolve(file); goBackend.Emit
  passes info to emitFile(file, info); emitter stores info + a pointerRecv set
  computed from File.Decls.

One watch item: the data-less variant lowering sits in the SelectorExpr expr
case, which is on a hot path — the info.Enums membership guard must come first so
ordinary selectors are untouched. Noted in the plan.

Buildable without guessing.

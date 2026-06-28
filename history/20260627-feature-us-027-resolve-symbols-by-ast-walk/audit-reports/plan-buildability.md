# Plan Audit: Buildability — US-027

## Checks
- Dependency order (types → typeString → Resolve → test) is a valid topological
  sort; no forward references.
- Interface contracts use real Go syntax and reference existing ast nodes
  (EnumDecl/Variant/PayloadField, StructType/FieldList/Field, FuncDecl.Mod/Recv/
  Type, IndexExpr/IndexListExpr, SealedInterfaceDecl) — all verified present in
  internal/ast.
- File paths are real: internal/sema exists (sema.go, sema_test.go). New files
  resolve.go / resolve_test.go do not collide.
- No import cycle: sema imports ast (+token via ast); test imports parser+analyze,
  neither of which imports sema.
- Each layer compiles independently: sema.go types compile alone; resolve.go
  compiles against ast; test compiles against the public API.

## Findings
- MINOR: lone parenthesized single result `(T)` is ambiguous post-parse vs bare
  `T`; representative inputs avoid it. Documented.
- No CRITICAL/MAJOR.

## Assumptions
- `analyze.Mode` int values line up with `sema.Mode` because both use the same
  iota order (None, Result, ResultClosed, Option) — the test may compare via int.
- Method Arity/EndsInError use raw result counts (no Result/Option override),
  matching `analyze.methodFrom`.

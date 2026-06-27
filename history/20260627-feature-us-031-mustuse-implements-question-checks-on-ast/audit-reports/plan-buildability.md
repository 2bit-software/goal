# Plan Audit — Buildability

- **Dependency order valid.** Info extension (1) precedes implements (3); the three
  checks (2,3,4) precede the `sema.Check` wiring (5); tests (6) last. No forward refs.
- **Interface contracts agree.** New checks share the signature
  `func(*ast.File, *Info) []Diagnostic` (same as `CheckExhaustive`/`CheckFields`), so
  wiring into `Check` is `append(diags, CheckX(file, info)...)`.
- **Signature normalization consistent.** Interface `Method.Sig` is built with the same
  `paramTypeListFL`/`joinTypes` helpers used by `resolveMethod`, so interface-vs-concrete
  comparison is apples-to-apples.
- **File paths verified.** All new files are in existing dirs (`internal/sema`,
  `internal/corpus`); no path collisions (existing files: check.go, fields.go,
  resolve.go, sema.go, *_test.go — new names distinct).
- **Compiles at each step.** Each check is self-contained; helpers it needs
  (`exprName`, `visitorFunc`, `funcSig`, `typeString`) already exist in the package.

One watch item (MINOR): `Info` map additions must be initialized in `Resolve` to stay
nil-safe for callers that read them; the plan calls this out. Ready to build.

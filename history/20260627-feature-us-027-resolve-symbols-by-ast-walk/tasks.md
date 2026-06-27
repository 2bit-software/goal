# Tasks — US-027 Resolve symbols by AST walk

Status: T1 completed, T2 completed, T3 completed. All verify gates green.

## T1 — sema.Info fact types (foundation)
- Expand `internal/sema/sema.go`: add `Mode` (iota order None/Result/ResultClosed/
  Option), `FuncSig`, `Field`, `Variant`, `Enum`, `ConvEntry`, `Method`; give
  `Info` the maps (FuncSignatures/Enums/Sealed/Structs/FromRegistry/Methods).
  Keep `New()` returning an empty placeholder `*Info`.
- Covers: FR-1..FR-5 (data shapes).
- Files: internal/sema/sema.go.
- Verify: `go build ./internal/sema`.

## T2 — Resolve + typeString (AST walk)
- New `internal/sema/resolve.go`: `typeString(ast.Expr) string` and
  `Resolve(*ast.File) *Info` with per-decl walkers for EnumDecl, GenDecl(TYPE)+
  StructType, SealedInterfaceDecl, and FuncDecl (plain signature, from/derive
  conversion, method-by-receiver). Result/Option mode detection off the result
  list (IndexExpr/IndexListExpr head Ident).
- Covers: FR-1..FR-6.
- Files: internal/sema/resolve.go.
- Verify: `go build ./... && go vet ./internal/sema`.

## T3 — Parity + comma-bug test
- New `internal/sema/resolve_test.go` (`package sema`): `TestResolveMatchesAnalyze`
  (compare resolved symbols vs `analyze.Build` for the representative source) and
  `TestResolveStructCommaFieldType` (func-typed comma field resolves correctly).
- Covers: AC-2, AC-3.
- Files: internal/sema/resolve_test.go.
- Verify: `go test ./internal/sema -count=1`, then full
  `go build ./... && go vet ./... && go test ./... -count=1`.

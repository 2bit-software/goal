# Technical Requirements / Research — US-038

Architecture: REWRITE-ARCHITECTURE.md Phase 2 AST backend. Goal-construct lowering
is folded into `internal/backend` (no separate `lower` package): encoders in
`internal/backend/lower.go`, dispatch cases in `internal/backend/emit.go`. Mirror
the known-good legacy splice passes but read off resolved `sema.Info` / the AST
rather than token scans.

Legacy references to mirror:
- `internal/pass/defaults.go` — `...defaults` expansion + `zeroSafety`; uses
  `analyze.ZeroLit` + `analyze.Tables.TypeDecls`.
- `internal/pass/assert.go` — `assert` -> `if !(cond) { panic(...) }`, printf
  message form, and `fmt` import injection.

AST facts available:
- `sema.Info.Structs[name]` — ordered struct fields (name + type string).
- `sema.Info.Enums` / `sema.Info.Sealed` — sum-type membership (zero-unsafe).
- The parser emits `...defaults` as `*ast.SpreadElement{X: *ast.Ident "defaults"}`
  inside a `*ast.CompositeLit`, and `assert` as `*ast.AssertStmt{Cond, Msg, Args}`.

Plan:
- emit.go: add `*ast.AssertStmt` case in `stmt()`; expand `...defaults` in the
  CompositeLit emission; inject `import "fmt"` in `file()` when a message-form
  assert is present and fmt is not already imported.
- lower.go: a `typeDecls` map built from the File's TypeSpecs (struct/interface/
  alias underlying), plus `zeroLit` + `zeroSafety` mirrored from the legacy pass
  reading that map + sema enum/sealed sets.

Tests (behavioral tier, mirror existing US-03x tests): run the 3 08-no-zero-value
and 3 10-assert cases through `corpus.RunCompile` + `backend.Transpile`, plus a
focused encoding test pinning the expanded-defaults and assert shapes. Zero-dep,
stdlib `testing` only.

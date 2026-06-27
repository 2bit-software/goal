# Research — US-033

This is an internal lowering task; the canonical encoding already exists in-repo
(the legacy splice passes), so no external research was required. Findings:

- The §8.1 enum sum encoding and the §8.5 implements assertion/marker are
  defined verbatim in `internal/pass/enums.go` (genEnum/genInterface/construct/
  exported) and `internal/pass/implements.go` (genMarker, scanPointerReceivers).
  These produce known-good, build+vet-clean Go and are mirrored here on the AST.
- The behavioral-tier seam is `corpus.RunCompile(repoRoot, case,
  corpus.TranspilerFunc(backend.Transpile))` — writes the emitted Go into a temp
  module and runs `go build` + `go vet` (used by US-026/US-032).
- sema already resolves Enums/Sealed/Methods (US-027/US-031); the only fact it
  does NOT carry is whether a receiver is a pointer (Methods are star-stripped),
  so pointer-receiver detection is computed locally in the emitter by walking
  File.Decls for a FuncDecl whose receiver Type is an *ast.StarExpr.

Decision: fold lowering into the emitter (no separate `lower` package — the arch
doc marks `lower`/`ir` as an OPTIONAL middle layer), consuming `*sema.Info`,
matching the existing emit.go direct-emission pattern.

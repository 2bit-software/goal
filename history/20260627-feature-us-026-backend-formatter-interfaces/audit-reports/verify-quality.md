# Verify: Quality — US-026

## Error handling
- `Transpile` wraps parse errors (`parse: %w`) and format failures (with the
  unformatted Go attached, mirroring pipeline's style), and propagates emitter
  errors verbatim.
- The emitter records the FIRST error and short-circuits (`emitter.err`), so an
  unsupported node yields one clear `backend: unsupported ... %T` message rather
  than malformed output.
- `parseFlags` rejects unknown engine values and a bare `--engine`.

## Default-unchanged guarantee (FR-6)
- When `--engine` is absent, `parseFlags` returns `engineSplice` and
  `transpileAll` calls `pipeline.TranspilePackage` exactly as before. The splice
  code path is byte-for-byte unchanged; existing pipeline/cmd tests still pass.

## Scope discipline (no creep into US-032+)
- The emitter covers only the plain-Go subset; goal-specific nodes
  (EnumDecl/MatchExpr/UnwrapExpr/VariantLit/SpreadElement/AssertStmt, from/derive
  modifiers, struct `implements`) return an error. This is correct for a seam
  story — full subset is US-032, goal lowering is US-033+.
- `sema.Info` is a deliberate placeholder (US-027).

## Tests assert real behavior
- `TestASTEngineBehavioralTier` actually compiles + vets the generated Go in a
  temp module (not a string compare), so it proves behavior, not spelling.
- `TestBuildWithASTEngine` drives the real `run([]string{"build","--engine=ast",dir})`
  path, proving the flag is wired through the driver, not just parsed.
- `TestGoFormatterFormats` asserts idempotence and a concrete formatted signature.

## Findings
- MINOR: `--engine` is accepted on `check` (shared `parseFlags`) but inert there;
  documented in code comments and not advertised in the guide for `check`. A
  future story wires the checker onto the AST. Not a defect for US-026.

No CRITICAL/MAJOR findings.

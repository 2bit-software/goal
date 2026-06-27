# Plan Audit — Buildability (US-025)

## Findings

No CRITICAL or MAJOR findings.

- Dependency order valid: `interp.RunDoctests` (leaf, depends only on existing
  parser/sema/ast/interp) is built first; `corpus.RunInterp` depends on it. No
  forward references.
- Interface contracts agree: `RunInterp(root string, c Case) error` matches the
  shape of sibling runners; `RunDoctests(src string) ([]DoctestFailure, int, error)`
  is consumed exactly once, in `RunInterp`.
- File paths verified against the tree: `internal/interp/` and `internal/corpus/`
  both exist; the four new filenames do not collide with existing files.
- Import-cycle check: `interp` does not import `corpus`; `corpus` importing `interp`
  adds no cycle. The corpus test stays in `package corpus` because `RunInterp` pulls
  in `interp` (not pipeline/check), so no internal/external split is forced.
- Expression-node extraction: `parser.ParseFile` is the only exported parse entry;
  the `__dt := <input>` wrap-and-lift is the established way to obtain an `ast.Expr`.
  `ip.evalExpr(expr, ip.root)` is proven to return a single Value for a call (see
  call_test.go evalFn) and handles ParenExpr/CallExpr/SelectorExpr/VariantLit.

## Assumptions

- The four committed doctest cases produce int/string results whose `Value.String()`
  rendering equals the locked expected line verbatim. Verified by inspection of the
  `.goal` fixtures and their expected outputs.
- Building the interpreter does not run `main` or the sema gate for doctest eval; the
  doctest runner evaluates expressions directly against the registered root scope.

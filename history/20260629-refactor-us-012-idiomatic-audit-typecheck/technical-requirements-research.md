# Technical requirements / research — US-012

## Idiom-conversion rules (from prd notes + progress.txt patterns)

- open-E `(T, error)` -> `Result[T, error]` lowers to byte-identical Go and is safe on an
  exported fn ONLY when it has no in-tree callers AND no oracle test pins it.
- `?` propagates the error UNCHANGED; error-WRAPPING (`fmt.Errorf("ctx: %w", err)`) is NOT a
  `?` site.
- The backend emitter requires a `?`'s host function to be Result-returning.
- A goal `enum` lowers to a sealed interface (unordered, no integer identity); an iota
  `type X int` consumed via `==` / numeric literals / array-index / range does NOT fit enum.
- Machine check: `goal fix <pkg>/*.goal` must produce no source diff (a `skipped`/`suggestion`
  report is not an auto-conversion).

## Package survey (selfhost/typecheck)

- Only two error-returning functions exist:
  - `Load(pkg) (*Package, error)` — exported; all internal propagation WRAPS context
    (`fmt.Errorf(... %w ...)`); callees (`backend.TranspilePackage`, `parser.ParseFile`) are
    Go-tuples not Result; oracle-pinned by 6+ `p, err := Load(...)` tuple call sites and has
    in-tree caller `Check`.
  - `GoTypesChecker.Check(pkg) ([]Diagnostic, error)` — TypeChecker interface method; pinned by
    `var _ TypeChecker = GoTypesChecker{}` and `tc.Check(pkg)` tuple calls.
- All depth checks (`CheckImplements`/`CheckMustUse`/`CheckNoZeroValue`) return `[]Diagnostic`
  with no error — not fallible.
- Helper predicates are bool / comma-ok (`mustUseFieldKind`, `litClassOf`, `litFieldKeys`) —
  not `?` sites.
- `litClass` (`type litClass int` + iota: classElided/classGeneric) is consumed via
  `kind == classGeneric` and a `return 0, false` numeric literal; the only switch nearby is a
  TYPE switch over `go/ast` types, not over `litClass`. No switch-over-in-file-enum exists.

## Expected outcome

Documented refusal (no `.goal` source change), mirroring US-008 (parser) and US-010
(project/pipeline): every fallible function is exported/interface-pinned with wrapping or
non-Result-host propagation, and `litClass` is an iota int that does not fit `enum`.
`goal fix` already produces zero diff. Deliverable is the DECISIONS.md refusal entry.

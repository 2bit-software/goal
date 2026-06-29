# Tasks

Status: Tasks 1-5 completed. All gates green (task check, task build,
task fixpoint=FIXPOINT OK); corpus behavioral tier unchanged.

## Task 1 — Foreign enum reconstruction in sema (foundation)
- Files: `internal/sema/foreign.go`
- `foreignDecls` gains a 4th return `enums map[string]*Enum` (keyed `alias.Enum`),
  reconstructed from the §8.1 encoding: an exported interface `type X interface{ isX() }`
  is a marker for enum X; exported struct types `X_<Variant>` are its variants.
- `EnrichForeign` initializes `info.Enums` if nil and folds the reconstructed enums in.
- Spec coverage: FR-2.

## Task 2 — Qualified variant pattern in matchQualifier
- Files: `internal/backend/lower.go`
- `matchQualifier` returns `"pkg.Enum"` when the first arm's `vp.Enum` is a
  `*ast.SelectorExpr` (X is an *ast.Ident pkg, Sel is the enum). Bare `*ast.Ident`
  behavior preserved.
- Spec coverage: FR-1, FR-3 (case-label builder already correct for qualified names).

## Task 3 — Foreign fixture + corpus case + behavioral test
- Files: `internal/backend/testdata/extenum/extenum.go`,
  `testdata/package/cross-pkg-enum/use.goal`,
  `internal/backend/crosspkg_enum_test.go`, `corpus/manifest.json`
- Foreign Go fixture carries the §8.1 encoding of `Light { On Off }`.
- Goal fixture `match`es over `light.Light` in statement and return position.
- Unit test asserts transpile succeeds, emits `case light.Light_On/Off:`, and the
  mapping equals the equivalent hand-written switch.
- Corpus case `testdata-package-cross-pkg-enum` proves transpile + `go build`.
- Spec coverage: FR-1, FR-2, FR-3, FR-4, all acceptance criteria.

## Task 4 — selfhost mirrors
- Files: `selfhost/sema/foreign.goal`, `selfhost/backend/lower.goal`
- Apply Task 1 + Task 2 changes to the `.goal` mirror so fixpoint stays green and
  the capability exists in the self-hosted compiler.
- Spec coverage: FR-1, FR-2 (self-host parity); gate: `task fixpoint`.

## Task 5 — Verify gates
- Run `task check`, `task build`, `task fixpoint`; run the new corpus case and
  unit test; confirm corpus behavioral tier unchanged.
- Spec coverage: final acceptance criterion.

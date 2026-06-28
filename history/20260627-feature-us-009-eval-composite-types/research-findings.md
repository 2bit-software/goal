# Research Findings — US-009

No external research required: this story extends the existing in-tree
interpreter using patterns already established by US-005..US-008.

## Findings (from the codebase)

- `internal/interp/eval.go` `evalExpr` is the expression-dispatch switch; new
  node kinds (`*ast.CompositeLit`, `*ast.SelectorExpr`, `*ast.IndexExpr`) slot
  in as new cases returning `(Value, error)`.
- `internal/interp/interp.go` `execStmt` is the statement-dispatch switch;
  `*ast.RangeStmt` slots in alongside the existing `*ast.ForStmt`. Range was
  deliberately deferred from US-008 to this story (per progress.txt US-008 note).
- The value model (`value.go`) already provides the composite carriers:
  `StructValue` (pointer, `Fields map[string]Value`), `[]Value` slices,
  `MapValue` (pointer, `Entries map[string]Value`, STRING-KEYED in v1). The
  constructors `StructVal` / `SliceVal` / `MapVal` exist. No value-model change
  is needed; `Equal`/`String` already cover all three.
- Reference semantics: a `MapValue` and `StructValue` are pointers and a `[]Value`
  slice shares its backing array, so index/field assignment through a looked-up
  binding mutates in place — matching Go.
- `bindTargets` (interp.go) currently rejects non-`*ast.Ident` targets; it must
  grow `*ast.IndexExpr` (map/slice element) and `*ast.SelectorExpr` (struct
  field) target handling for key/element/field assignment.

## Decisions

- Maps stay string-keyed per the documented v1 value model; a non-string map
  key is a descriptive refusal (non-string keys deferred to a later story).
- Struct composite literals use keyed `field: value` elements (the goal idiom);
  positional struct literals are out of scope and refused descriptively.
- internal/interp stays dependency-clean (no go/types, backend, typecheck) for
  the US-022 gate.

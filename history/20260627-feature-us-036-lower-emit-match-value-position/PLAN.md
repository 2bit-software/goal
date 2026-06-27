# Implementation Plan — US-036

## Components

### 1. `enumMatch` lowering (internal/backend/emit.go)
A single helper `enumMatch(m *ast.MatchExpr, pos matchPos, name string)` that
emits the type-switch over the §8.1 encoding:

- Compute `usesBinding` by walking each non-rest arm: a binding is "used" when
  `vp.Binding != nil && usesIdent(arm.Body, vp.Binding.Name)`.
- If any binding is used, mint `val := e.gensym("v")` and open
  `switch val := <subject>.(type) {`; else `switch <subject>.(type) {`.
- For each variant-pattern arm emit `case <Enum>_<Variant>:` then the arm body
  (wrapped per `pos`). Rename the arm's binding to `val` for the body via the
  existing `e.renames` mechanism, and set `e.armBinding`/`e.armFields` so field
  accesses on the binding export (`a.since` -> `val.Since`).
- For the `_` rest arm (`*ast.RestPattern`), buffer it and emit as `default:`;
  otherwise emit a `default:` that panics with
  `unreachable: non-exhaustive <Enum> (compiler invariant violated)`.
- Position wrapping (new `matchPos` enum: posStmt/posReturn/posVar):
  statement -> body; return -> `return <body>`; var -> `name = <body>`.

### 2. Field export in `selectorExpr` (emit.go)
Add new emitter fields `armBinding string`, `armFields map[string]bool`. In
`selectorExpr`, after emitting the base + `.`, if `x.X` is an `*ast.Ident` named
`e.armBinding` and `e.armFields[x.Sel.Name]` is set, emit `exported(name)`.

### 3. Dispatch wiring (emit.go)
- `matchStmt`: in the default branch, if `enumOf(e.info, q) != nil` call
  `enumMatch(m, posStmt, "")`; else keep the descriptive error.
- `returnStmt`: if `len(s.Results)==1` and `Results[0]` is an `*ast.MatchExpr`
  whose qualifier names a resolved enum, call `enumMatch(m, posReturn, "")`.
- `stmt` `*ast.DeclStmt`: if the decl is a `var` GenDecl with a single
  `ValueSpec` of one name, one type, and a single `*ast.MatchExpr` value over a
  resolved enum, emit `var name T` then `enumMatch(m, posVar, name)`; else fall
  through to `e.decl`.

### 4. New corpus fixture
Add a value-position-match example (`.goal` + `.go.expected`) under
`testdata/`, regenerate `corpus/manifest.json` via
`go run ./cmd/corpus-gen -root .`. Update the corpus count test if it pins a
total (US-002/009 assert specific counts — check and adjust if needed).

### 5. Tests (internal/backend/backend_test.go)
`TestASTEngineEnumMatchBehavioralTier`: drive the four 02-match cases plus the
new value-position case through `backend.Transpile` + `corpus.RunCompile`
(temp-module build+vet); assert no failures.

## Ordering / dependencies
1. Add the `enumMatch` helper + `matchPos` + field-export plumbing.
2. Wire the three dispatch sites.
3. Add the corpus fixture + regenerate manifest + fix count assertions.
4. Add the behavioral test; run verifyCommands.

## Risks
- The US-002/009 corpus-count tests assert exact transpile/check counts; adding
  a transpile case bumps the count and those assertions must be updated in the
  same change. Mitigation: grep for the count literals and update.
- `var name T = match` parses to a DeclStmt; confirm the AST shape
  (GenDecl/ValueSpec) before wiring.

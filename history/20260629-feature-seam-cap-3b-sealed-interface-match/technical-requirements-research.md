# Technical Requirements / Research — SEAM-CAP-3b

## Four gaps to close (verified against current code)

1. **Implementor registry** (sema). Today `sema.Info.Sealed` is `map[string]bool`
   (internal/sema/sema.go:97-98) — sealedness only, no implementors. Add a real
   registry mapping each sealed interface to its concrete implementor types,
   built from `implements` clauses (`type T struct implements I {...}`, carried
   on `ast.StructType.Implements`, resolved in resolve.go `resolveTypeDecl`).

   DECISION: keep `Sealed map[string]bool` (sealedness) UNCHANGED and add a new
   `SealedImpls map[string][]string` (interface name -> implementor concrete
   type names). Rationale: `implements` clauses also target ORDINARY (non-sealed)
   interfaces, which feature-07 (CheckImplements) verifies by short-circuiting on
   `info.Sealed[iface]`. Conflating sealedness with "has implementors" into one
   `map[string][]string` keyed by presence would make ordinary-interface
   `implements` clauses look sealed and SKIP their feature-07 verification —
   a corpus-breaking regression. Two distinct facts => two fields. SealedImpls is
   the implementor registry; consulted only for keys that are also in Sealed.
   Unioned across files in Merge (same-package multi-file support).

2. **Parser** (goal_match.go). `parsePattern` handles `_` (RestPattern) and
   `Enum.Variant` (VariantPattern, starts with IDENT). Add a `TypePattern` node
   for a type-pattern arm. Trigger: a pattern starting with `*` (pointer type) is
   a type pattern `*T`, optionally with a binding `*T(x)`. Bare-ident patterns
   stay VariantPattern (backward compatible).

3. **Backend lowering** (emit.go / lower.go). matchStmt/matchValue route on
   `matchQualifier` (Result/Option/enum) and emit `Enum_Variant` case labels. Add
   a SEPARATE `sealedMatch` path (distinct from enumMatch) that emits a Go
   type-switch with concrete `case *T:` labels for a sealed scrutinee. Dispatch:
   a match whose arms are TypePatterns is a sealed match. Bind via
   `switch <guard> := <subject>.(type)` when an arm uses its binding (mirrors
   enumMatch). Handle posStmt / posReturn / posVar.

4. **Sema exhaustiveness** (check.go). `checkOneMatch` covers enums. Add: a match
   whose arms are TypePatterns resolves its sealed interface (via the implementor
   registry reverse-lookup of the first arm's concrete type), then requires every
   registered implementor covered or a `_` rest-arm — else a `non-exhaustive`
   Error, consistent with the §9 switch-coexistence rule.

## Mirror discipline (CAP-3a learning)

selfhost/backend/emit.goal and lower.goal are real Go compiled by the self-host
port gate; any emitter helper added on one side MUST be mirrored on the other or
the port test fails immediately. Mirror ALL changes in internal/ and selfhost/
(ast, parser, sema, backend).

## Gotchas

- enum/sealed zero value is nil.
- value-position match lowers only as `:=` / `var x T = match` / `return match`.
- Commit with `-c commit.gpgsign=false` (1Password signing fails non-interactively).

## Acceptance proof

A SAME-package regression fixture: a `match` over a same-package sealed interface
transpiles, behaves identically to `switch x := n.(type)`, and a non-exhaustive
match is a sema error.

## Gates

task check, task build, task fixpoint all green; corpus behavioral unchanged.

## Research findings (codebase investigation)

Confidence: High (read the live code paths directly).

- Enum match lowering: internal/backend/emit.go `enumMatch` (line ~2056) already
  emits a Go type-switch (`switch [guard :=] subject.(type)`) with `Enum_Variant`
  case labels, a `_`-rest -> `default`, else a panicking default. The sealedMatch
  path is structurally the SAME but case labels are the concrete implementor types
  rendered directly (`case *Ident:`), and no field-set/binding-export machinery is
  needed (the binding is the narrowed value itself, not an enum payload).
- Dispatch is by `matchQualifier(m)` reading the first VariantPattern's Enum. A
  TypePattern arm yields no qualifier, so a new `isSealedMatch(m)` (any arm is a
  TypePattern) cleanly selects the new path at every dispatch site (matchStmt,
  returnStmt, tryVarMatch/matchValue, tryAssignMatch).
- Pattern parsing: goal_match.go `parsePattern` -> add a `*`-leading TypePattern
  branch. AST: new `ast.TypePattern{Type ast.Expr; Binding *Ident; ...}` mirroring
  VariantPattern, with walk.go + ast_test coverage.
- Exhaustiveness: check.go `checkOneMatch` -> branch on TypePattern arms; resolve
  sealed iface by reverse-lookup in the implementor registry; require full cover
  or `_`.
- Mirror: selfhost/{ast/goal_expr.goal, ast/walk.goal, parser/goal_match.goal,
  sema/{sema,resolve,check}.goal, backend/{lower,emit}.goal} mirror the internal
  Go line-for-line (the port gate compiles them as real Go).

Open questions: none blocking. Cross-package implementor propagation is explicitly
out of scope (CAP-3c).

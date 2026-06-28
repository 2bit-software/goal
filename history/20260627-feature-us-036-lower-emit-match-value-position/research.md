# Research — US-036

This is an in-repo lowering task; the authoritative reference is the existing
codebase, not external sources.

## Reference implementation (known-good): `internal/pass/match.go`

The legacy splice engine lowers an enum `match` to a Go type-switch:

```
switch [v := ]scrut.(type) {
case Enum_Variant:
    <arm body, wrapped per position>
...
default:
    panic("unreachable: non-exhaustive <Enum> (compiler invariant violated)")
}
```

Key behaviors mirrored:
- The `v :=` guard is emitted only when some non-rest arm references its payload
  binding (`usesBinding`); otherwise a bare `switch scrut.(type)`.
- Payload binding `a` is renamed to the guard var; field accesses are exported
  (`a.since` -> `v.Since`) using the variant's field set.
- An explicit `_` rest arm becomes a real `default:`; otherwise the default is a
  panic with the non-exhaustive message.
- Position wrapping (`armStatement`): statement -> `<body>`, return ->
  `return <body>`, var -> `<name> = <body>`. For `var name T = match`, a
  `var name T` declaration is emitted before the switch.

## AST-backend machinery already present (US-034/035)

- `matchQualifier(m)` returns the first variant-pattern arm's enum name.
- `enumOf(info, name)` -> `*sema.Enum`; `sema.Enum.FieldSet[variant]` gives the
  exported-field set.
- `e.renames` + `armBodyRenamed` rename a binding within one arm body.
- `e.gensym("v")` mints a scope-aware temp (no `__goal_` prefix; US-035).
- `usesIdent(body, name)` reports whether an arm references its binding.
- `e.armBody` emits a block/stmt or an expression body.

## Plan-relevant decisions

- Enum match dispatches from three sites: `matchStmt` default (statement),
  `returnStmt` (`return match`), and the `*ast.DeclStmt` case in `stmt`
  (`var name T = match`).
- `selectorExpr` gains a field-export branch keyed on the current arm's binding
  + field set (new emitter fields `armBinding`/`armFields`).
- A new value-position-match corpus fixture is added under `testdata/` and the
  manifest regenerated.

## Confidence

High — the encoding is an exact analogue of a shipped, build+vet-clean legacy
pass, reimplemented over resolved sema facts (the same approach US-033/034/035
used successfully).

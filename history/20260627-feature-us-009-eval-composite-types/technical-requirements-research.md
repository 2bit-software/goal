# Technical Requirements / Research — US-009

## Existing seam

- Expression evaluation lives in `internal/interp/eval.go` (`evalExpr`); the
  statement dispatcher is `interp.go` `execStmt`. Both are switch statements on
  AST node type that extend cleanly.
- The runtime value model (`value.go`) already has `KindStruct`
  (`StructValue{TypeID, Fields map[string]Value}`, a pointer), `KindSlice`
  (`[]Value`), and `KindMap` (`MapValue{Entries map[string]Value}`, a pointer).
  v1 maps are STRING-KEYED (per the value-model note); non-string keys are
  deferred. `StructVal`, `SliceVal`, `MapVal` constructors exist.

## Plan

Expressions (eval.go):
- `*ast.CompositeLit`: dispatch on `Type` — `*ast.ArrayType` (slice),
  `*ast.MapType` (map), `*ast.Ident` naming a struct type (struct, keyed
  `field: value` elements).
- `*ast.SelectorExpr`: struct field access (`x.field`). Reads `Struct.Fields`.
  Package-qualified / variant selectors stay later stories.
- `*ast.IndexExpr`: slice index (int) and map index (string key).

Statements (interp.go):
- `*ast.RangeStmt`: range over a slice (key = int index, value = elem) and a
  map (key = string, value); honors `:=` / `=` and the blank identifier.
- Index / field assignment targets in `bindTargets`: `s[i] = v`, `m[k] = v`,
  `x.field = v`. Slices share a backing array, MapValue and StructValue are
  pointers, so element/field mutation through a looked-up value is visible.

## No new dependencies

internal/interp must stay free of go/types, internal/backend, internal/typecheck
(US-022 gate). Only ast/token/sema plus stdlib.

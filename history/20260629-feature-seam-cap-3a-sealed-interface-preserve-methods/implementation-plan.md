# Implementation Plan

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/backend/sealed_methods_test.go` | Regression test: sealed interface declaring methods keeps them + marker in emitted Go; implementor providing the methods builds and a call through the interface value runs (proves preservation + callability). |

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/emit.go` | Rewrite `sealedInterfaceDecl` to emit declared methods (when present) + the marker method; keep compact form when empty. Extract the interface-method-emit loop from `interfaceType` into a shared `interfaceMethod(*ast.Field)` helper and reuse in both. |
| `selfhost/backend/emit.goal` | Mirror the same `sealedInterfaceDecl` + `interfaceMethod` change line-for-line. |

`internal/backend/lower.go` `genSealedInterface` and `selfhost/backend/lower.goal`
remain unchanged — still used for the empty-body compact form.

## Package Structure

```
internal/backend/
  emit.go                 (modified)
  lower.go                (unchanged — genSealedInterface kept for empty case)
  sealed_methods_test.go  (new)
selfhost/backend/
  emit.goal               (modified — mirror)
  lower.goal              (unchanged — mirror)
```

## Dependency Graph

1. `interfaceMethod` helper extraction in emit.go (refactor, behavior-preserving).
2. `sealedInterfaceDecl` rewrite using the helper (depends on 1).
3. selfhost/backend/emit.goal mirror of 1 + 2.
4. Regression test (depends on 2).

## Interface Contracts

```go
// New shared helper (internal/backend/emit.go) — renders one interface method
// element (a named method or an embedded interface) without a trailing newline.
func (e *emitter) interfaceMethod(m *ast.Field)

// Rewritten:
func (e *emitter) sealedInterfaceDecl(d *ast.SealedInterfaceDecl)
//   d.Methods == nil || len == 0  -> e.p(genSealedInterface(d.Name.Name))   // compact, unchanged
//   else -> "type Name interface {\n" + each interfaceMethod + "\n" + "isName()\n" + "}"
```

The selfhost mirror uses the identical signatures (Go superset = valid goal).

## Integration Points

- `emit.go interfaceType` (L521-540) loses its inline method loop, calling
  `interfaceMethod` instead — no behavior change for ordinary interfaces.
- `emit.go decl` dispatch already routes `*ast.SealedInterfaceDecl` to
  `sealedInterfaceDecl`; no dispatch change.
- `genSealedInterface` (lower.go:297) still called for the empty case.

## Testing Strategy

- `internal/backend/sealed_methods_test.go`:
  - Shape test: `backend.Transpile` an inline goal source with
    `sealed interface Node { Pos() Position; End() Position }` plus an implementor;
    assert emitted Go contains `Pos() Position`, `End() Position`, and `isNode()`.
  - Behavioral test: transpile a package whose implementor provides the methods and
    a func calls a method through a `Node` value; write to a temp `module goal`,
    `go build`/`go test`; success proves methods are preserved and callable (if the
    interface dropped the methods, the call through the interface would not compile).
    Skipped under `-short` (spawns the go toolchain), matching existing tests.
- Whole-tree gates: `task check`, `task build`, `task fixpoint` (watch fixpoint —
  emit/lower change; empty-body compact form kept to protect byte-identity).

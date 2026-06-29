# Implementation Plan — nested sealed-interface hierarchies

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/backend/nested_sealed_test.go` | Regression: 2-level hierarchy transpiles, `go build`/`go test` against reference type-switch, match over A and B both include T. |
| `internal/sema/nested_sealed_test.go` | Exhaustiveness enforced at both A and B levels; cascade registers T under both. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/sema/resolve.go` | Add `cascadeSealedImpls()`; call at end of `Resolve` and end of `ResolvePackage`. |
| `selfhost/sema/resolve.goal` | Line-for-line mirror of the same. |
| `internal/backend/lower.go` | Add `sealedEmbeds(info, iface) []string` (transitive embedded sealed ifaces). |
| `selfhost/backend/lower.goal` | Mirror. |
| `internal/backend/emit.go` | `implementsMarker`: emit cascade markers for embedded sealed ifaces. |
| `selfhost/backend/emit.goal` | Mirror. |
| `DECISIONS.md` | Record the cascade design choice (vs multi-clause). |
| `prd.json` | SEAM-CAP-3d passes:true (after green). |
| `progress.txt` | Append entry. |

## Dependency Graph

1. sema cascade (`cascadeSealedImpls` in resolve.go/.goal) — registry side.
2. backend `sealedEmbeds` helper (lower.go/.goal) — depends on info.EmbeddedIfaces.
3. backend `implementsMarker` cascade emission (emit.go/.goal) — depends on 2.
4. Regression tests — depend on 1-3.

## Interface Contracts

```go
// sema/resolve.go
func (info *Info) cascadeSealedImpls()      // propagate B's impls to embedded sealed A (transitive)
// called at tail of Resolve(f) and ResolvePackage(files)

// backend/lower.go
func sealedEmbeds(info *sema.Info, iface string) []string  // transitively-embedded sealed ifaces, source order, deduped
```

## Integration Points

- `sema.Resolve` / `sema.ResolvePackage` (resolve.go) — append cascade call.
- `backend.emitter.implementsMarker` (emit.go ~279) — after emitting the sealed
  marker for `iface`, loop `sealedEmbeds` and emit a marker per embedded sealed.
- No change to foreign.go: goalForeignDecls projects the already-cascaded
  SealedImpls/Sealed.

## Testing Strategy

- internal/backend/nested_sealed_test.go: const goal source with sealed A, sealed
  B embeds A, concrete T implements B (+ a second impl per level for real
  exhaustiveness). Assert emitted Go contains both `func (T) isA()` and
  `func (T) isB()`; transpile + temp `module goal` + `go test` reference switch
  over both A and B. Mirrors internal/backend/sealed_match_test.go.
- internal/sema/nested_sealed_test.go: Resolve the source; assert SealedImpls[A]
  and SealedImpls[B] both contain `*T`; assert a non-exhaustive match over A and
  over B each yields `non-exhaustive-match`.
- selfhost mirrors are exercised by the existing port gate + `task fixpoint`.

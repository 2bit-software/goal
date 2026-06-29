# Implementation Plan

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/sema/testdata/sealedshape/shape.goal` | Sibling `.goal` package defining a sealed interface `Node` with implementors `*Lit`, `*Neg` — fixture for the sema cross-package enrichment test. |
| `internal/sema/crosspkg_sealed_test.go` | Sema test: EnrichForeign projects the sibling sealed interface into info.Sealed/SealedImpls; CheckExhaustive resolves cross-package (clean when exhaustive; `non-exhaustive-match` Error when missing an implementor). |
| `internal/backend/testdata/goalsealed/shape/shape.goal` | Sibling `.goal` package defining sealed `Node`/`Lit`/`Neg` — defining half of the behavioral fixture. |
| `internal/backend/testdata/goalsealed/use/use.goal` | Consumer `.goal` package: a cross-package `match` over `shape.Node`. |
| `internal/backend/crosspkg_sealed_test.go` | Behavioral test mirroring crosspkg_goal_enum_test.go: transpile both packages per-package, build into a temp `module goal`, run against a reference `switch x := n.(type)`. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/sema/foreign.go` | Add 6th return `sealed map[string][]string` to `foreignDecls` + `goalForeignDecls`; project EXPORTED sealed interfaces in `goalForeignDecls` (qualify iface + implementors); `.go` path returns nil sealed; `EnrichForeign` merges into `info.Sealed`/`info.SealedImpls`. |
| `selfhost/sema/foreign.goal` | Line-for-line mirror of the foreign.go changes (real Go compiled by the port gate). |

## Package Structure

```
internal/sema/foreign.go                 (modified)
internal/sema/crosspkg_sealed_test.go    (new)
internal/sema/testdata/sealedshape/shape.goal (new)
internal/backend/crosspkg_sealed_test.go (new)
internal/backend/testdata/goalsealed/shape/shape.goal (new)
internal/backend/testdata/goalsealed/use/use.goal     (new)
selfhost/sema/foreign.goal               (modified, mirror)
```

## Dependency Graph

1. `internal/sema/foreign.go` — the capability (no new deps; uses existing ResolvePackage/qualifyForeignType).
2. `selfhost/sema/foreign.goal` — mirror of 1 (required for fixpoint/port gate).
3. Fixtures (shape.goal/use.goal) — pure data.
4. Tests (sema + backend) — depend on 1 and 3.

## Interface Contracts

```go
// foreignDecls/goalForeignDecls gain a 6th result:
func foreignDecls(dir, requestedAlias string) (
    structs map[string][]Field, funcs, methods map[string]FuncSig,
    enums map[string]*Enum, sealed map[string][]string, err error)

func goalForeignDecls(dir, requestedAlias string, goalFiles []string) (
    structs map[string][]Field, funcs, methods map[string]FuncSig,
    enums map[string]*Enum, sealed map[string][]string, err error)

// EnrichForeign merge:
//   info.Sealed[iface] = true
//   info.SealedImpls[iface] = impls   // iface = "alias.Iface", impls = ["*alias.T", ...]
```

`goalForeignDecls` projection: for each `iface` in `ResolvePackage(files).Sealed` where
`isExportedName(iface)`, set `sealed[alias+"."+iface]` to the `info.SealedImpls[iface]`
list each requalified via `qualifyForeignType(impl, alias)` (`*Lit` → `*shape.Lit`).

## Integration Points

- Single call site each: `EnrichForeign` line ~95 in both files (`structs, funcs, methods,
  enums, sealed, err := foreignDecls(...)`).
- Consumed downstream by `checkOneSealedMatch` → `sealedInterfaceOf` (already reads
  info.Sealed + info.SealedImpls; no change needed there).
- Backend `sealedMatch` lowering is pattern-driven (no registry use) — unchanged.

## Testing Strategy

- Sema unit test with an injected resolver pointing at the `.goal` fixture dir; assert
  registry contents + CheckExhaustive outcomes (clean / Error).
- Backend behavioral test: per-package transpile of both `.goal` packages, temp module
  build + `go test` against a reference type-switch (pattern from crosspkg_goal_enum_test.go).
- Gates: `task check`, `task build`, `task fixpoint`; corpus behavioral tier unchanged
  (fixtures are additive).

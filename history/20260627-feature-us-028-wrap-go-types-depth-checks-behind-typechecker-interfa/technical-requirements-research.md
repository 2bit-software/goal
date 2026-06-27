# Technical Requirements / Research — US-028

## Current state

- `internal/typecheck` exposes `Load(pkg *project.Package) (*Package, error)`
  (transpile via `pipeline.TranspilePackage`, parse lowered Go, run go/types)
  plus three depth checks operating on `*Package`:
  `CheckImplements`, `CheckMustUse`, `CheckNoZeroValue`.
- `cmd/goal/main.go`'s `runDepthChecks(pkg)` calls `Load` then the three checks
  and returns `[]typecheck.Diagnostic`.

## Approach (per REWRITE-ARCHITECTURE.md §3.2 / decision 4)

- Define `TypeChecker interface { Check(pkg *project.Package) ([]Diagnostic, error) }`
  — the whole "transpile → go/types → depth checks" run is one implementation;
  a native goal checker is a future implementation, both producing `[]Diagnostic`,
  so the caller never changes when the implementation flips.
- Provide `GoTypesChecker` (the existing crutch) implementing it by delegating to
  `Load` + the three depth-check functions.
- Route `cmd/goal`'s `runDepthChecks` through the interface so no caller reaches
  for the concrete `Load`/`Check*` functions directly.
- Test in `internal/typecheck` drives the depth checks through the interface
  value over the existing harness fixtures and asserts they still pass.

## Out of scope

- Any native (non-go/types) type checker (US-028 note: "no native goal type
  checker until the runtime forces it").
- Changes to the depth-check logic itself.

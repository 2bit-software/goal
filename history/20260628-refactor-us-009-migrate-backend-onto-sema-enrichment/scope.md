# Scope — US-009 Migrate backend package onto sema enrichment

## What is being refactored and why

`internal/backend/package.go` (`TranspilePackage`) is the AST engine's
package-mode driver. It already resolves cross-file facts via
`sema.ResolvePackage(files)`, but its `enrichForeign` helper still reaches into
`internal/analyze` (`analyze.BuildPackage` + `analyze.EnrichForeign`) to load
imported Go packages' struct field sets, then hand-copies those facts into the
sema `Info`. This is the last live consumer of `internal/analyze` in the backend.
US-010 cannot delete `internal/analyze` until this dependency is gone.

## What the old code looks like

```go
info := sema.ResolvePackage(files)
enrichForeign(info, srcs, pkg.Dir)
...
func enrichForeign(info *sema.Info, srcs []string, dir string) {
    t := analyze.BuildPackage(srcs)
    analyze.EnrichForeign(t, srcs, dir, nil)
    // copy t.Structs (skip in-package) and t.FromRegistry into info
}
```

- Imports `goal/internal/analyze`.
- Re-parses sources via analyze and copies only Structs + FromRegistry.

## What the new code should look like

- `enrichForeign` calls `sema.EnrichForeign(info, imports, dir, nil)` directly,
  where `imports` is the aggregate of every parsed file's `ast.File.Imports`.
- `sema.EnrichForeign` already keys foreign structs as `alias.Type` (no collision
  with bare in-package names), and also populates `info.FuncSignatures` and
  `info.ForeignMethods` — strictly more facts than the old copy.
- In-package `FromRegistry` is already populated by `sema.ResolvePackage`, so the
  old analyze FromRegistry copy is redundant and dropped (analyze.EnrichForeign
  never added foreign FromRegistry entries).
- Remove the `goal/internal/analyze` import from package.go.

## What must NOT change

- `TranspilePackage`'s signature and output (`pipeline.PackageOutput`).
- The shared prelude / option-boxing emission behavior.
- Behavioral conformance: the corpus package-mode behavioral tier must pass
  unchanged.

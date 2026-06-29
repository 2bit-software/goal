# Implementation Plan — US-004 Bootstrap and fixpoint harness

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `selfhost/main.goal` | Goal-source `package main`; a thin `goal build --emit` equivalent. Parses `build --emit=<dir> <path>`, runs `project.Discover` + `backend.TranspilePackage`, writes `out.Files`/`out.Tests` under `filepath.Join(emitDir, pkg.Dir)`. |

### Modified Files
| File | Changes |
|------|---------|
| `Taskfile.yml` | Add `bootstrap` (stage-0 -> goal-c-1 -> goal-c-2) and `fixpoint` (diff-r the two emissions) targets. |
| `.gitignore` | Ignore `_bootstrap/` and `bin/goal-c-*` (build outputs). |

## Package Structure

```
selfhost/
  main.goal          # package main — goal-written goalc/goal-build skeleton
_bootstrap/          # (gitignored) stage + fixpoint emit dirs: s1, s2, fa, fb
bin/                 # (gitignored) goal, goalc, goal-c-1, goal-c-2
```

`selfhost/` holds only `.goal` files, invisible to the Go toolchain's `./...`.

## Dependency Graph

1. `selfhost/main.goal` — depends only on existing `internal/*` Go packages.
2. `Taskfile.yml` targets — depend on `selfhost/main.goal` and existing `build`.
3. `.gitignore` — independent.

## Implementation Notes

- `main.goal` must be plain Go-superset goal: no reserved identifiers
  (`match`/`enum`/`assert`); use `strings.HasPrefix`/`strings.TrimPrefix` for
  flag parsing; octal `0o755`/`0o644` are valid Go literals.
- Emit layout must match `cmd/goal`'s `emitFiles` exactly so goal-c-1 and
  goal-c-2 emit identically: `filepath.Join(emitDir, pkg.Dir)` then each
  `GoFile.Name`.
- Bootstrap target sequence:
  1. `task build` (stage 0: ./bin/goal)
  2. `rm -rf _bootstrap`
  3. `./bin/goal build --emit=_bootstrap/s1 ./selfhost`
  4. `go build -o bin/goal-c-1 ./_bootstrap/s1/selfhost`
  5. `./bin/goal-c-1 build --emit=_bootstrap/s2 ./selfhost`
  6. `go build -o bin/goal-c-2 ./_bootstrap/s2/selfhost`
- Fixpoint target: `deps: [bootstrap]`, then
  `./bin/goal-c-1 build --emit=_bootstrap/fa ./selfhost`,
  `./bin/goal-c-2 build --emit=_bootstrap/fb ./selfhost`,
  `diff -r _bootstrap/fa _bootstrap/fb`, echo FIXPOINT OK.

## Verification

- `task bootstrap` exits 0; `bin/goal-c-1` and `bin/goal-c-2` exist.
- `task fixpoint` exits 0 (diff finds no differences).
- `task check` and `task build` green (artifacts under `_bootstrap/` are
  ignored by `go ... ./...`).

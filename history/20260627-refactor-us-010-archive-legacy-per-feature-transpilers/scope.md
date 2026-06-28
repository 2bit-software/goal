# Scope — US-010 Archive legacy per-feature transpilers

## What is being refactored and why

The early-phase, per-feature standalone transpilers live under
`features/NN/transpiler/` (one per feature: 01-enums, 02-match, 03-result,
04-option, 05-question-prop, 06-error-e, 07-implements, 08-no-zero-value,
10-assert, 11-doctests, 12-derive-convert). Each is its own Go module
(`module goal/features/NN/transpiler` with its own `go.mod`), so they are NOT
part of the root `goal` module and never participate in `go build ./...` /
`go test ./...` / `go vet ./...`.

They have been superseded by the unified `internal/pipeline` front-end and the
in-progress AST rewrite. Leaving them in `features/*/transpiler` implies they are
live implementations. They should be archived under `attic/`.

## What the old code looks like

- `features/NN/transpiler/{go.mod, main.go, transpile.go, transpile_test.go}`
  plus a compiled `transpiler` binary in some dirs.
- Separate modules; not referenced by the root module, the corpus manifest
  (`corpus/manifest.json` references only `features/NN/examples/*`), or any
  `.go` source in the root module.
- The only textual references are prose in `TODO.md` describing each feature's
  deliverables ("`transpiler/` + `examples/`") — documentation, not code paths.

## What the new code should look like

- The 11 `features/NN/transpiler/` directories relocated to
  `attic/features/NN/transpiler/`, preserving their contents (history-via-git
  move with `git mv`).
- No `features/*/transpiler` path remains (glob check returns nothing).

## What must NOT change

- The root `goal` module build/test/vet behavior — it already excludes these
  modules, so moving them is inert to the root module.
- `features/NN/examples/` golden inputs and `.go.expected` outputs (the live
  corpus) — untouched.
- `features/_cut/09-pure/transpiler` is already an attic-like cut area at a
  deeper path; it is not matched by `features/*/transpiler` and is out of scope.

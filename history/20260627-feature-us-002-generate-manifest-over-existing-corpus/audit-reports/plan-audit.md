# Plan Audit — US-002

## Findings

- MINOR: `go run ../../cmd/corpus-gen` in a `go:generate` directive is relative
  to the package dir; acceptable since generation is a maintainer task. Not a
  blocker.
- MINOR: ID-collision risk if two paths slugify identically. Mitigated by basing
  IDs on the full relative path (directory + base), which is unique per file.

No CRITICAL or MAJOR findings. Every spec FR traces to a plan element:
FR-1/FR-2 → `generate.go` walk rules; FR-3 → read-only walk; FR-4 → `corpus-gen`
+ `corpus/manifest.json`; FR-5 → sorted ordering + determinism test.

## Assumptions

- The generator command lives at `cmd/corpus-gen` alongside existing commands
  (`cmd/goal`, `cmd/goalc`).
- `corpus/manifest.json` is committed to the tree (not gitignored) so runners in
  later stories can load it directly.

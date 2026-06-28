# Plan — US-025 Add AST snapshot tests

## Components

1. `internal/ast/dump.go` (new) — deterministic s-expression renderer.
   - Signature: `func Sexpr(n Node) string`
   - Reflection-driven over exported struct fields. Rules:
     - `token.Pos` fields skipped (structure, not offsets).
     - `token.Kind` fields rendered via `.String()`.
     - `FuncMod` / `ChanDir` rendered by name.
     - Zero/empty fields omitted; slices rendered as bracketed lists.
     - Pointers/interfaces recurse; nil renders `nil`.
   - Stdlib only: `reflect`, `strings`, `fmt`, `goal/internal/token`.

2. `internal/parser/snapshot_test.go` (new) — snapshot suite.
   - `var updateSnapshots = flag.Bool("update-snapshots", false, ...)`.
   - Table of one representative input per goal construct (see research doc).
   - Per case: read input, `ParseFile`, `ast.Sexpr`, compare to golden under
     `internal/parser/testdata/snapshots/<name>.sexpr`; with the flag, write
     the golden instead. Also asserts `ParseFile` returns no error and that
     `Sexpr` is deterministic (two renders equal).

3. `internal/parser/testdata/snapshots/*.sexpr` (new, generated then committed) —
   one golden per representative input.

## Ordering

1. dump.go → 2. snapshot_test.go → 3. generate goldens with
`-update-snapshots` → 4. re-run without the flag to confirm green.

## Traceability

- AC1 (render to deterministic textual form + one snapshot per construct) →
  components 1 and 3.
- AC2 (test compares rendered AST to checked-in snapshot) → component 2.

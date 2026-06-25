There are **two** binaries. Use the umbrella `goal` for whole programs; use `goalc` for
single-file inspection.

`goal` discovers `.goal` packages under a path, transpiles them with the unified front-end, and
drives the Go toolchain over the result. By default it is **ephemeral**: generated Go is mapped
into the module via `go build -overlay`, so **nothing is written to your source tree**, and
module/stdlib imports still resolve. `--emit` instead writes the generated `.go` (and
`_test.go`) beside each `.goal`, which you need before `go test` can run doctests.

**Discovery rules:** a "package" = all `.goal` files in one directory (subdirectories are
separate packages); every `.goal` file must start with a `package <name>` clause; two different
package names in one directory is an error; `goal run` requires **exactly one** `package main`.

**Requirements:** Go **1.26+**, and the program must live in a Go **module** (a `go.mod` at or
above the path) so imports resolve. goal itself is zero-dependency.

**`goalc`** reads one `.goal` file, runs the checker, then the lowering pipeline, and writes the
transpiled Go to **stdout** — use it to see exactly what a snippet lowers to (`go run
./cmd/goalc file.goal`, `-test` for the doctest sidecar, `-nocheck` to skip the checker, `-` for
stdin). It checks the lexical stage only (no package context); for real correctness feedback use
`goal check` on a package.

**The iteration loop to run:** `goal check` first (fastest — located correctness feedback before
any Go compile), then `goal run` (transpile + compile + run; Go errors map back to `.goal` lines
via `//line` directives), then, if you have doctests, `goal build --emit` followed by
`go test -count=1 ./...`.

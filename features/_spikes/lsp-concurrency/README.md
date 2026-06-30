# LSP concurrency spike (US-014)

A de-risking probe for the LSP port (US-015..US-017). It proves that **goal
source** can express the concurrency the LSP server needs — goroutines and a
`sync.Mutex` — through the real goal toolchain.

The goal backend already *emits* `go`/`chan`/`select` constructs, but until this
probe no goal source had ever *used* concurrency (the compiler itself is
single-threaded). This closes that gap end to end.

## What it does

`main.goal` launches 8 goroutines (`go func(){...}()`); each performs 1000
`sync.Mutex`-guarded increments of a shared counter, synchronized with a
`sync.WaitGroup`, then prints the total. The result is correct only if both the
goroutine launch and the mutex lower and run correctly.

## Run it

```sh
go build -o bin/goal ./cmd/goal
./bin/goal run features/_spikes/lsp-concurrency   # prints 8000 (deterministic)
```

## Why it's parked here

`features/_spikes/` is underscore-prefixed, so the Go toolchain
(`go build/test/vet ./...`) ignores it, and the nested `go.mod` isolates it from
`module goal`. It lives outside `internal/compiler`, so `task generate`,
`verify-generated`, `bootstrap`, and `fixpoint` (all scoped to
`./internal/compiler`) never touch it.

See DECISIONS.md ("US-014 — lsp concurrency spike") for the recorded finding.

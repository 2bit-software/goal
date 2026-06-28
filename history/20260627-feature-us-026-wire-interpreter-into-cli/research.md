# Research — US-026 Wire interpreter into the CLI

Internal codebase wiring; no external research required. Findings from the tree:

## How to run a program under the interpreter (existing seams)

```go
src, _ := os.ReadFile(file)
f, err := parser.ParseFile(string(src))   // internal/parser
info := sema.Resolve(f)                    // internal/sema
ip := interp.New(f, info, interp.WithStdout(out))
err = ip.Run()                             // gates on native sema, runs func main
```

- `interp.New` defaults to `cap.GrantAll()` + `os.Stdout`; `WithStdout(out)`
  redirects the program's stdout effect to the command's writer so a test can
  capture it. (internal/interp/interp.go)
- `Run()` already gates on native sema (`gate()`), returns `ErrNoMain` when no
  `func main`, and surfaces eval/panic failures as a returned error — so the CLI
  just propagates the error (main exits 1 on non-nil).

## CLI integration point

- `cmd/goal/main.go` `run()` dispatches `build|run|check` after `parseFlags`.
  The old `--engine` flag was removed in US-043 (progress.txt). Reintroduce
  `--engine` parsing for `run`, with `interp` selecting the interpreter path and
  the default keeping the current transpile-and-`go run` behavior.
- Interpreter path is FILE-based: the path arg is a single `.goal` file (interp
  operates on one parsed `*ast.File`). Non-interp `run` keeps its
  directory/package semantics.

## Test convention (cmd/goal/main_test.go)

- Tests call `run([]string{...}, &out, &errOut)` directly and assert the error
  and `out.String()`. `goalModule(t, files)` writes a temp module. Stdlib
  `testing` only; no testify.

## Confidence: High — all seams exist and are unit-tested; this is wiring only.

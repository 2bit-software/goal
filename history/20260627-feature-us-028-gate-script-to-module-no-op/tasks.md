# Implementation Tasks — US-028

## Task 1: Add the script-to-module no-op gate test
**Status**: completed
**Files**: `internal/corpus/script_module_gate_test.go` (new)
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4 (all)
**Verify**: `go build ./... && go vet ./... && go test ./internal/corpus/ -run TestScriptToModuleNoOp -count=1`

### Instructions
Create `internal/corpus/script_module_gate_test.go` in external package
`corpus_test`.

1. Define a sample goscript program constant exercising a genuine goal construct:
   an `enum Color { Red Green }`, a `name(c Color) string` function returning a
   value-position `match c { Color.Red => "red"; Color.Green => "green" }`, and a
   `func main() { fmt.Println(name(Color.Green)) }` with `import "fmt"` and
   `package main`. Expected output: `green`.

2. Helper `runUnderInterp(t, src) string`:
   - `parser.ParseFile(src)` -> file (t.Fatal on error).
   - `sema.Resolve(file)` -> info.
   - `var buf bytes.Buffer; ip := interp.New(file, info, interp.WithStdout(&buf))`.
   - `ip.Run()` (t.Fatal on error).
   - return `buf.String()`.

3. Helper `runAsGoModule(t, src) string`:
   - `backend.Transpile(src)` -> out (t.Fatal on error).
   - `os.MkdirTemp` a dir, `defer os.RemoveAll`.
   - Write `go.mod` (`module goalscript\n\ngo 1.26\n`) and `case.go` (out.Go).
   - `cmd := exec.Command("go", "run", "."); cmd.Dir = tmp`.
   - Capture stdout (`cmd.Output()` or a captured `cmd.Stdout` buffer); on error
     t.Fatalf including combined output and out.Go.
   - return the captured stdout string.

4. `TestScriptToModuleNoOp`:
   - `interpOut := strings.TrimSpace(runUnderInterp(t, sample))`.
   - `moduleOut := strings.TrimSpace(runAsGoModule(t, sample))`.
   - Assert both non-empty and equal; assert each equals `"green"`.
   - On mismatch `t.Fatalf("script-to-module upgrade is not a no-op: interp=%q module=%q", interpOut, moduleOut)`.

Use stdlib `testing`, `bytes`, `os`, `os/exec`, `path/filepath`, `strings` only.
No testify. Imports: `goal/internal/{interp,parser,sema,backend}`.

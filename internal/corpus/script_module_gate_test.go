package corpus_test

// US-028 — Gate: script-to-module no-op.
//
// This gate proves the headline promise of goscript: a program written for the
// tree-walking interpreter "graduates" into a compiled Go+ module WITHOUT being
// rewritten. The SAME, unchanged source is (1) run under the interpreter and
// (2) transpiled via the existing AST backend and built+run as a Go module, and
// the two observable outputs must be identical — the upgrade is a no-op.
//
// The sample program exercises a GENUINE goal construct (an enum plus a
// value-position `match`, one of the two non-Go runtime mechanics), so the
// no-op upgrade is meaningful rather than a trivial pure-Go round trip. The test
// lives in the external `corpus_test` package so it can import both
// internal/interp and internal/backend (neither imports internal/corpus, so
// there is no import cycle).

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/backend"
	"goal/internal/interp"
	"goal/internal/parser"
	"goal/internal/sema"
)

// scriptModuleSample is the sample goscript program. It declares an enum and a
// value-position match, then prints the resolved variant name from func main.
// Both the interpreter run and the transpiled-then-built binary must emit
// "green".
const scriptModuleSample = `package main

import "fmt"

enum Color {
	Red
	Green
}

func name(c Color) string {
	return match c {
		Color.Red => "red"
		Color.Green => "green"
	}
}

func main() {
	fmt.Println(name(Color.Green))
}
`

// TestScriptToModuleNoOp asserts the interpreter run and the transpiled-then-built
// binary of the SAME unchanged source produce identical output (FR-1..FR-4).
func TestScriptToModuleNoOp(t *testing.T) {
	interpOut := strings.TrimSpace(runUnderInterp(t, scriptModuleSample))
	moduleOut := strings.TrimSpace(runAsGoModule(t, scriptModuleSample))

	if interpOut == "" {
		t.Fatalf("interpreter produced no output")
	}
	if moduleOut == "" {
		t.Fatalf("transpiled module produced no output")
	}
	if interpOut != moduleOut {
		t.Fatalf("script-to-module upgrade is NOT a no-op: interp=%q module=%q", interpOut, moduleOut)
	}
	if interpOut != "green" {
		t.Fatalf("unexpected output %q, want %q (both paths)", interpOut, "green")
	}
}

// runUnderInterp parses, sema-resolves, and runs src under the goscript
// interpreter, returning the program's captured standard output. The stdout
// effect is routed through the capability sink via interp.WithStdout (US-023).
func runUnderInterp(t *testing.T, src string) string {
	t.Helper()

	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("interp parse: %v", err)
	}
	info := sema.Resolve(file)

	var buf bytes.Buffer
	ip := interp.New(file, info, interp.WithStdout(&buf))
	if err := ip.Run(); err != nil {
		t.Fatalf("interp run: %v", err)
	}
	return buf.String()
}

// runAsGoModule transpiles src via the AST backend, writes the generated Go into
// an isolated temp module, builds+runs it with `go run .`, and returns the
// binary's captured standard output. It mirrors the temp-module + toolchain
// pattern of corpus.RunDoctestExec.
func runAsGoModule(t *testing.T, src string) string {
	t.Helper()

	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}

	tmp, err := os.MkdirTemp("", "goal-script-module-*")
	if err != nil {
		t.Fatalf("temp module: %v", err)
	}
	defer os.RemoveAll(tmp)

	const goMod = "module goalscript\n\ngo 1.26\n"
	files := map[string]string{
		"go.mod":  goMod,
		"case.go": out.Go,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmp
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run failed: %v\n--- generated Go ---\n%s\n--- stderr ---\n%s", err, out.Go, stderr.String())
	}
	return stdout.String()
}

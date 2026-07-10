package backendtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"goal/internal/backend"
)

// TestNativeTrailingErrorSignatureUnchanged asserts that the native trailing-error
// `?` rewrite engages ONLY when the function body actually contains a `?` site: a
// `?`-free error-returning function's signature must lower byte-for-byte as before
// (no injected named returns), so no existing golden churns (US-002 AC #3).
func TestNativeTrailingErrorSignatureUnchanged(t *testing.T) {
	src := `package p

func h() (int, error) {
	return 1, nil
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	if !strings.Contains(out.Go, "func h() (int, error)") {
		t.Errorf("`?`-free error function signature was rewritten; got:\n%s", out.Go)
	}
}

// TestNativeTrailingErrorRuntime proves the AC #1 runtime contract end-to-end: a
// native `(A, B, error)` function hosting `a, b := g()?` compiles and, when g
// fails, returns the zero values of A and B plus g's error; when g succeeds it
// returns the computed values. The transpiled program is built and executed in an
// isolated temp module and its stdout is asserted.
func TestNativeTrailingErrorRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain")
	}
	src := `package main

import "errors"
import "fmt"

func g(fail bool) (int, string, error) {
	if fail {
		return 0, "", errors.New("boom")
	}
	return 7, "ok", nil
}

func f(fail bool) (int, string, error) {
	a, b := g(fail)?
	return a * 2, b + "!", nil
}

func main() {
	a, b, err := f(false)
	fmt.Printf("ok=%d|%s|%v\n", a, b, err)
	a2, b2, err2 := f(true)
	fmt.Printf("fail=%d|%q|%v\n", a2, b2, err2)
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}

	tmp := t.TempDir()
	const goMod = "module goaltest\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(out.Go), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmp
	got, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\noutput:\n%s\ngenerated:\n%s", err, got, out.Go)
	}

	// Success path: computed values (7*2=14, "ok"+"!") and a nil error.
	// Failure path: zero-valued int (0) and string ("") plus g's propagated error.
	wantOK := "ok=14|ok!|<nil>"
	wantFail := `fail=0|""|boom`
	text := string(got)
	if !strings.Contains(text, wantOK) {
		t.Errorf("success path: want %q in output, got:\n%s", wantOK, text)
	}
	if !strings.Contains(text, wantFail) {
		t.Errorf("failure path (zero-fill + propagated error): want %q in output, got:\n%s", wantFail, text)
	}
}

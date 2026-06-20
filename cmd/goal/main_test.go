package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goalModule writes a minimal goal module (go.mod + the given .goal files) into a temp
// dir and returns its path. Files is a map of relative path -> source.
func goalModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, src := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const mainGoal = `package main

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

func TestBuildEphemeralLeavesNoGoFiles(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": mainGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"build", dir}, &out, &errOut); err != nil {
		t.Fatalf("build failed: %v\n%s", err, errOut.String())
	}
	// Default build is ephemeral: no generated .go is left in the source tree.
	if entries, _ := filepath.Glob(filepath.Join(dir, "*.go")); len(entries) != 0 {
		t.Errorf("ephemeral build wrote .go files into the source tree: %v", entries)
	}
}

func TestRunExecutesMain(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": mainGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"run", dir}, &out, &errOut); err != nil {
		t.Fatalf("run failed: %v\n%s", err, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != "green" {
		t.Errorf("program output = %q, want green\nstderr: %s", got, errOut.String())
	}
}

func TestBuildErrorMapsToGoalSource(t *testing.T) {
	// A plain Go type error in a passed-through body, on line 4 of the file.
	const bad = "package main\n\nfunc f() int {\n\tvar x int = \"nope\"\n\treturn x\n}\n"
	dir := goalModule(t, map[string]string{"bad.goal": bad})

	var out, errOut bytes.Buffer
	err := run([]string{"build", dir}, &out, &errOut)
	if err == nil {
		t.Fatal("expected build to fail on the type error")
	}
	if !strings.Contains(errOut.String(), "bad.goal:4") {
		t.Errorf("error not mapped to bad.goal:4:\n%s", errOut.String())
	}
}

func TestEmitWritesSiblingGo(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": mainGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"build", "--emit", dir}, &out, &errOut); err != nil {
		t.Fatalf("emit build failed: %v\n%s", err, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		t.Errorf("--emit did not write main.go beside the source: %v", err)
	}
}

func TestParseFlags(t *testing.T) {
	emit, emitDir, root, err := parseFlags([]string{"--emit=out", "./pkg/..."})
	if err != nil {
		t.Fatal(err)
	}
	if !emit || emitDir != "out" || root != "./pkg" {
		t.Errorf("parseFlags = (%v, %q, %q), want (true, out, ./pkg)", emit, emitDir, root)
	}
}

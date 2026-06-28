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

// TestBuildWithASTEngine drives `goal build` over a plain-Go (no goal-specific
// constructs) package through the real command path, proving the AST engine (the
// sole, default front-end) is wired into the driver and produces buildable Go
// (FR-5/FR-6).
func TestBuildWithASTEngine(t *testing.T) {
	const plain = "package main\n\nimport \"fmt\"\n\nfunc add(a int, b int) int {\n\treturn a + b\n}\n\nfunc main() {\n\tfmt.Println(add(2, 3))\n}\n"
	dir := goalModule(t, map[string]string{"main.goal": plain})

	var out, errOut bytes.Buffer
	if err := run([]string{"build", dir}, &out, &errOut); err != nil {
		t.Fatalf("build failed: %v\n%s", err, errOut.String())
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

// goal check runs BOTH stages: the depth (go/types) stage catches an elided composite
// literal that omits a field — a feature-08 violation the lexical stage cannot see — and
// its type-backed finding suppresses the lexical stage's misfire on the same construct.
func TestCheckDepthStageCatchesElidedLiteral(t *testing.T) {
	const src = `package demo

type Inner struct {
    a int
    b int
}

func f() []Inner {
    return []Inner{{a: 1}}
}
`
	dir := goalModule(t, map[string]string{"x.goal": src})

	var out, errOut bytes.Buffer
	err := run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the dropped field\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "[elided-missing-field]") {
		t.Errorf("type-backed diagnostic not surfaced:\n%s", errOut.String())
	}
	// Dedup: the lexical stage's misfiring `[missing-field]` for the same construct is
	// suppressed in favor of the type-backed one.
	if strings.Contains(errOut.String(), "[missing-field]") {
		t.Errorf("lexical misfire was not suppressed by the type-backed finding:\n%s", errOut.String())
	}
}

// A program that violates no guarantee passes both stages.
func TestCheckCleanProgramPasses(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": mainGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"check", dir}, &out, &errOut); err != nil {
		t.Fatalf("clean check failed: %v\nstderr: %s", err, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "ok" {
		t.Errorf("want ok, got stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

// A depth-stage transpile failure (here: a single file checked alone, whose enum
// constructor references an enum declared in a sibling) is reported as a concise,
// non-fatal note — not the full "--- generated ---" Go dump, which is reserved for
// `goal build`, the gate that hard-fails on non-transpiling source.
func TestCheckDepthNoteOmitsGeneratedDump(t *testing.T) {
	dir := goalModule(t, map[string]string{
		"types.goal": "package demo\n\nenum Agent {\n\tOneShot { command: string }\n\tMissing\n}\n",
		"use.goal":   "package demo\n\nfunc mk() Agent {\n\treturn Agent.OneShot(command: \"x\")\n}\n",
	})

	var out, errOut bytes.Buffer
	if err := run([]string{"check", filepath.Join(dir, "use.goal")}, &out, &errOut); err != nil {
		t.Fatalf("single-file check should not hard-fail: %v\nstderr: %s", err, errOut.String())
	}
	stderr := errOut.String()
	if !strings.Contains(stderr, "depth stage unavailable") {
		t.Fatalf("expected the depth-stage note, got:\n%s", stderr)
	}
	if strings.Contains(stderr, "--- generated ---") {
		t.Errorf("depth-stage note leaked the generated-Go dump:\n%s", stderr)
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

func TestParseFlagsUnknownFlag(t *testing.T) {
	// The legacy --engine flag was removed in US-043; it is now an unknown flag.
	if _, _, _, err := parseFlags([]string{"--engine=splice"}); err == nil {
		t.Error("--engine should be rejected as an unknown flag")
	}
}

// interpMainGoal is a self-contained single-file goal program (it calls a goal
// function over a sum type and prints the result) used to exercise the
// interpreter run path end to end.
const interpMainGoal = `package main

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

// TestRunInterpEngineExecutesMain drives `goal run --engine=interp <file>` through
// the real command path: the program is parsed, sema-resolved, and interpreted
// in-process (no Go toolchain), func main runs, and its stdout reaches the
// command's out writer (FR-1/FR-2/FR-3).
func TestRunInterpEngineExecutesMain(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.goal")
	if err := os.WriteFile(file, []byte(interpMainGoal), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := run([]string{"run", "--engine=interp", file}, &out, &errOut); err != nil {
		t.Fatalf("interp run failed: %v\n%s", err, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != "green" {
		t.Errorf("program output = %q, want green\nstderr: %s", got, errOut.String())
	}
}

// TestRunInterpUnknownEngineRejected asserts an unrecognized --engine value is a
// descriptive error rather than a silent fallback to a different back-end.
func TestRunInterpUnknownEngineRejected(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"run", "--engine=bogus", "."}, &out, &errOut); err == nil {
		t.Error("unknown --engine value should be rejected")
	}
}

// TestRunInterpNoMain asserts a program with no func main run under the
// interpreter exits non-zero with an error (FR-5: loud failure, never a silent
// success).
func TestRunInterpNoMain(t *testing.T) {
	const noMain = "package main\n\nfunc helper() int {\n\treturn 1\n}\n"
	dir := t.TempDir()
	file := filepath.Join(dir, "main.goal")
	if err := os.WriteFile(file, []byte(noMain), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := run([]string{"run", "--engine=interp", file}, &out, &errOut); err == nil {
		t.Error("interp run of a program with no func main should fail")
	}
}

package backend_test

import (
	"go/format"
	"os"
	"strings"
	"testing"

	"goal/internal/backend"
	"goal/internal/corpus"
)

// repoRoot is the path from internal/backend (cwd during the test) to the repo
// root, where the corpus behavioral tier builds its temp modules from.
const repoRoot = "../.."

// TestInterfacesExist pins the seam at compile time: GoFormatter satisfies
// Formatter (AC1's Formatter interface) and the package exposes the Backend
// interface that Transpile drives.
func TestInterfacesExist(t *testing.T) {
	var _ backend.Formatter = backend.GoFormatter{}
	// Backend existence is exercised end-to-end by Transpile (parse -> Emit ->
	// format); a nil typed assertion keeps the type referenced without a concrete
	// exported implementation.
	var _ backend.Backend
}

// TestGoFormatterFormats asserts the Go formatter normalizes unformatted Go and
// is idempotent on its own output.
func TestGoFormatterFormats(t *testing.T) {
	const messy = "package p\nfunc  F( )  int {return  1+2}\n"
	got, err := backend.GoFormatter{}.Format([]byte(messy))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	again, err := backend.GoFormatter{}.Format(got)
	if err != nil {
		t.Fatalf("Format (idempotent): %v", err)
	}
	if string(got) != string(again) {
		t.Fatalf("Format not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
	}
	if !strings.Contains(string(got), "func F() int") {
		t.Fatalf("formatted output missing expected signature:\n%s", got)
	}
}

// TestASTEngineTranspilesPlainGo runs the no-goal-constructs fixture through the
// new engine and asserts the output is valid, gofmt-parseable Go (AC1's
// engine path).
func TestASTEngineTranspilesPlainGo(t *testing.T) {
	src := readFixture(t)
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if out.Go == "" {
		t.Fatal("Transpile produced empty Go output")
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
}

// TestASTEngineBehavioralTier is AC2: a goal file using no goal-specific
// constructs transpiles through the new engine and the output compiles + vets
// via the corpus behavioral tier (temp-module go build + go vet). It reuses
// corpus.RunCompile through the corpus.Transpiler seam, judging the engine by
// behavior rather than Go spelling.
func TestASTEngineBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	c := corpus.Case{
		ID:    "plain-no-goal-constructs",
		Kind:  corpus.KindTranspile,
		Mode:  corpus.ModeFile,
		Input: "internal/backend/testdata/plain.goal",
	}
	if err := corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile)); err != nil {
		t.Fatalf("behavioral tier failed for the AST engine: %v", err)
	}
}

func readFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/plain.goal")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return string(b)
}

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

// TestASTEngineEmitsSwitch asserts the emitter handles expression switch
// statements (case + default), the ordinary-Go form the US-026 seed emitter
// omitted. The output must be valid, gofmt-parseable Go that still contains the
// switch/case/default keywords.
func TestASTEngineEmitsSwitch(t *testing.T) {
	const src = `package p

func classify(n int) string {
	switch n {
	case 0:
		return "zero"
	case 1, 2:
		return "small"
	default:
		return "many"
	}
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
	for _, kw := range []string{"switch", "case", "default"} {
		if !strings.Contains(out.Go, kw) {
			t.Fatalf("expected emitted Go to contain %q, got:\n%s", kw, out.Go)
		}
	}
}

// TestASTEngineBehavioralTierFull is the AC-2 witness over the full ordinary-Go
// subset: a goal file exercising switch, struct/map/slice composites, defer, a
// multi-return func, and type/const/var declarations transpiles through the new
// engine and the generated Go builds + vets cleanly via the corpus behavioral
// tier (temp-module go build + go vet).
func TestASTEngineBehavioralTierFull(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	c := corpus.Case{
		ID:    "plain-full-go-subset",
		Kind:  corpus.KindTranspile,
		Mode:  corpus.ModeFile,
		Input: "internal/backend/testdata/plain_full.goal",
	}
	if err := corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile)); err != nil {
		t.Fatalf("behavioral tier failed for the full Go subset: %v", err)
	}
}

// enumsImplementsCases lists the 01-enums and 07-implements transpile cases the
// US-033 lowering must carry through the new backend. They are addressed by their
// repo-relative input path (the corpus behavioral tier reads the source itself).
var enumsImplementsCases = []string{
	"features/01-enums/examples/status.goal",
	"features/01-enums/examples/traffic.goal",
	"features/01-enums/examples/shape.goal",
	"features/01-enums/examples/nested.goal",
	"features/07-implements/examples/value_recv.goal",
	"features/07-implements/examples/pointer_recv.goal",
	"features/07-implements/examples/qualified_iface.goal",
}

// TestASTEngineEnumsImplementsBehavioralTier is US-033 AC2: every 01-enums and
// 07-implements transpile case passes the behavioral tier (temp-module go build +
// go vet) through the new AST backend, proving the §8.1 enum encoding and the
// §8.5 implements marker/assertion lowering produce build+vet-clean Go.
func TestASTEngineEnumsImplementsBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	if len(enumsImplementsCases) == 0 {
		t.Fatal("no enums/implements cases to run")
	}
	for _, input := range enumsImplementsCases {
		t.Run(input, func(t *testing.T) {
			c := corpus.Case{
				ID:    input,
				Kind:  corpus.KindTranspile,
				Mode:  corpus.ModeFile,
				Input: input,
			}
			if err := corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile)); err != nil {
				t.Fatalf("behavioral tier failed: %v", err)
			}
		})
	}
}

// TestASTEngineEnumEncoding pins the §8.1 sum encoding (AC1) on a representative
// enum: the marker interface, per-variant structs, per-variant marker methods,
// and the construction encoding for both a data-less and a payload variant.
func TestASTEngineEnumEncoding(t *testing.T) {
	src := mustRead(t, "../../features/01-enums/examples/status.goal")
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	for _, want := range []string{
		"type Status interface{ isStatus() }",
		"type Status_Pending struct{}",
		"func (Status_Pending) isStatus()",
		"Status(Status_Pending{})",
		"Status(Status_Active{Since: now()})",
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("enum encoding missing %q in:\n%s", want, out.Go)
		}
	}
}

// TestASTEngineImplementsMarkers pins the §8.5 lowering (AC1): a sealed interface
// yields a marker method, an ordinary value-receiver type a `T{}` assertion, and
// a pointer-receiver type a `(*T)(nil)` assertion.
func TestASTEngineImplementsMarkers(t *testing.T) {
	cases := []struct {
		file string
		want string
	}{
		{"../../features/01-enums/examples/shape.goal", "func (Circle) isShape() {}"},
		{"../../features/07-implements/examples/value_recv.goal", "var _ Stringer = Point{}"},
		{"../../features/07-implements/examples/pointer_recv.goal", "var _ Resetter = (*Counter)(nil)"},
		{"../../features/07-implements/examples/qualified_iface.goal", "var _ io.Writer = Discard{}"},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			out, err := backend.Transpile(mustRead(t, c.file))
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			if !strings.Contains(out.Go, c.want) {
				t.Fatalf("implements marker missing %q in:\n%s", c.want, out.Go)
			}
		})
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

func readFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/plain.goal")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return string(b)
}

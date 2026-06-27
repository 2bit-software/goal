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

// resultOptionCases lists the 03-result and 04-option transpile cases the US-034
// lowering must carry through the new backend: open-E Result[T, error] -> native
// (T, error) named returns and Option[T] -> *T, including statement-position
// match over each.
var resultOptionCases = []string{
	"features/03-result/examples/result_int.goal",
	"features/03-result/examples/result_match.goal",
	"features/03-result/examples/result_parse.goal",
	"features/03-result/examples/result_passthrough.goal",
	"features/04-option/examples/option_int.goal",
	"features/04-option/examples/option_find.goal",
	"features/04-option/examples/option_exists.goal",
	"features/04-option/examples/option_passthrough.goal",
}

// TestASTEngineResultOptionBehavioralTier is US-034 AC2: every 03-result and
// 04-option transpile case passes the behavioral tier (temp-module go build + go
// vet) through the new AST backend, proving the open-E Result (T, error) and
// Option *T lowering — including statement-position match — produces build +
// vet-clean Go.
func TestASTEngineResultOptionBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	if len(resultOptionCases) == 0 {
		t.Fatal("no result/option cases to run")
	}
	for _, input := range resultOptionCases {
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

// TestASTEngineResultOptionEncoding pins the AC1 lowering shapes: an open-E
// Result function emits a native (T, error) return and the Ok/Err pair, and an
// Option function emits *T with nil / address-of returns.
func TestASTEngineResultOptionEncoding(t *testing.T) {
	cases := []struct {
		file  string
		wants []string
	}{
		{"../../features/03-result/examples/result_int.goal", []string{
			// The user's own `err` (from strconv.Atoi) forces the named error return
			// to a scope-aware `err1`, the witness that gensym avoids source collisions.
			"(ok int, err1 error)", "return n, nil", "return ok,",
		}},
		{"../../features/03-result/examples/result_match.goal", []string{
			"v, err := parse(input)", "if err != nil {",
		}},
		{"../../features/04-option/examples/option_int.goal", []string{
			") *int {", "return nil", "return &some",
		}},
		{"../../features/04-option/examples/option_find.goal", []string{
			"if o := find(id); o != nil {", "u := *o",
		}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			out, err := backend.Transpile(mustRead(t, c.file))
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			for _, want := range c.wants {
				if !strings.Contains(out.Go, want) {
					t.Errorf("missing %q in:\n%s", want, out.Go)
				}
			}
		})
	}
}

// questionPropCases lists the 05-question-prop transpile cases the US-035 lowering
// must carry through the new backend: postfix `?` over open-E Result and Option, in
// the binding (`name := expr?`), discard (`_ := expr?`), and bare (`expr?`) forms.
var questionPropCases = []string{
	"features/05-question-prop/examples/qprop_bare.goal",
	"features/05-question-prop/examples/qprop_discard.goal",
	"features/05-question-prop/examples/qprop_erronly.goal",
	"features/05-question-prop/examples/qprop_result.goal",
	"features/05-question-prop/examples/qprop_option.goal",
}

// TestASTEngineQuestionBehavioralTier is US-035 AC2 (behavioral half): every
// 05-question-prop transpile case passes the behavioral tier (temp-module go build
// + go vet) through the new AST backend, proving the `?` propagation lowering — for
// Result and Option, in every binding form — produces build + vet-clean Go.
func TestASTEngineQuestionBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	if len(questionPropCases) == 0 {
		t.Fatal("no question-prop cases to run")
	}
	for _, input := range questionPropCases {
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

// TestASTEngineQuestionNoMagicPrefix is US-035 AC1/AC2 (gensym half): the generated
// Go for every `?`-bearing case contains no literal `__goal_` substring — the magic
// prefix is retired in favor of scope-aware generated identifiers.
func TestASTEngineQuestionNoMagicPrefix(t *testing.T) {
	for _, input := range questionPropCases {
		t.Run(input, func(t *testing.T) {
			out, err := backend.Transpile(mustRead(t, "../../"+input))
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			if strings.Contains(out.Go, "__goal_") {
				t.Fatalf("generated Go still carries the magic `__goal_` prefix:\n%s", out.Go)
			}
		})
	}
}

// TestASTEngineQuestionScopeAware proves the generated names are scope-aware, not a
// fixed prefix: `parsePositive` already binds its own `err` (from strconv.Atoi), so
// the lowering must mint a non-colliding error return (`err1`) rather than shadow
// or reuse the source `err` — and never emit `__goal_`.
func TestASTEngineQuestionScopeAware(t *testing.T) {
	out, err := backend.Transpile(mustRead(t, "../../features/03-result/examples/result_int.goal"))
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if strings.Contains(out.Go, "__goal_") {
		t.Fatalf("generated Go still carries the magic `__goal_` prefix:\n%s", out.Go)
	}
	if !strings.Contains(out.Go, "(ok int, err1 error)") {
		t.Fatalf("expected a scope-aware error return `err1` avoiding the source `err`, got:\n%s", out.Go)
	}
}

// TestASTEngineQuestionEncoding pins the `?`-propagation shapes (AC1): a Result `?`
// destructures the callee's trailing error and returns the function's (ok, err)
// pair (one value for a plain-error callee, two for a Result callee); an Option `?`
// stores the *T temp, returns nil when nil, and dereferences into the bound name.
func TestASTEngineQuestionEncoding(t *testing.T) {
	cases := []struct {
		file  string
		wants []string
	}{
		{"../../features/05-question-prop/examples/qprop_result.goal", []string{
			"raw, err := readFile(p)", "if err != nil {", "return ok, err", "return cfg, nil",
		}},
		{"../../features/05-question-prop/examples/qprop_bare.goal", []string{
			"if _, err := flush(); err != nil {", "return ok, err",
		}},
		{"../../features/05-question-prop/examples/qprop_erronly.goal", []string{
			"if err := clean(); err != nil {", "return ok, err",
		}},
		{"../../features/05-question-prop/examples/qprop_option.goal", []string{
			"o := find(name)", "if o == nil {", "return nil", "u := *o", "return &p",
		}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			out, err := backend.Transpile(mustRead(t, c.file))
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			for _, want := range c.wants {
				if !strings.Contains(out.Go, want) {
					t.Errorf("missing %q in:\n%s", want, out.Go)
				}
			}
		})
	}
}

// enumMatchCases lists the 02-match transpile cases plus the new
// value-position-match case (shape_value) that the US-036 lowering must carry
// through the new backend: an enum `match` as a statement, with a `_` rest arm,
// in `return` position, and in `var name T = match` position.
var enumMatchCases = []string{
	"features/02-match/examples/status_match.goal",
	"features/02-match/examples/status_rest.goal",
	"features/02-match/examples/status_return.goal",
	"features/02-match/examples/status_var.goal",
	"features/02-match/examples/shape_value.goal",
}

// TestASTEngineEnumMatchBehavioralTier is US-036 AC2: every 02-match case plus the
// new value-position-match case passes the behavioral tier (temp-module go build +
// go vet) through the new AST backend, proving the §8.2 enum-match type-switch
// lowering — statement and value position — produces build+vet-clean Go.
func TestASTEngineEnumMatchBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	if len(enumMatchCases) == 0 {
		t.Fatal("no enum-match cases to run")
	}
	for _, input := range enumMatchCases {
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

// TestASTEngineEnumMatchEncoding pins the §8.2 enum-match lowering shapes: a
// type-switch with a `case <Enum>_<Variant>:` per variant, a guard variable that
// exports a payload field access, a panicking default for an exhaustive match,
// and the value-position wrappers (`return <body>` / `name = <body>`).
func TestASTEngineEnumMatchEncoding(t *testing.T) {
	cases := []struct {
		file string
		want []string
	}{
		{"../../features/02-match/examples/status_match.goal", []string{
			"switch v := s.(type) {",
			"case Status_Pending:",
			"case Status_Active:",
			"render(v.Since)",
			`panic("unreachable: non-exhaustive Status (compiler invariant violated)")`,
		}},
		{"../../features/02-match/examples/status_rest.goal", []string{
			"case Status_Active:",
			"default:",
			"showPlaceholder()",
		}},
		{"../../features/02-match/examples/status_return.goal", []string{
			`return "pending"`,
			"return v.Reason",
		}},
		{"../../features/02-match/examples/status_var.goal", []string{
			"var d string",
			`d = "pending"`,
			"d = v.Reason",
		}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			out, err := backend.Transpile(mustRead(t, c.file))
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			for _, want := range c.want {
				if !strings.Contains(out.Go, want) {
					t.Errorf("missing %q in:\n%s", want, out.Go)
				}
			}
		})
	}
}

// errorEClosedCases lists the 06-error-e transpile cases the US-037 lowering must
// carry through the new backend: a closed-E `match`, a closed-E `?` with a matching
// error type, and a closed-E `?` across error types via a `from func` conversion.
var errorEClosedCases = []string{
	"features/06-error-e/examples/qclosed_match.goal",
	"features/06-error-e/examples/qclosed_prop_same.goal",
	"features/06-error-e/examples/qclosed_prop_from.goal",
}

// TestASTEngineClosedResultBehavioralTier is US-037 AC2: every 06-error-e case
// passes the behavioral tier (temp-module go build + go vet) through the new AST
// backend, proving the closed-E Result lowering — the §8.1 Ok/Err sum prelude and
// constructors, the closed match type-switch, and the closed `?` with From
// conversion — produces build + vet-clean Go.
func TestASTEngineClosedResultBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	if len(errorEClosedCases) == 0 {
		t.Fatal("no error-E closed cases to run")
	}
	for _, input := range errorEClosedCases {
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

// TestASTEngineClosedResultEncoding pins the closed-E lowering shapes (US-037 AC1):
// the generic Ok/Err sum prelude emitted once, the sum constructors carrying the
// argument, the closed match type-switch over Ok[T,E]/Err[T,E], the closed `?`
// type-switch, and the From-conversion invoked in the Err arm across error types.
func TestASTEngineClosedResultEncoding(t *testing.T) {
	cases := []struct {
		file  string
		wants []string
	}{
		{"../../features/06-error-e/examples/qclosed_match.goal", []string{
			"type Result[T, E any] interface{ isResult() }",
			"type Ok[T, E any] struct{ Value T }",
			"return Err[Config, ParseError]{Value: ParseError(ParseError_Empty{})}",
			"return Ok[Config, ParseError]{Value: Config{Raw: s}}",
			"case Ok[Config, ParseError]:",
			"case Err[Config, ParseError]:",
			`panic("unreachable: non-exhaustive Result[Config, ParseError] (compiler invariant violated)")`,
		}},
		{"../../features/06-error-e/examples/qclosed_prop_same.goal", []string{
			"var cfg Config",
			"case Ok[Config, ParseError]:",
			"return Err[Config, ParseError]{Value:",
		}},
		{"../../features/06-error-e/examples/qclosed_prop_from.goal", []string{
			"func toApp(e ParseError) AppError {",
			"return Err[Config, AppError]{Value: toApp(",
		}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			out, err := backend.Transpile(mustRead(t, c.file))
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			for _, want := range c.wants {
				if !strings.Contains(out.Go, want) {
					t.Errorf("missing %q in:\n%s", want, out.Go)
				}
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

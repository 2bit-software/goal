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

// TestASTEngineEmitsGoGrammarForms is the round-trip witness for the ordinary-Go
// expression/statement forms the AST front-end gained: type assertion, type
// switch, variadic call spread, channel send, select, and labeled statements.
// Each must lower to valid, recognizable Go.
func TestASTEngineEmitsGoGrammarForms(t *testing.T) {
	const src = `package p

func sum(xs ...int) int {
	total := 0
	for _, x := range xs {
		total = total + x
	}
	return total
}

func describe(x any) string {
	switch v := x.(type) {
	case int:
		return "int"
	case string, []byte:
		return "stringy"
	default:
		return "other"
	}
}

func run(ch chan int) {
	n := sum([]int{1, 2}...)
	ch <- n
	select {
	case v := <-ch:
		_ = v
	default:
	}
	_ = any(n).(int)
Outer:
	for {
		for {
			break Outer
		}
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
	for _, want := range []string{
		"sum(xs ...int)",      // variadic param
		"sum([]int{1, 2}...)", // variadic call spread
		".(type)",             // type switch guard
		"case int:",           // type-switch case
		"ch <- n",             // channel send
		"select {",            // select
		"case v := <-ch:",     // comm clause (receive)
		").(int)",             // type assertion
		"Outer:",              // labeled statement
		"break Outer",         // labeled branch
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("emitted Go missing %q, got:\n%s", want, out.Go)
		}
	}
}

// TestASTEngineLowersNestedOptionInResult pins Option construction nested in a
// Result.Ok payload (a Result[Option[T], error] return): Option.None must lower to
// nil and Option.Some(x) to the pointer form, not be emitted verbatim (which left
// `Option` undefined in the generated Go).
func TestASTEngineLowersNestedOptionInResult(t *testing.T) {
	const src = `package p

import "os"

func readIfExists(path string) Result[Option[[]byte], error] {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Result.Ok(Option.None)
	}
	if err != nil {
		return Result.Err(err)
	}
	return Result.Ok(Option.Some(data))
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
	if strings.Contains(out.Go, "Option.") {
		t.Errorf("nested Option construction left unlowered (Option undefined):\n%s", out.Go)
	}
	for _, want := range []string{"return nil, nil", "return &data, nil"} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("emitted Go missing %q, got:\n%s", want, out.Go)
		}
	}
}

// TestASTEngineLowersOptionInValuePositions proves Option construction lowers to
// its *T pointer encoding wherever an Option value is produced — a var/assign RHS,
// a call argument, a struct-literal field, and slice- and map-literal elements — not
// only at a direct return or Result.Ok payload. Option.None -> nil; Option.Some(v)
// -> &v for an addressable identifier; Option.Some(<non-ident>) -> the goalSome
// boxing helper. No literal `Option.` token may survive (it would leave Option
// undefined in the generated Go).
func TestASTEngineLowersOptionInValuePositions(t *testing.T) {
	const src = `package p

type Box struct {
	opt *int
}

func sink(o *int) {
}

func build(v int) {
	x := Option.Some(v)
	sink(x)
	sink(Option.Some(v))
	sink(Option.None)
	b := Box{opt: Option.Some(v)}
	sink(b.opt)
	xs := []*int{Option.Some(v), Option.None}
	sink(xs[0])
	m := map[string]*int{"a": Option.Some(v), "b": Option.None}
	sink(m["a"])
	boxed := Option.Some(1)
	sink(boxed)
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
	if strings.Contains(out.Go, "Option.") {
		t.Errorf("Option construction left unlowered (Option undefined):\n%s", out.Go)
	}
	// FR-1: var-assignment binds x to the *T pointer form.
	// FR-3: None -> nil, addressable Some -> &v, non-addressable Some -> goalSome box.
	for _, want := range []string{
		"x := &v",                            // var-assignment, addressable
		"sink(&v)",                           // call argument, addressable
		"sink(nil)",                          // call argument, None -> nil
		"opt: &v",                            // struct-field value
		"[]*int{&v, nil}",                    // slice-literal elements
		`map[string]*int{"a": &v, "b": nil}`, // map-literal elements
		"goalSome(1)",                        // non-addressable Some -> boxed temp
		"func goalSome[T any](v T) *T",       // the boxing helper is emitted once
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("emitted Go missing %q, got:\n%s", want, out.Go)
		}
	}
}

// TestASTEngineResolvesErrorOnlyArityByImport proves the `?` arity is resolved
// generally through the file's imports, not only from the curated table: os.Lchown
// is an error-only stdlib function absent from stdlibErrorOnly, yet a bare `?` on
// it must still lower to the single-variable `if err := …` form.
func TestASTEngineResolvesErrorOnlyArityByImport(t *testing.T) {
	// os.Lchown is error-only but deliberately not in the curated stdlibErrorOnly
	// fallback, so a correct lowering here can only come from the import resolver.
	const src = `package p

import "os"

func chown(path string) Result[int, error] {
	os.Lchown(path, 0, 0)?
	return Result.Ok(0)
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
	if !strings.Contains(out.Go, "if err := os.Lchown(") {
		t.Errorf("import-resolved error-only `?` did not lower to a single var, got:\n%s", out.Go)
	}
	if strings.Contains(out.Go, "_, err := os.Lchown(") {
		t.Errorf("over-destructured an import-resolved error-only call:\n%s", out.Go)
	}
}

// TestASTEngineLowersClosedResultMethod pins closed-E Result lowering for a
// method: a method returning Result[T, Enum] must lower its body's Result.Ok/Err
// to the Ok[T, Enum]/Err[T, Enum] sum carriers, just like a plain function. The
// closed-E context was previously set up only via a name lookup that missed
// methods (sema keys them by receiver), leaving `Result.Ok(…)` unlowered.
func TestASTEngineLowersClosedResultMethod(t *testing.T) {
	const src = `package p

enum ProvisionError {
	Denied
	Timeout
}

type Env struct {
	id string
}

type Backend struct{}

func (b Backend) Provision(name string) Result[Env, ProvisionError] {
	return Result.Ok(Env{id: name})
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
	if !strings.Contains(out.Go, "Ok[Env, ProvisionError]{") {
		t.Errorf("method body did not lower Result.Ok to the closed-E carrier, got:\n%s", out.Go)
	}
	if strings.Contains(out.Go, "Result.Ok(") {
		t.Errorf("method body left Result.Ok unlowered (Result used without instantiation):\n%s", out.Go)
	}
}

// TestASTEngineUnwrapsErrorOnlyStdlibCall pins the arity of `?` on an error-only
// standard-library call: os.MkdirAll / os.WriteFile / json.Unmarshal return only
// error, so a bare `?` must lower to `if err := …`, never the two-value
// `if _, err := …` that fails to compile. A `(value, error)` call (os.ReadFile)
// must still bind both.
func TestASTEngineUnwrapsErrorOnlyStdlibCall(t *testing.T) {
	const src = `package p

import (
	"encoding/json"
	"os"
)

func setup(data []byte, dir string) Result[int, error] {
	os.MkdirAll(dir, 0o755)?
	os.WriteFile(dir+"/f", data, 0o644)?
	raw := os.ReadFile(dir + "/f")?
	var out int
	json.Unmarshal(raw, &out)?
	return Result.Ok(out)
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
	for _, want := range []string{
		"if err := os.MkdirAll(",  // error-only -> single var
		"if err := os.WriteFile(", // error-only -> single var
		"if err := json.Unmarshal(",
		"raw, err := os.ReadFile(", // (value, error) -> two vars
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("emitted Go missing %q, got:\n%s", want, out.Go)
		}
	}
	// The over-destructured form for an error-only call must never appear.
	for _, bad := range []string{"_, err := os.MkdirAll(", "_, err := os.WriteFile(", "_, err := json.Unmarshal("} {
		if strings.Contains(out.Go, bad) {
			t.Errorf("emitted Go has over-destructured error-only call %q:\n%s", bad, out.Go)
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

// defaultsAssertCases lists the 08-no-zero-value and 10-assert transpile cases the
// US-038 lowering must carry through the new backend: `...defaults` expansion (a
// complete literal, primitive zeros, and reference/alias zeros with some fields set
// explicitly because their zero is unsafe) and `assert` (bare, printf-message, and a
// mix with internal commas / a `%` in the expr text).
var defaultsAssertCases = []string{
	"features/08-no-zero-value/examples/complete.goal",
	"features/08-no-zero-value/examples/defaults_primitives.goal",
	"features/08-no-zero-value/examples/defaults_refs.goal",
	"features/10-assert/examples/bank.goal",
	"features/10-assert/examples/message.goal",
	"features/10-assert/examples/multiple.goal",
}

// TestASTEngineDefaultsAssertBehavioralTier is US-038 AC2: every 08-no-zero-value
// and 10-assert case passes the behavioral tier (temp-module go build + go vet)
// through the new AST backend, proving the `...defaults` zero expansion and the
// `assert` if-panic lowering (with its injected fmt import) produce build+vet-clean
// Go.
func TestASTEngineDefaultsAssertBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	if len(defaultsAssertCases) == 0 {
		t.Fatal("no defaults/assert cases to run")
	}
	for _, input := range defaultsAssertCases {
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

// TestASTEngineDefaultsAssertEncoding pins the US-038 lowering shapes (AC1): a
// `...defaults` element expands to explicit `name: <zero>` entries for the omitted
// fields (primitive zeros, a named-struct `T{}`, a nil slice, and an alias's `""`),
// a bare `assert` becomes `if !(cond) { panic("assertion failed: cond") }`, and the
// printf-message form appends `fmt.Sprintf(...)` with the fmt import injected.
func TestASTEngineDefaultsAssertEncoding(t *testing.T) {
	cases := []struct {
		file  string
		wants []string
	}{
		{"../../features/08-no-zero-value/examples/defaults_primitives.goal", []string{
			"return User{name: name, role: RoleMember, email: \"\", active: false, logins: 0}",
		}},
		{"../../features/08-no-zero-value/examples/defaults_refs.goal", []string{
			// The expanded literal stays on one line, so gofmt does not column-align.
			"meta: Addr{}, tags: nil, label: \"\", retries: 0",
		}},
		{"../../features/08-no-zero-value/examples/complete.goal", []string{
			// A complete literal has no `...defaults`, so it passes through verbatim.
			"return User{name: \"root\", email: \"root@x\", role: RoleAdmin, admin: true}",
		}},
		{"../../features/10-assert/examples/bank.goal", []string{
			`if !(amount > 0) {`,
			`panic("assertion failed: amount > 0")`,
		}},
		{"../../features/10-assert/examples/message.goal", []string{
			`import "fmt"`,
			`if !(age >= 0) {`,
			`panic("assertion failed: age >= 0: " + fmt.Sprintf("age must be non-negative, got %d", age))`,
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

// TestASTEngineDefaultsUnsafeZeroRejected pins the located-error behavior: a
// `...defaults` cannot fill a field whose zero is unsafe (a nil map here), so the
// emitter fails loudly rather than emitting a silent nil that panics on write.
func TestASTEngineDefaultsUnsafeZeroRejected(t *testing.T) {
	src := "package p\n\ntype Cfg struct {\n\tname string\n\tm    map[string]int\n}\n\nfunc mk() Cfg {\n\treturn Cfg{name: \"x\", ...defaults}\n}\n"
	if _, err := backend.Transpile(src); err == nil {
		t.Fatal("expected an error defaulting a nil-map field, got nil")
	}
}

// deriveCases lists the 12-derive-convert transpile cases the US-039 lowering
// must carry through the new backend: a bodyless total derive with slice
// container recursion (slice), a bodyless fallible derive that threads a leaf
// error (from_storage), and a bodied derive with a verbatim override, a `_` skip,
// and `...derive(src)` registry fill (to_storage).
var deriveCases = []string{
	"features/12-derive-convert/examples/slice.goal",
	"features/12-derive-convert/examples/from_storage.goal",
	"features/12-derive-convert/examples/to_storage.goal",
}

// TestASTEngineDeriveBehavioralTier is US-039 AC: every 12-derive-convert case
// passes the behavioral tier (temp-module go build + go vet) through the new AST
// backend, proving the `derive func` field-by-field expansion (identity, total
// and fallible leaf conversions, slice recursion, overrides, `_` skip, and
// `...derive(src)` fill) and the `from func` strip produce build+vet-clean Go.
func TestASTEngineDeriveBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	if len(deriveCases) == 0 {
		t.Fatal("no derive cases to run")
	}
	for _, input := range deriveCases {
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

// TestASTEngineDeriveEncoding pins the US-039 lowering shapes (FR-1..FR-5): a
// `from func` emits as a plain func (the `from` modifier is stripped), an
// identical-typed field is a direct assignment, a registered total leaf becomes a
// call, a fallible leaf threads the error with an early return, slice recursion
// builds a fresh slice with an indexed loop, and a `_` skip leaves its field
// unassigned (at the `var out` zero).
func TestASTEngineDeriveEncoding(t *testing.T) {
	cases := []struct {
		file   string
		wants  []string
		absent []string
	}{
		{file: "../../features/12-derive-convert/examples/slice.goal", wants: []string{
			"func uuidToString(u UUID) string",             // from-strip (FR-1)
			"out.Name = g.Name",                            // identity (FR-2)
			"out.Members = make([]string, len(g.Members))", // slice recursion (FR-4)
			"out.Members[i] = uuidToString(g.Members[i])",
		}},
		{file: "../../features/12-derive-convert/examples/from_storage.goal", wants: []string{
			"func fromStorage(s StoredEvent) (EventExecution, error)", // fallible sig (FR-3)
			"parseUUID(s.ID)",
			"return out, err", // early-return threading
			"out.Railroad = s.Railroad",
		}},
		{file: "../../features/12-derive-convert/examples/to_storage.goal",
			wants: []string{
				"out.ExternalID = e.ID.String()", // verbatim override (FR-5)
				"out.ID = uuidToString(e.ID)",    // ...derive fill via leaf
				"out.TraceID = ptrToNull(e.TraceID)",
			},
			absent: []string{
				"out.Audit", // `_` skip leaves the field at its zero (FR-5)
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
			for _, no := range c.absent {
				if strings.Contains(out.Go, no) {
					t.Errorf("unexpected %q in:\n%s", no, out.Go)
				}
			}
		})
	}
}

// doctestCases are the four features/11-doctests fixtures: a plain function, a
// multi-doctest function, a mixed file (a doc block with no `>>>` yields no
// test), and an enum file whose doctests construct (nested) enum payloads.
var doctestCases = []string{
	"features/11-doctests/examples/add.goal",
	"features/11-doctests/examples/enum.goal",
	"features/11-doctests/examples/mixed.goal",
	"features/11-doctests/examples/multi.goal",
}

// TestASTEngineDoctestTier is US-040 AC: every 11-doctests case passes the
// doctest tier through the new AST backend. The runner transpiles each case via
// backend.Transpile and asserts the emitted `_test.go` sidecar (Output.Test)
// matches the checked-in golden (gofmt-normalized both sides) — proving the AST
// backend extracts `///` doctests and emits the sidecar lowered through the same
// path as the function bodies (the enum case's variant constructions lower).
func TestASTEngineDoctestTier(t *testing.T) {
	if len(doctestCases) == 0 {
		t.Fatal("no doctest cases to run")
	}
	for _, input := range doctestCases {
		t.Run(input, func(t *testing.T) {
			c := corpus.Case{
				ID:       input,
				Kind:     corpus.KindDoctest,
				Mode:     corpus.ModeFile,
				Input:    input,
				Expected: strings.TrimSuffix(input, ".goal") + ".go.expected",
			}
			if err := corpus.RunDoctest(repoRoot, c, corpus.TranspilerFunc(backend.Transpile)); err != nil {
				t.Fatalf("doctest tier failed: %v", err)
			}
		})
	}
}

// TestASTEngineDoctestExecTier is the behavioral doctest tier for the AST
// backend: each 11-doctests case's generated package plus its emitted doctest
// sidecar must pass `go test` in an isolated temp module — proving the doctests
// are not merely emitted but actually run green through the new engine.
func TestASTEngineDoctestExecTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain per case; skipped under -short")
	}
	for _, input := range doctestCases {
		t.Run(input, func(t *testing.T) {
			c := corpus.Case{
				ID:    input,
				Kind:  corpus.KindDoctest,
				Mode:  corpus.ModeFile,
				Input: input,
			}
			if err := corpus.RunDoctestExec(repoRoot, c, corpus.TranspilerFunc(backend.Transpile)); err != nil {
				t.Fatalf("behavioral doctest tier failed: %v", err)
			}
		})
	}
}

// TestASTEngineDoctestEnumLowering pins that a doctest body lowers goal-specific
// values through the SAME path as a function body: the enum.goal sidecar must
// lower the (nested) variant constructions in its doctest examples to the §8.1
// sum encoding, not leave them as the surface `Enum.Variant(field: …)` syntax.
func TestASTEngineDoctestEnumLowering(t *testing.T) {
	out, err := backend.Transpile(mustRead(t, "../../features/11-doctests/examples/enum.goal"))
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if out.Test == "" {
		t.Fatal("enum.goal produced no doctest sidecar")
	}
	wants := []string{
		"Rejection(Rejection_MountNotGranted{Path: \"/etc\"})",                       // construction lowered
		"Decision(Decision_Admit{})",                                                 // data-less variant lowered
		"Decision(Decision_Reject{Reason: Rejection(Rejection_MountNotGranted{Path:", // nested construction lowered
	}
	for _, w := range wants {
		if !strings.Contains(out.Test, w) {
			t.Errorf("missing %q in sidecar:\n%s", w, out.Test)
		}
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

// TestASTEngineAssignMatchBehavioralTier is the US-041 gap-1 witness: a
// value-position `name := match …` (a short-var assignment, not the explicitly
// typed `var x T = match` US-036 already lowered) transpiles through the AST engine
// and the generated Go builds + vets cleanly. The result type is inferred from the
// arm bodies (here an enum and a string).
func TestASTEngineAssignMatchBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	c := corpus.Case{
		ID:    "match_infer_value",
		Kind:  corpus.KindTranspile,
		Mode:  corpus.ModeFile,
		Input: "testdata/match_infer_value.goal",
	}
	if err := corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile)); err != nil {
		t.Fatalf("behavioral tier failed for value-position `:=` match: %v", err)
	}
}

// TestASTEngineAssignMatchEncoding pins the gap-1 lowering: `name := match` over an
// enum emits a `var name T` declaration (T inferred from the arms) followed by the
// value-position type-switch whose arms assign `name = <body>`.
func TestASTEngineAssignMatchEncoding(t *testing.T) {
	src := mustRead(t, "../../testdata/match_infer_value.goal")
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	for _, want := range []string{
		"var label Label",
		"label = Label(Label_Hot{})",
		"var s string",
		"s = \"red\"",
		"switch c.(type)",
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("assign-match encoding missing %q in:\n%s", want, out.Go)
		}
	}
}

// astPackageCases are the Mode=package fixtures the US-041 gap-3 AST package driver
// must carry: a cross-file demo (an enum and a closed-E Result split across two
// files, needing the merged sema and one shared prelude) and a foreign-derive (a
// `derive func` over an imported Go struct, needing foreign-import resolution).
var astPackageCases = []struct {
	id      string
	dir     string
	name    string
	files   []string
	imports map[string]string
}{
	{
		id:    "cross-file-demo",
		dir:   "testdata/package/cross-file-demo",
		name:  "demo",
		files: []string{"testdata/package/cross-file-demo/math.goal", "testdata/package/cross-file-demo/types.goal"},
	},
	{
		id:      "foreign-derive",
		dir:     "testdata/package/foreign-derive",
		name:    "conv",
		files:   []string{"testdata/package/foreign-derive/conv.goal"},
		imports: map[string]string{"goal/internal/pipeline/testdata/extpkg": "internal/pipeline/testdata/extpkg"},
	},
}

// TestASTEnginePackageBehavioralTier is the US-041 gap-3 witness: each package
// fixture is lowered cross-file through the AST package driver (one merged sema, a
// single shared prelude, foreign imports wired in) and the resulting package builds.
func TestASTEnginePackageBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	for _, pc := range astPackageCases {
		t.Run(pc.id, func(t *testing.T) {
			c := corpus.Case{
				ID:    pc.id,
				Kind:  corpus.KindTranspile,
				Mode:  corpus.ModePackage,
				Input: pc.dir,
				Package: &corpus.PackageSpec{
					Name:    pc.name,
					Files:   pc.files,
					Imports: pc.imports,
				},
			}
			if err := corpus.RunPackage(repoRoot, c, corpus.PackageTranspilerFunc(backend.TranspilePackage)); err != nil {
				t.Fatalf("package behavioral tier failed: %v", err)
			}
		})
	}
}

// valueMatchCases are the US-002 fixtures: a value-position `match` over a Result
// and over an Option, each used both as a `return` result and as a `:=`
// assignment RHS. Before US-002 these reached the generic expr path and failed
// with `unsupported expression *ast.MatchExpr`.
var valueMatchCases = []string{
	"testdata/match_value_result.goal",
	"testdata/match_value_option.goal",
}

// TestASTEngineValueMatchBehavioralTier is US-002 AC: each value-position
// Result/Option match fixture transpiles through the AST engine and the generated
// Go builds + vets cleanly (temp-module behavioral tier).
func TestASTEngineValueMatchBehavioralTier(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns the go toolchain; skipped under -short")
	}
	for _, input := range valueMatchCases {
		t.Run(input, func(t *testing.T) {
			c := corpus.Case{
				ID:    input,
				Kind:  corpus.KindTranspile,
				Mode:  corpus.ModeFile,
				Input: "internal/backend/" + input,
			}
			if err := corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile)); err != nil {
				t.Fatalf("behavioral tier failed for value-position match: %v", err)
			}
		})
	}
}

// TestASTEngineValueMatchEncoding pins the US-002 lowering: a value-position match
// over a Result/Option lowers to the same split as the statement-position form,
// with each arm body wrapped for its position — `return <body>` in return
// position and `name = <body>` in `:=` assignment position — and BOTH arm bodies
// (Ok/Some payload arm and Err/None arm) are emitted. The output parses under
// go/format (valid Go), with no surviving `match` keyword.
func TestASTEngineValueMatchEncoding(t *testing.T) {
	cases := []struct {
		file  string
		wants []string
	}{
		{"testdata/match_value_result.goal", []string{
			// return-position Result match: destructure + each arm `return <body>`,
			// payload bindings (cfg->v, e->err) reachable.
			"v, err := parse(input)",
			"return v.Raw",
			"return err.Error()",
			// assignment-position Result match: `var tag T` + each arm `tag = <body>`.
			"var tag string",
			"tag = \"ok\"",
			"tag = \"err\"",
		}},
		{"testdata/match_value_option.goal", []string{
			// return-position Option match: nil-check + Some payload binding + each
			// arm `return <body>`.
			"if o := find(id); o != nil {",
			"u := *o",
			"return u.Name",
			"return \"none\"",
			// assignment-position Option match: `var tag T` + each arm `tag = <body>`.
			"var tag string",
			"tag = \"found\"",
			"tag = \"missing\"",
		}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			out, err := backend.Transpile(mustRead(t, c.file))
			if err != nil {
				t.Fatalf("Transpile: %v", err)
			}
			if _, ferr := format.Source([]byte(out.Go)); ferr != nil {
				t.Fatalf("emitted Go does not parse under go/format: %v\n%s", ferr, out.Go)
			}
			if strings.Contains(out.Go, "match ") || strings.Contains(out.Go, "unsupported expression") {
				t.Errorf("value-position match did not lower (residual match/unsupported) in:\n%s", out.Go)
			}
			for _, want := range c.wants {
				if !strings.Contains(out.Go, want) {
					t.Errorf("value-match encoding missing %q in:\n%s", want, out.Go)
				}
			}
		})
	}
}

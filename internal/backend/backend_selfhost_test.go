// Self-contained backend tests: the subset of backend_test.go that depends on
// neither the corpus harness nor repo-relative fixtures (no goal/internal/corpus
// import, no repoRoot/../../features/testdata reads). Split out of backend_test.go
// so the self-host behavioral gate (internal/selfhost BuildAndTest, US-001) can
// copy a faithful, fixture-free suite into its throwaway temp module, where only
// the ported packages and stdlib are available. The corpus/fixture-dependent
// tests remain in backend_test.go and run under `task check` against the real tree.
package backend_test

import (
	"go/format"
	"strings"
	"testing"

	"goal/internal/backend"
	"goal/internal/project"
)

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

// TestPackageDoctestPreludeNotRedeclared pins the package-mode invariant the
// single-file emitter already held: the closed-E Result prelude is defined
// exactly once per package. A Result-returning function emits the shared
// goal_prelude.go; a doctest in the same package emits a `_test.go` sidecar that
// compiles into that same package. Before the fix the sidecar inlined a second
// copy of the prelude, so `go vet` failed with `Result/Ok/Err redeclared in this
// block`. The prelude must live only in goal_prelude.go, never in the sidecar.
func TestPackageDoctestPreludeNotRedeclared(t *testing.T) {
	src := `package domain

enum ParseError {
	Empty
}

/// Adds two ints.
/// >>> add(2, 3)
/// 5
func add(a int, b int) int {
	return a + b
}

func parse(s string) Result[int, ParseError] {
	if s == "" {
		return Result.Err(ParseError.Empty)
	}
	return Result.Ok(len(s))
}
`
	pkg := &project.Package{
		Dir:   "domain",
		Name:  "domain",
		Files: []project.File{{Path: "domain/domain.goal", Name: "domain.goal", Src: src}},
	}
	out, err := backend.TranspilePackage(pkg)
	if err != nil {
		t.Fatalf("TranspilePackage: %v", err)
	}

	var sharedPrelude int
	for _, f := range out.Files {
		if f.Name == "goal_prelude.go" {
			sharedPrelude++
		}
	}
	if sharedPrelude != 1 {
		t.Fatalf("want exactly one shared goal_prelude.go, got %d", sharedPrelude)
	}
	if len(out.Tests) == 0 {
		t.Fatal("expected a doctest sidecar, got none")
	}
	for _, ft := range out.Tests {
		if strings.Contains(ft.Go, "type Result[") {
			t.Errorf("doctest sidecar %s redeclares the prelude (`type Result[`); it must defer to goal_prelude.go:\n%s", ft.Name, ft.Go)
		}
	}
}

// TestASTEngineUnwrapsMethodCallResult is the US-003 backend witness: a `?` on a
// method call lowers through the receiver's resolved method signature. A
// value-binding `?` on a Result[T, error] method binds the value and propagates the
// trailing error; a bare `?` on an error-only method lowers to the single-variable
// `if err := recv.M(); err != nil` form, never the two-value over-destructure. Both a
// concrete-struct receiver and an interface-typed receiver are covered.
func TestASTEngineUnwrapsMethodCallResult(t *testing.T) {
	const src = `package p

type Store struct{ n int }

func (s Store) Load() Result[int, error] { return Result.Ok(s.n) }

func (s Store) Save() error { return nil }

sealed interface Backend {
	Fetch() Result[int, error]
	Flush() error
}

func viaStruct(s Store) Result[int, error] {
	v := s.Load()?
	s.Save()?
	return Result.Ok(v)
}

func viaIface(b Backend) Result[int, error] {
	w := b.Fetch()?
	b.Flush()?
	return Result.Ok(w)
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
		"v, err := s.Load()",                // Result method -> two-value bind
		"if err := s.Save(); err != nil {",  // error-only method -> single var
		"w, err := b.Fetch()",               // interface Result method -> two-value bind
		"if err := b.Flush(); err != nil {", // interface error-only method -> single var
	} {
		if !strings.Contains(out.Go, want) {
			t.Errorf("emitted Go missing %q, got:\n%s", want, out.Go)
		}
	}
	for _, bad := range []string{"_, err := s.Save()", "_, err := b.Flush()"} {
		if strings.Contains(out.Go, bad) {
			t.Errorf("emitted Go over-destructured an error-only method call %q:\n%s", bad, out.Go)
		}
	}
}

// TestASTEngineEmitsIotaConstBlock pins the fix for the iota const-block
// miscompile (US-001): bare iota-continuation names in a grouped const must
// emit as separate const specs, not collapse into the preceding spec as a type.
// The old defect transpiled `Green` + `Blue` into a single `Green Blue`
// (name + type) spec, silently dropping Blue's iota value.
func TestASTEngineEmitsIotaConstBlock(t *testing.T) {
	const src = `package p

type Color int

const (
	Red Color = iota
	Green
	Blue
)
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
	// The miscompile collapsed Green and Blue into one `Green Blue` name+type
	// spec. Guard against the regression directly.
	if strings.Contains(out.Go, "Green Blue") {
		t.Fatalf("Green and Blue collapsed into a name+type spec:\n%s", out.Go)
	}
	for _, name := range []string{"Red", "Green", "Blue"} {
		if !strings.Contains(out.Go, name) {
			t.Fatalf("expected emitted Go to declare const %q, got:\n%s", name, out.Go)
		}
	}
}

// TestASTEngineEmitsGenericFuncDecls is the round-trip witness for top-level
// generic function declarations (US-003): a type-parameter list after the name
// must parse without `expected (, found [`, survive emission, and produce valid
// Go (with go/format standing in for `go build` acceptance). Both an
// unconstrained `[T any]` and a constrained `[K comparable, V any]` param list
// are exercised.
func TestASTEngineEmitsGenericFuncDecls(t *testing.T) {
	const src = `package p

func Identity[T any](x T) T {
	return x
}

func Keys[K comparable, V any](m map[K]V) []K {
	out := []K{}
	for k := range m {
		out = append(out, k)
	}
	return out
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}
	if _, err := format.Source([]byte(out.Go)); err != nil {
		t.Fatalf("engine output is not valid Go: %v\n--- output ---\n%s", err, out.Go)
	}
	for _, want := range []string{"Identity[T any]", "Keys[K comparable, V any]"} {
		if !strings.Contains(out.Go, want) {
			t.Fatalf("expected emitted Go to contain %q, got:\n%s", want, out.Go)
		}
	}
}

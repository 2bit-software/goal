package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

// goal check --json emits its diagnostics as a single JSON array of objects with
// stable field names, so an AI agent can consume them without regexing prose. The
// diagnostic case round-trips and the command still exits non-zero (US-009).
func TestCheckJSONReportsDiagnostics(t *testing.T) {
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
	err := run([]string{"check", "--json", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the dropped field\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	if !json.Valid(out.Bytes()) {
		t.Fatalf("stdout is not valid JSON:\n%s", out.String())
	}
	var diags []jsonDiag
	if err := json.Unmarshal(out.Bytes(), &diags); err != nil {
		t.Fatalf("unmarshal diagnostics: %v\n%s", err, out.String())
	}
	if len(diags) == 0 {
		t.Fatalf("expected at least one diagnostic in JSON output, got none:\n%s", out.String())
	}
	found := false
	for _, d := range diags {
		if d.Code == "elided-missing-field" {
			found = true
			if d.Severity != "error" {
				t.Errorf("severity = %q, want error", d.Severity)
			}
			if !strings.HasSuffix(d.File, "x.goal") {
				t.Errorf("file = %q, want a path ending in x.goal", d.File)
			}
			if d.Line == 0 || d.Message == "" {
				t.Errorf("diagnostic missing line/message: %+v", d)
			}
		}
	}
	if !found {
		t.Errorf("expected an [elided-missing-field] diagnostic in JSON:\n%s", out.String())
	}
}

// A clean package emits an empty JSON array to stdout and exits 0 in --json mode.
func TestCheckJSONCleanEmitsEmptyArray(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": mainGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"check", "--json", dir}, &out, &errOut); err != nil {
		t.Fatalf("clean --json check failed: %v\nstderr: %s", err, errOut.String())
	}
	if !json.Valid(out.Bytes()) {
		t.Fatalf("stdout is not valid JSON:\n%s", out.String())
	}
	var diags []jsonDiag
	if err := json.Unmarshal(out.Bytes(), &diags); err != nil {
		t.Fatalf("unmarshal clean output: %v\n%s", err, out.String())
	}
	if len(diags) != 0 {
		t.Errorf("clean package should emit an empty array, got: %+v", diags)
	}
	if strings.Contains(out.String(), "ok") {
		t.Errorf("the human-readable ok line must be suppressed in --json mode:\n%s", out.String())
	}
}

// --json is meaningful only for check; build rejects it as an unknown flag.
func TestBuildRejectsJSONFlag(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": mainGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"build", "--json", dir}, &out, &errOut); err == nil {
		t.Errorf("build --json should be rejected as an unknown flag\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
}

// An if/for header whose init statement is followed by no condition (e.g.
// `if x := 1 { }`) must be a parse error: the init would otherwise be silently
// dropped and the block run unconditionally. `goal check` reports it at a source
// position and names the missing condition, and exits non-zero (US-007).
func TestCheckRejectsMalformedIfForInit(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "if",
			src:  "package demo\n\nfunc f() {\n\tif x := 1 {\n\t}\n}\n",
			want: "missing if condition",
		},
		{
			name: "for",
			src:  "package demo\n\nfunc f() {\n\tfor x := 1 {\n\t}\n}\n",
			want: "missing for condition",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := goalModule(t, map[string]string{"bad.goal": tc.src})

			var out, errOut bytes.Buffer
			// The malformed header is a parse error. Since US-008 it renders on
			// stderr in the unified `file:line:col: error: [syntax] message` format
			// (main prints only the "N checker error(s)" tally to stderr via the
			// returned error). Require a located [syntax] line naming the missing
			// condition and a non-nil (exit 1) result.
			err := run([]string{"check", dir}, &out, &errOut)
			if err == nil {
				t.Fatalf("expected check to fail on the malformed %s header\nstdout: %s\nstderr: %s", tc.name, out.String(), errOut.String())
			}
			stderr := errOut.String()
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("diagnostic does not name the missing condition (want %q):\n%s", tc.want, stderr)
			}
			// The header is on line 4; the diagnostic renders as bad.goal:4:col:.
			if !strings.Contains(stderr, "bad.goal:4:") {
				t.Errorf("diagnostic not located to bad.goal:4:\n%s", stderr)
			}
			if !strings.Contains(stderr, "error: [syntax] ") {
				t.Errorf("diagnostic not rendered in the [syntax] format:\n%s", stderr)
			}
		})
	}
}

// The valid init+condition forms must still parse and check cleanly — the
// rejection above must not catch `if x := f(); cond { }` or the three-clause
// `for i := 0; i < n; i++ { }` (US-007 FR-3).
func TestCheckAcceptsValidIfForInit(t *testing.T) {
	const src = "package demo\n\nfunc f() int {\n\treturn 1\n}\n\nfunc g() int {\n\ttotal := 0\n\tif x := f(); x > 0 {\n\t\ttotal = x\n\t}\n\tfor i := 0; i < 3; i++ {\n\t\ttotal = total + i\n\t}\n\treturn total\n}\n"
	dir := goalModule(t, map[string]string{"ok.goal": src})

	var out, errOut bytes.Buffer
	if err := run([]string{"check", dir}, &out, &errOut); err != nil {
		t.Fatalf("valid init+condition forms should check clean: %v\nstderr: %s", err, errOut.String())
	}
}

// syntaxLine matches the unified one-line diagnostic format for a syntax error:
// `<path>.goal:<line>:<col>: error: [syntax] <message>` (US-008 AC-1).
var syntaxLine = regexp.MustCompile(`^\S+\.goal:\d+:\d+: error: \[syntax\] `)

// A malformed file (unterminated function body) must be reported by BOTH
// `goal check` and `goal build` as one located `[syntax]` line per error, using
// the full path the user passed, with no duplicate lines and a non-zero exit —
// so a single regex captures every error class (US-008).
func TestSyntaxErrorsRenderInDiagnosticFormat(t *testing.T) {
	// Unterminated body: the closing brace is missing, so the parser errors at EOF.
	const bad = "package demo\n\nfunc f() int {\n\treturn 1\n"

	assertSyntaxFormat := func(t *testing.T, verb string) {
		t.Helper()
		dir := goalModule(t, map[string]string{"bad.goal": bad})

		var out, errOut bytes.Buffer
		err := run([]string{verb, dir}, &out, &errOut)
		if err == nil {
			t.Fatalf("%s: expected failure on the malformed file\nstdout: %s\nstderr: %s", verb, out.String(), errOut.String())
		}

		lines := strings.Split(strings.TrimRight(errOut.String(), "\n"), "\n")
		var syntaxLines []string
		for _, ln := range lines {
			if ln == "" {
				continue
			}
			if !syntaxLine.MatchString(ln) {
				t.Errorf("%s: stderr line not in the [syntax] format: %q", verb, ln)
				continue
			}
			syntaxLines = append(syntaxLines, ln)
			// The path must be the full path under the passed dir, not a bare basename.
			if !strings.HasPrefix(ln, dir) {
				t.Errorf("%s: diagnostic path is not the full user path (want prefix %q): %q", verb, dir, ln)
			}
		}
		if len(syntaxLines) == 0 {
			t.Fatalf("%s: no [syntax] diagnostic emitted:\nstderr: %s", verb, errOut.String())
		}
		// Duplicate trailing errors (e.g. repeated EOF) must be deduplicated.
		seen := map[string]bool{}
		for _, ln := range syntaxLines {
			if seen[ln] {
				t.Errorf("%s: duplicate diagnostic line emitted: %q", verb, ln)
			}
			seen[ln] = true
		}
		// The bare `check <dir>:` / `parse:` wrapper must not leak into output.
		if strings.Contains(errOut.String(), "check "+dir+":") || strings.Contains(errOut.String(), ": parse:") {
			t.Errorf("%s: legacy wrapper leaked into output:\n%s", verb, errOut.String())
		}
	}

	t.Run("check", func(t *testing.T) { assertSyntaxFormat(t, "check") })
	t.Run("build", func(t *testing.T) { assertSyntaxFormat(t, "build") })
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
	emit, jsonOut, emitDir, root, err := parseFlags([]string{"--emit=out", "./pkg/..."})
	if err != nil {
		t.Fatal(err)
	}
	if !emit || jsonOut || emitDir != "out" || root != "./pkg" {
		t.Errorf("parseFlags = (%v, %v, %q, %q), want (true, false, out, ./pkg)", emit, jsonOut, emitDir, root)
	}
	_, jsonOut, _, root, err = parseFlags([]string{"--json", "./pkg"})
	if err != nil {
		t.Fatal(err)
	}
	if !jsonOut || root != "./pkg" {
		t.Errorf("parseFlags --json = (json=%v, root=%q), want (true, ./pkg)", jsonOut, root)
	}
}

func TestParseFlagsUnknownFlag(t *testing.T) {
	// The legacy --engine flag was removed in US-043; it is now an unknown flag.
	if _, _, _, _, err := parseFlags([]string{"--engine=splice"}); err == nil {
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

// argEchoGoal prints every program argument it receives, so a run that passes
// arguments through can be observed end to end.
const argEchoGoal = `package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println(strings.Join(os.Args[1:], " "))
}
`

// TestParseRunFlagsCollectsProgramArgs pins the run grammar: goal's own flags
// precede the path, the first positional is the path, and every token after it —
// flags included — is collected verbatim as a program argument rather than
// interpreted by goal.
func TestParseRunFlagsCollectsProgramArgs(t *testing.T) {
	engine, _, _, root, progArgs, err := parseRunFlags([]string{"--engine=ast", "./cmd/hob", "connect", "--dev"})
	if err != nil {
		t.Fatalf("parseRunFlags returned error: %v", err)
	}
	if engine != "ast" {
		t.Errorf("engine = %q, want ast", engine)
	}
	if root != "./cmd/hob" {
		t.Errorf("root = %q, want ./cmd/hob", root)
	}
	if want := []string{"connect", "--dev"}; !slices.Equal(progArgs, want) {
		t.Errorf("progArgs = %v, want %v", progArgs, want)
	}
}

// TestRunPassesProgramArgs drives `goal run <path> <args...>` through the real
// command path and confirms the running program receives the trailing arguments,
// mirroring `go run <pkg> [args...]`.
func TestRunPassesProgramArgs(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": argEchoGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"run", dir, "connect", "--dev"}, &out, &errOut); err != nil {
		t.Fatalf("run failed: %v\n%s", err, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != "connect --dev" {
		t.Errorf("program saw args %q, want %q\nstderr: %s", got, "connect --dev", errOut.String())
	}
}

// TestRunInterpRejectsProgramArgs asserts the interpreter engine refuses program
// arguments by name rather than silently dropping them, since it has no os.Args
// bridge yet.
func TestRunInterpRejectsProgramArgs(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.goal")
	if err := os.WriteFile(file, []byte(interpMainGoal), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := run([]string{"run", "--engine=interp", file, "connect"}, &out, &errOut); err == nil {
		t.Error("interp run with program arguments should be rejected")
	}
}

// TestCheckCorpusOutputUnchanged is the end-to-end proof that rewiring `goal check`
// onto the AST/sema checker (US-004) leaves its corpus output unchanged. It drives a
// real corpus KindCheck case — a non-exhaustive `match` whose missing variant the
// checker must reject — through the full command path and asserts the exact rendered
// finding the checker produced before the rewire: a `[non-exhaustive-match]` Error at
// the `match` keyword that names the uncovered variant, with `goal check` exiting
// non-zero. The fixture's own `// want` marker pins the same guarantee in the corpus.
func TestCheckCorpusOutputUnchanged(t *testing.T) {
	const fixture = "../../testdata/check/02-match/non_exhaustive_stmt.goal"
	src, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read corpus fixture: %v", err)
	}
	dir := goalModule(t, map[string]string{"status.goal": string(src)})

	var out, errOut bytes.Buffer
	err = run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("check should reject the non-exhaustive match\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	stderr := errOut.String()
	if !strings.Contains(stderr, "error: [non-exhaustive-match]") {
		t.Errorf("missing the non-exhaustive-match error finding:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Status.Cancelled") {
		t.Errorf("finding should name the uncovered variant Status.Cancelled:\n%s", stderr)
	}
	// The rendered line is `path:line:col: severity: [code] message`; assert the
	// finding is located in the rewritten file (not at offset 0) so the AST position
	// — not a dropped byte offset — is what reaches the output.
	if !strings.Contains(stderr, "status.goal:") {
		t.Errorf("finding should be located in status.goal:\n%s", stderr)
	}
}

// TestCheckReportsUnterminatedString is US-006: a lex-malformed but still
// parseable file (an unterminated string literal) makes `goal check` exit
// non-zero with a located `[unterminated-string]` diagnostic pointing at the
// offending file, instead of the malformation slipping through to generated Go.
func TestCheckReportsUnterminatedString(t *testing.T) {
	src := "package demo\n\nfunc greet() string {\n\treturn \"hello\n}\n"
	dir := goalModule(t, map[string]string{"bad.goal": src})

	var out, errOut bytes.Buffer
	err := run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("check should reject the unterminated string\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	stderr := errOut.String()
	if !strings.Contains(stderr, "error: [unterminated-string]") {
		t.Errorf("missing the unterminated-string finding:\n%s", stderr)
	}
	// The diagnostic is located on the return line (line 4) of bad.goal.
	if !strings.Contains(stderr, "bad.goal:4:") {
		t.Errorf("finding should be located at bad.goal:4:\n%s", stderr)
	}
}

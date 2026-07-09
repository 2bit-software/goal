package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

// A Go-level build error renders in the same one-line diagnostic format as check and
// syntax errors — `path/file.goal:line:col: error: [go-build] message` — with the Go
// package-clause header suppressed, so one regex captures every error class (US-010).
func TestBuildErrorRendersInDiagnosticFormat(t *testing.T) {
	const bad = "package main\n\nfunc f() int {\n\tvar x int = \"nope\"\n\treturn x\n}\n\nfunc main() { _ = f() }\n"
	dir := goalModule(t, map[string]string{"bad.goal": bad})

	var out, errOut bytes.Buffer
	err := run([]string{"build", dir}, &out, &errOut)
	if err == nil {
		t.Fatal("expected build to fail on the type error")
	}

	// The type error on line 4 renders with the shared format and a [go-build] code. Go
	// reports a line-only position, so the column defaults to 1.
	line := regexp.MustCompile(`(?m)^\S+bad\.goal:4:1: error: \[go-build\] `)
	if !line.MatchString(errOut.String()) {
		t.Errorf("build error not in the [go-build] diagnostic format:\n%s", errOut.String())
	}
	// The Go package-clause header (`# demo`) must be suppressed.
	if regexp.MustCompile(`(?m)^# `).MatchString(errOut.String()) {
		t.Errorf("package-clause header line was not suppressed:\n%s", errOut.String())
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

// multiPkgFiles models a module whose main package imports a sibling internal
// package, both written in goal. Building or emitting the main package alone must
// transpile the imported sibling too (the import closure) — otherwise the Go
// toolchain sees internal/foo as a .goal-only package with no Go source and fails
// with a misleading "package is not in std" / "no required module provides" error.
func multiPkgFiles() map[string]string {
	return map[string]string{
		"cmd/app/main.goal": `package main

import (
	"fmt"
	"demo/internal/foo"
)

func main() { fmt.Println(foo.Hello()) }
`,
		"internal/foo/foo.goal": `package foo

func Hello() string { return "hello from foo" }
`,
	}
}

// TestBuildResolvesSiblingImportClosure proves `goal build ./cmd/app` transpiles
// the imported sibling internal/foo (a .goal-only package outside the named path)
// so the overlay build finds Go source for every dependency.
func TestBuildResolvesSiblingImportClosure(t *testing.T) {
	dir := goalModule(t, multiPkgFiles())

	var out, errOut bytes.Buffer
	if err := run([]string{"build", filepath.Join(dir, "cmd", "app")}, &out, &errOut); err != nil {
		t.Fatalf("build of sub-package failed to resolve its sibling import: %v\nstderr: %s", err, errOut.String())
	}
}

// TestEmitResolvesSiblingImportClosure proves `--emit ./cmd/app` also writes the
// imported sibling internal/foo's Go beside its source, so a separate `go build`
// over the emitted tree finds Go source for the whole dependency closure.
func TestEmitResolvesSiblingImportClosure(t *testing.T) {
	dir := goalModule(t, multiPkgFiles())

	var out, errOut bytes.Buffer
	if err := run([]string{"build", "--emit", filepath.Join(dir, "cmd", "app")}, &out, &errOut); err != nil {
		t.Fatalf("emit build failed: %v\nstderr: %s", err, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "cmd", "app", "main.go")); err != nil {
		t.Errorf("--emit did not write the named package's main.go: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal", "foo", "foo.go")); err != nil {
		t.Errorf("--emit did not transpile the imported sibling internal/foo: %v", err)
	}
}

// goal check runs BOTH stages: the depth (go/types) stage catches an elided composite
// literal that omits a field whose zero is UNSAFE — a safety-only feature-08 violation the
// lexical stage cannot see on an elided literal. Omitting the nil-map field `m` (whose zero
// panics on write) is rejected with the shared `unsafe-zero` code; the safe-zero `a` is
// free to be omitted.
func TestCheckDepthStageCatchesElidedLiteral(t *testing.T) {
	const src = `package demo

type Inner struct {
    a int
    m map[string]int
}

func f() []Inner {
    return []Inner{{a: 1}}
}
`
	dir := goalModule(t, map[string]string{"x.goal": src})

	var out, errOut bytes.Buffer
	err := run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the omitted unsafe-zero field\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "[unsafe-zero]") {
		t.Errorf("type-backed [unsafe-zero] diagnostic not surfaced:\n%s", errOut.String())
	}
}

// An elided literal that omits only SAFE-zero fields (here `b int`) defaults them silently
// and checks clean — the depth stage no longer enforces completeness.
func TestCheckDepthElidedSafeOmissionClean(t *testing.T) {
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
	if err := run([]string{"check", dir}, &out, &errOut); err != nil {
		t.Fatalf("clean check failed on a safe-only elided omission: %v\nstderr: %s", err, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "ok" {
		t.Errorf("want ok, got stdout=%q stderr=%q", out.String(), errOut.String())
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
    m map[string]int
}

func f() []Inner {
    return []Inner{{a: 1}}
}
`
	dir := goalModule(t, map[string]string{"x.goal": src})

	var out, errOut bytes.Buffer
	err := run([]string{"check", "--json", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the omitted unsafe-zero field\nstdout: %s\nstderr: %s", out.String(), errOut.String())
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
		if d.Code == "unsafe-zero" {
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
		t.Errorf("expected an [unsafe-zero] diagnostic in JSON:\n%s", out.String())
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

// A directory (full-package) check has every sibling file, so a depth-stage transpile
// failure is a genuine compile error and must hard-fail — `check` may not report `ok`
// for source that cannot build. The full "--- generated ---" dump stays reserved for
// `goal build`.
func TestCheckDirectoryHardFailsOnTranspileError(t *testing.T) {
	dir := goalModule(t, map[string]string{
		"bad.goal": "package demo\n\n" +
			"enum Value {\n\tStr { s: string }\n\tInt { n: int64 }\n}\n\n" +
			"func bad(v Value) string {\n" +
			"\tx := match v {\n" +
			"\t\tValue.Str(a) => a.s\n" +
			"\t\tValue.Int(a) => int64(a.n)\n" +
			"\t}\n" +
			"\treturn x\n}\n",
	})

	var out, errOut bytes.Buffer
	err := run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("directory check should hard-fail on a non-transpiling package\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	if strings.Contains(out.String(), "ok") {
		t.Errorf("a hard-failing check must not print ok:\nstdout: %s", out.String())
	}
	if !strings.Contains(errOut.String(), "depth-transpile") {
		t.Errorf("expected the depth-transpile error diagnostic, got:\n%s", errOut.String())
	}
	if strings.Contains(errOut.String(), "--- generated ---") {
		t.Errorf("the hard-fail diagnostic leaked the generated-Go dump:\n%s", errOut.String())
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

// TestExitCodeClassification locks the exit-code contract an AI consumer relies on
// (US-012): user-code diagnostics exit 1, invocation/usage mistakes exit 2, and
// goal-internal failures exit 3. The classifier is asserted directly (deterministic,
// including the hard-to-reproduce internal tier) and end-to-end through run().
func TestExitCodeClassification(t *testing.T) {
	// Direct classifier: nil is success; a bare error defaults to user-code (1);
	// the usage/internal tags override to 2/3.
	if got := exitCode(nil); got != 0 {
		t.Errorf("exitCode(nil) = %d, want 0", got)
	}
	if got := exitCode(errors.New("plain")); got != 1 {
		t.Errorf("exitCode(plain) = %d, want 1", got)
	}
	if got := exitCode(usageErr(errors.New("bad flag"))); got != 2 {
		t.Errorf("exitCode(usageErr) = %d, want 2", got)
	}
	if got := exitCode(internalErr(errors.New("ICE"))); got != 3 {
		t.Errorf("exitCode(internalErr) = %d, want 3", got)
	}
	// The tag survives wrapping so classification works through an error chain.
	if got := exitCode(fmt.Errorf("context: %w", internalErr(errors.New("ICE")))); got != 3 {
		t.Errorf("exitCode(wrapped internalErr) = %d, want 3", got)
	}

	// End-to-end through run(): an unknown subcommand and an unknown flag are usage
	// errors (2); a checker diagnostic over the user's goal is user-code (1).
	badUserCode := goalModule(t, map[string]string{"x.goal": `package demo

type Inner struct {
    a int
    m map[string]int
}

func f() []Inner {
    return []Inner{{a: 1}}
}
`})
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"unknown-subcommand", []string{"bogus-subcommand"}, 2},
		{"unknown-flag", []string{"check", "--nope"}, 2},
		{"user-code-diagnostic", []string{"check", badUserCode}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if got := exitCode(run(tc.args, &out, &errOut)); got != tc.want {
				t.Errorf("exitCode(run(%v)) = %d, want %d\nstderr: %s", tc.args, got, tc.want, errOut.String())
			}
		})
	}
}

// TestPanicMapsToStatementLine proves per-statement /*line*/ directives (US-013):
// a `?` propagation on line 13 lowers into several Go lines, yet the panic on the
// NEXT source line (14) must still report at crash.goal:14, not an off-by-N line.
// Before US-013 the decl-level //line alone drifted the position by the number of
// lines the `?` lowering added.
func TestPanicMapsToStatementLine(t *testing.T) {
	const src = `package main

import "fmt"

func mightFail(bad bool) Result[int, error] {
	if bad {
		return Result.Err(fmt.Errorf("bad"))
	}
	return Result.Ok(7)
}

func crash() Result[int, error] {
	x := mightFail(false)?
	panic(fmt.Sprintf("boom %d", x))
}

func main() {
	v, e := crash()
	fmt.Println(v, e)
}
`
	dir := goalModule(t, map[string]string{"crash.goal": src})

	var out, errOut bytes.Buffer
	err := run([]string{"run", dir}, &out, &errOut)
	if err == nil {
		t.Fatal("expected the program to panic and run to fail")
	}
	// The panic traceback must name the true .goal line of the panic statement.
	if !strings.Contains(errOut.String(), "crash.goal:14") {
		t.Errorf("panic not mapped to crash.goal:14 (the panic statement's true line):\n%s", errOut.String())
	}
}

// TestBuildErrorMapsToStatementLine proves per-statement directives carry Go build
// errors to the true .goal line even after a reflowing lowering: the type error is
// on source line 14, one line after a `?` propagation that expands into multiple Go
// lines. Without statement-level /*line*/ directives the error would land off-by-N.
func TestBuildErrorMapsToStatementLine(t *testing.T) {
	const src = `package main

import "fmt"

func mightFail(bad bool) Result[int, error] {
	if bad {
		return Result.Err(fmt.Errorf("bad"))
	}
	return Result.Ok(7)
}

func compute() Result[int, error] {
	x := mightFail(false)?
	var y int = "nope"
	return Result.Ok(x + y)
}

func main() {
	v, e := compute()
	fmt.Println(v, e)
}
`
	dir := goalModule(t, map[string]string{"compute.goal": src})

	var out, errOut bytes.Buffer
	err := run([]string{"build", dir}, &out, &errOut)
	if err == nil {
		t.Fatal("expected build to fail on the type error")
	}
	if !strings.Contains(errOut.String(), "compute.goal:14") {
		t.Errorf("build error not mapped to compute.goal:14 (the type-error statement's true line):\n%s", errOut.String())
	}
}

// docModule is a package (not `main`) carrying a single doctest. wantErr controls
// whether the doctest's expected value is wrong (making `goal test` fail).
const docPassGoal = `package mathx

/// Squares an int.
/// >>> square(3)
/// 9
func square(x int) int {
	return x * x
}
`

const docFailGoal = `package mathx

/// Squares an int.
/// >>> square(3)
/// 10
func square(x int) int {
	return x * x
}
`

// TestTestRunsDoctestsEphemeral proves `goal test` runs a passing doctest to a
// zero exit and leaves no generated .go/_test.go behind in the source tree (AC-1).
func TestTestRunsDoctestsEphemeral(t *testing.T) {
	dir := goalModule(t, map[string]string{"mathx.goal": docPassGoal})

	before, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	if len(before) != 0 {
		t.Fatalf("fixture already has .go files: %v", before)
	}

	var out, errOut bytes.Buffer
	if err := run([]string{"test", dir}, &out, &errOut); err != nil {
		t.Fatalf("goal test failed on a passing doctest: %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}

	// Ephemeral: no generated .go or _test.go is written into the source tree.
	if entries, _ := filepath.Glob(filepath.Join(dir, "*.go")); len(entries) != 0 {
		t.Errorf("goal test wrote .go files into the source tree: %v", entries)
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, "*_test.go")); len(entries) != 0 {
		t.Errorf("goal test wrote _test.go files into the source tree: %v", entries)
	}
}

// TestTestFailingDoctestReportsGoalPosition proves a failing doctest makes
// `goal test` exit non-zero (exit 1) and that its output names the .goal source
// position (US-014 package-mode doctest rendering) (AC-2, FR-2).
func TestTestFailingDoctestReportsGoalPosition(t *testing.T) {
	dir := goalModule(t, map[string]string{"mathx.goal": docFailGoal})

	var out, errOut bytes.Buffer
	err := run([]string{"test", dir}, &out, &errOut)
	if err == nil {
		t.Fatal("expected goal test to fail on a wrong-expected doctest")
	}
	combined := out.String() + errOut.String()
	if !strings.Contains(combined, "mathx.goal:") {
		t.Errorf("doctest failure did not name the .goal position (mathx.goal:<line>):\n%s", combined)
	}
	if !strings.Contains(combined, "doctest square") {
		t.Errorf("doctest failure output missing the failing doctest name:\n%s", combined)
	}
}

// The depth (go/types) stage runs checkers the lexical stage cannot: the
// must-use ones (discarded-result-error, dropped-stored-result) and the
// safety-only feature-08 residual on generic/elided literals (emitting the
// shared unsafe-zero code). The corpus check harness is sema-only, so `goal
// check` via cmd/goal is the only seam that exercises them. Each test below
// asserts the specific bracketed [code] string (not merely a non-zero exit), so
// a run that silently degraded to the "depth stage unavailable" note would fail
// rather than masquerade as a pass (US-028).

// discardedResultErrorGoal binds an open-E Result call's two returns and throws
// the error position away with `_`, which the depth stage flags.
const discardedResultErrorGoal = `package demo

import "errors"

func f(ok bool) Result[int, error] {
    if ok {
        return Result.Ok(1)
    }
    return Result.Err(errors.New("no"))
}

func use() int {
    v, _ := f(true)
    return v
}
`

// discardedResultErrorCleanGoal binds and inspects the error instead of
// discarding it, so the depth stage is silent.
const discardedResultErrorCleanGoal = `package demo

import "errors"

func f(ok bool) Result[int, error] {
    if ok {
        return Result.Ok(1)
    }
    return Result.Err(errors.New("no"))
}

func use() int {
    v, e := f(true)
    if e != nil {
        return 0
    }
    return v
}
`

func TestCheckDepthDiscardedResultError(t *testing.T) {
	dir := goalModule(t, map[string]string{"x.goal": discardedResultErrorGoal})

	var out, errOut bytes.Buffer
	err := run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the discarded Result error\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "[discarded-result-error]") {
		t.Errorf("depth-stage [discarded-result-error] not surfaced:\n%s", errOut.String())
	}
}

func TestCheckDepthDiscardedResultErrorClean(t *testing.T) {
	dir := goalModule(t, map[string]string{"x.goal": discardedResultErrorCleanGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"check", dir}, &out, &errOut); err != nil {
		t.Fatalf("clean check failed: %v\nstderr: %s", err, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "ok" {
		t.Errorf("want ok, got stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

// droppedStoredResultGoal declares an unexported Option-typed field that is
// never read via a selector — a provably dropped must-use value.
const droppedStoredResultGoal = `package demo

type box struct {
    o Option[int]
}

func make() box {
    return box{o: Option.Some(1)}
}
`

// droppedStoredResultCleanGoal reads the field via a selector (match b.o), so
// the value is consulted and the depth stage is silent.
const droppedStoredResultCleanGoal = `package demo

type box struct {
    o Option[int]
}

func make() box {
    return box{o: Option.Some(1)}
}

func read(b box) int {
    return match b.o {
        Option.Some(v) => v
        Option.None => 0
    }
}
`

func TestCheckDepthDroppedStoredResult(t *testing.T) {
	dir := goalModule(t, map[string]string{"x.goal": droppedStoredResultGoal})

	var out, errOut bytes.Buffer
	err := run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the dropped stored Option\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "[dropped-stored-result]") {
		t.Errorf("depth-stage [dropped-stored-result] not surfaced:\n%s", errOut.String())
	}
}

func TestCheckDepthDroppedStoredResultClean(t *testing.T) {
	dir := goalModule(t, map[string]string{"x.goal": droppedStoredResultCleanGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"check", dir}, &out, &errOut); err != nil {
		t.Fatalf("clean check failed: %v\nstderr: %s", err, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "ok" {
		t.Errorf("want ok, got stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

// genericMissingFieldGoal keys a generic literal `Pair[int]{...}` but omits the
// UNSAFE-zero field m (a nil map) — a residual feature-08 safety gap only the
// typed stage can see. The safe-zero field x may be omitted freely.
const genericMissingFieldGoal = `package demo

type Pair[T any] struct {
    x T
    m map[string]int
}

func f() Pair[int] {
    return Pair[int]{x: 1}
}
`

// genericMissingFieldCleanGoal omits only the safe-zero field y of the generic
// literal, so it defaults silently and checks clean.
const genericMissingFieldCleanGoal = `package demo

type Pair[T any] struct {
    x T
    y T
}

func f() Pair[int] {
    return Pair[int]{x: 1}
}
`

func TestCheckDepthGenericMissingField(t *testing.T) {
	dir := goalModule(t, map[string]string{"x.goal": genericMissingFieldGoal})

	var out, errOut bytes.Buffer
	err := run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the omitted unsafe-zero field\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "[unsafe-zero]") {
		t.Errorf("depth-stage [unsafe-zero] not surfaced:\n%s", errOut.String())
	}
}

// A generic literal that omits only a SAFE-zero field checks clean at the depth
// stage — the safety-only rule defaults it silently.
func TestCheckDepthGenericMissingFieldClean(t *testing.T) {
	dir := goalModule(t, map[string]string{"x.goal": genericMissingFieldCleanGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"check", dir}, &out, &errOut); err != nil {
		t.Fatalf("clean check failed: %v\nstderr: %s", err, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "ok" {
		t.Errorf("want ok, got stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

// boolArmMatchGoal exercises a value-position `x := match` whose arms are
// boolean-valued expressions (a comparison, and a logical `&&` combined with a
// `!`). This shape previously needed an explicit `var x bool = match ...`
// annotation; the emitter now infers `bool` from the syntactically boolean arm
// bodies and lowers to `var x bool` + a type switch.
const boolArmMatchGoal = `package main

import "fmt"

enum Color {
    Red
    Green
}

func warm(c Color, a int, b int) bool {
    x := match c {
        Color.Red => a == b
        Color.Green => a < b && !(b > a)
    }
    return x
}

func main() {
    fmt.Println(warm(Color.Green, 1, 2))
}
`

// TestRunValueMatchInfersBoolArms proves an `x := match` over boolean-valued
// arms now builds and runs, printing the arm's evaluated bool.
func TestRunValueMatchInfersBoolArms(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": boolArmMatchGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"run", dir}, &out, &errOut); err != nil {
		t.Fatalf("run failed: %v\nstderr: %s", err, errOut.String())
	}
	// warm(Green, 1, 2) == (1 < 2 && !(2 > 1)) == (true && !true) == false.
	if got := strings.TrimSpace(out.String()); got != "false" {
		t.Errorf("program output = %q, want false\nstderr: %s", got, errOut.String())
	}
}

// arithmeticArmMatchGoal is a value-position `x := match` whose arms are arithmetic
// over typed operands. The result type is not syntactically a literal/enum, but it is
// derivable from the operands' checked types — the go/types probe (Option B) recovers
// it as `int`, so this now infers rather than requiring an explicit annotation.
const arithmeticArmMatchGoal = `package main

import "fmt"

enum Color {
    Red
    Green
}

func total(c Color, a int, b int) int {
    x := match c {
        Color.Red => a + b
        Color.Green => a - b
    }
    return x
}

func main() { fmt.Println(total(Color.Red, 3, 4)) }
`

// TestRunInfersArithmeticArmValueMatch pins that a value-position `x := match` whose
// arms are arithmetic over typed operands now infers its result type through the
// go/types probe (Option B) and builds+runs — the type is derived from the checked
// operands, not guessed.
func TestRunInfersArithmeticArmValueMatch(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": arithmeticArmMatchGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"run", dir}, &out, &errOut); err != nil {
		t.Fatalf("expected the arithmetic-arm value-position match to build and run: %v\nstderr: %s", err, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != "7" {
		t.Errorf("total(Red, 3, 4) = %q, want %q\nstderr: %s", got, "7", errOut.String())
	}
}

// trailingCommentGoal places a trailing `//` comment on both a `?` line and an
// `assert` line — the two statement forms whose lowering reflows one source
// statement into several Go lines. A trailing comment on these lines once broke
// lowering ("generated Go did not parse"); this pins that it builds and runs
// clean so the fix cannot regress.
const trailingCommentGoal = `package main

import "fmt"

func f(s string) Result[string, error] {
	if s == "" {
		return Result.Err(fmt.Errorf("empty"))
	}
	return Result.Ok(s)
}

func g(s string) Result[string, error] {
	v := f(s)? // trailing comment on a ? line
	return Result.Ok(v)
}

func sq(x int) int {
	assert x >= 0, "neg: %d", x // trailing comment on an assert line
	return x * x
}

func main() {
	match g("hi") {
		Result.Ok(v) => fmt.Println(v, sq(3))
		Result.Err(e) => fmt.Println("err", e)
	}
}
`

// TestRunTrailingCommentOnQuestionAndAssert proves a trailing `//` comment on a
// `?` line and an `assert` line lowers to Go that parses, builds, and runs.
func TestRunTrailingCommentOnQuestionAndAssert(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": trailingCommentGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"run", dir}, &out, &errOut); err != nil {
		t.Fatalf("run failed: %v\nstderr: %s", err, errOut.String())
	}
	if got := strings.TrimSpace(out.String()); got != "hi 9" {
		t.Errorf("program output = %q, want %q\nstderr: %s", got, "hi 9", errOut.String())
	}
}

// callArmMatchGoal exercises a value-position `x := match` whose arms are a
// function call and a bare enum constructor, both statically typed `J`. The
// call arm's type is not syntactically a literal/enum, so it needs the emitter
// to resolve the callee's declared single return type to infer `J`; this shape
// previously required an explicit `var x J = match ...` annotation.
const callArmMatchGoal = `package main

import "fmt"

enum J {
	JNull
	JStr { s: string }
}

func fromList(xs []string) J {
	return J.JStr(s: "x")
}

func conv(o Option[[]string]) J {
	x := match o {
		Option.Some(xs) => fromList(xs)
		Option.None     => J.JNull
	}
	return x
}

func main() {
	fmt.Println(conv(Option.Some([]string{"a"})))
	fmt.Println(conv(Option.None))
}
`

// TestRunValueMatchInfersCallArm proves an `x := match` whose arms are a
// function call and an enum constructor (both statically `J`) builds and runs,
// inferring `var x J` from the callee's declared return type rather than
// refusing with the inferable-result-type diagnostic.
func TestRunValueMatchInfersCallArm(t *testing.T) {
	dir := goalModule(t, map[string]string{"main.goal": callArmMatchGoal})

	var out, errOut bytes.Buffer
	if err := run([]string{"run", dir}, &out, &errOut); err != nil {
		t.Fatalf("run failed: %v\nstderr: %s", err, errOut.String())
	}
	// Some(["a"]) => fromList(["a"]) => J.JStr{s:"x"} prints {x}; None => J.JNull prints {}.
	if got := strings.TrimSpace(out.String()); got != "{x}\n{}" {
		t.Errorf("program output = %q, want %q\nstderr: %s", got, "{x}\n{}", errOut.String())
	}
}

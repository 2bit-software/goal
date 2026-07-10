package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"goal/internal/ast"
	"goal/internal/fix"
)

const plainGoFix = `package app

func load(p string) ([]byte, error) {
	f, err := read(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}
`

// `goal fix` on a single file prints the idiomatic rewrite to stdout and writes nothing.
func TestFixStdoutLeavesFileUnchanged(t *testing.T) {
	dir := goalModule(t, map[string]string{"app/load.goal": plainGoFix})
	file := filepath.Join(dir, "app", "load.goal")

	var out, errOut bytes.Buffer
	if err := run([]string{"fix", file}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "Result[[]byte, error]") || !strings.Contains(out.String(), "read(p)?") {
		t.Fatalf("stdout did not contain the idiomatic rewrite:\n%s", out.String())
	}
	on, _ := os.ReadFile(file)
	if string(on) != plainGoFix {
		t.Fatalf("default mode must not modify the file on disk, but it changed:\n%s", on)
	}
}

// `goal fix -inplace` writes the rewrite back and reports the file as fixed.
func TestFixInplaceWritesFile(t *testing.T) {
	dir := goalModule(t, map[string]string{"app/load.goal": plainGoFix})
	file := filepath.Join(dir, "app", "load.goal")

	var out, errOut bytes.Buffer
	if err := run([]string{"fix", "-inplace", file}, &out, &errOut); err != nil {
		t.Fatalf("fix -inplace failed: %v\n%s", err, errOut.String())
	}
	on, _ := os.ReadFile(file)
	if !strings.Contains(string(on), "Result[[]byte, error]") {
		t.Fatalf("-inplace did not rewrite the file:\n%s", on)
	}
	if !strings.Contains(out.String(), "fixed") {
		t.Fatalf("expected a 'fixed' line on stdout, got:\n%s", out.String())
	}
}

// `goal fix -inplace` on an already-idiomatic file makes no change and reports nothing.
func TestFixInplaceNoChange(t *testing.T) {
	const idiomatic = `package app

func load(p string) Result[[]byte, error] {
	f := read(p)?
	return Result.Ok(f)
}
`
	dir := goalModule(t, map[string]string{"app/load.goal": idiomatic})
	file := filepath.Join(dir, "app", "load.goal")
	info, _ := os.Stat(file)
	before := info.ModTime()

	var out, errOut bytes.Buffer
	if err := run([]string{"fix", "-inplace", file}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}
	if strings.Contains(out.String(), "fixed") {
		t.Fatalf("unchanged file should not be reported as fixed:\n%s", out.String())
	}
	info2, _ := os.Stat(file)
	if !info2.ModTime().Equal(before) {
		t.Fatal("unchanged file was rewritten (mtime changed)")
	}
}

// Changing an exported function's signature emits a warning to stderr.
func TestFixExportedWarning(t *testing.T) {
	const exported = `package app

func Load(p string) ([]byte, error) {
	f, err := read(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}
`
	dir := goalModule(t, map[string]string{"app/load.goal": exported})
	var out, errOut bytes.Buffer
	if err := run([]string{"fix", filepath.Join(dir, "app", "load.goal")}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(errOut.String(), "exported") || !strings.Contains(errOut.String(), "Load") {
		t.Fatalf("expected an exported-signature warning on stderr, got:\n%s", errOut.String())
	}
}

// A non-conforming return hidden inside a type switch must be seen: the function is left
// unconverted and reported, never rewritten to a signature whose return it cannot reach.
func TestFixSkipsNonConformingReturnInTypeSwitch(t *testing.T) {
	const typeSwitch = `package app

func classify(v any, err error) (int, error) {
	switch t := v.(type) {
	case int:
		return t, err
	}
	return 0, nil
}
`
	dir := goalModule(t, map[string]string{"app/load.goal": typeSwitch})
	var out, errOut bytes.Buffer
	if err := run([]string{"fix", filepath.Join(dir, "app", "load.goal")}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}
	if strings.Contains(out.String(), "Result[") {
		t.Fatalf("function with a non-conforming return in a type switch must NOT be converted:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "non-propagating return") || !strings.Contains(errOut.String(), "classify") {
		t.Fatalf("expected a non-propagating-return skip for classify on stderr, got:\n%s", errOut.String())
	}
}

// Same guarantee for a non-conforming return inside a labeled statement.
func TestFixSkipsNonConformingReturnInLabeledStmt(t *testing.T) {
	const labeled = `package app

func scan(items []int, err error) (int, error) {
Outer:
	for _, it := range items {
		if it < 0 {
			break Outer
		}
		return it, err
	}
	return 0, nil
}
`
	dir := goalModule(t, map[string]string{"app/load.goal": labeled})
	var out, errOut bytes.Buffer
	if err := run([]string{"fix", filepath.Join(dir, "app", "load.goal")}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}
	if strings.Contains(out.String(), "Result[") {
		t.Fatalf("function with a non-conforming return in a labeled statement must NOT be converted:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "non-propagating return") || !strings.Contains(errOut.String(), "scan") {
		t.Fatalf("expected a non-propagating-return skip for scan on stderr, got:\n%s", errOut.String())
	}
}

// A `(error, error)` function has no success value to lift into a Result — neither error
// is a value to wrap — so `goal fix` must leave it unchanged and report it out of scope,
// exactly like the other multi-non-error shapes, while a normal `(T, error)` in the same
// file still converts to `Result[T, error]`.
func TestFixSkipsErrorErrorButConvertsTError(t *testing.T) {
	const mixed = `package app

func pair(k string) (error, error) {
	a, err := probe(k)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func load(p string) ([]byte, error) {
	f, err := read(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}
`
	dir := goalModule(t, map[string]string{"app/load.goal": mixed})
	file := filepath.Join(dir, "app", "load.goal")

	var out, errOut bytes.Buffer
	if err := run([]string{"fix", file}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}

	// `pair` must NOT be rewritten to a Result and must be reported out of scope.
	if strings.Contains(out.String(), "Result[error") {
		t.Fatalf("(error, error) must not be converted to a Result, but output has it:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "pair") ||
		!strings.Contains(errOut.String(), "not auto-converted to Result") {
		t.Fatalf("expected an out-of-scope skip for pair on stderr, got:\n%s", errOut.String())
	}

	// The ordinary `(T, error)` shape still converts, proving the guard is narrow.
	if !strings.Contains(out.String(), "Result[[]byte, error]") || !strings.Contains(out.String(), "read(p)?") {
		t.Fatalf("normal (T, error) conversion regressed:\n%s", out.String())
	}
}

// A non-existent path is an operational error.
func TestFixBadPath(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"fix", "/no/such/path.goal"}, &out, &errOut); err == nil {
		t.Fatal("expected an error for a missing path")
	}
}

// When fix's own rewrite fails to reparse, `goal fix -inplace` must abort with a
// diagnostic naming the file and leave the original file byte-identical — never
// overwrite it with the corrupt intermediate.
func TestFixInplaceAbortsOnReparseFailure(t *testing.T) {
	dir := goalModule(t, map[string]string{"app/load.goal": plainGoFix})
	file := filepath.Join(dir, "app", "load.goal")

	// The first parse (of the pristine input) succeeds, so the fixers rewrite the
	// file; the second parse (of that rewrite) fails, exercising the corruption
	// guard. Restore the hook afterwards so other tests see the real parser.
	orig := fix.ParseFileFn
	defer func() { fix.ParseFileFn = orig }()
	calls := 0
	fix.ParseFileFn = func(src string) (*ast.File, error) {
		calls++
		if calls >= 2 {
			return nil, fmt.Errorf("synthetic reparse failure")
		}
		return orig(src)
	}

	var out, errOut bytes.Buffer
	err := run([]string{"fix", "-inplace", file}, &out, &errOut)
	if err == nil {
		t.Fatal("expected fix to fail when its own rewrite cannot be reparsed")
	}
	if !strings.Contains(err.Error(), "load.goal") || !strings.Contains(err.Error(), "reparse") {
		t.Fatalf("error must name the file and the reparse failure, got: %v", err)
	}
	on, _ := os.ReadFile(file)
	if string(on) != plainGoFix {
		t.Fatalf("-inplace must leave the file byte-identical on abort, but it changed:\n%s", on)
	}
}

// A report emitted while another function's rewrite shifts the source must carry a
// line number against the FINAL text, not the stale intermediate the report was
// first detected in. `first` converts to Result (its body collapses, moving every
// later line up); the exported `Second` cannot be auto-converted and is reported.
func TestFixReportLineAgainstFinalText(t *testing.T) {
	const twoFuncs = `package app

func first(p string) ([]byte, error) {
	f, err := read(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func Second(p string) ([]byte, error) {
	f, err := read(p)
	if err != nil {
		return nil, err
	}
	return f, nil
}
`
	dir := goalModule(t, map[string]string{"app/load.goal": twoFuncs})
	file := filepath.Join(dir, "app", "load.goal")

	var out, errOut bytes.Buffer
	if err := run([]string{"fix", file}, &out, &errOut); err != nil {
		t.Fatalf("fix failed: %v\n%s", err, errOut.String())
	}

	// The exported-signature report for Second, e.g.
	//   .../load.goal:11: skipped: [result-sig] exported `Second` ...
	re := regexp.MustCompile(`:(\d+): .*Second`)
	m := re.FindStringSubmatch(errOut.String())
	if m == nil {
		t.Fatalf("expected a report mentioning Second, got stderr:\n%s", errOut.String())
	}
	var line int
	fmt.Sscanf(m[1], "%d", &line)

	// The reported line must point at `func Second` in the REWRITTEN source (final
	// text). Under the pre-fix behavior it pointed at Second's original line, which
	// after first's body collapsed no longer holds the func declaration.
	lines := strings.Split(out.String(), "\n")
	if line < 1 || line > len(lines) {
		t.Fatalf("reported line %d out of range for rewritten output (%d lines):\n%s", line, len(lines), out.String())
	}
	if !strings.Contains(lines[line-1], "func Second") {
		t.Fatalf("report line %d should point at `func Second` in the final text, but line %d is:\n%q\nfull output:\n%s",
			line, line, lines[line-1], out.String())
	}
}

// `goal fix` discovers and rewrites every .goal file under a directory.
func TestFixDirectory(t *testing.T) {
	dir := goalModule(t, map[string]string{"app/load.goal": plainGoFix})
	var out, errOut bytes.Buffer
	if err := run([]string{"fix", dir}, &out, &errOut); err != nil {
		t.Fatalf("fix dir failed: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "Result[[]byte, error]") {
		t.Fatalf("directory fix did not rewrite the package:\n%s", out.String())
	}
}

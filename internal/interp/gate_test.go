package interp

// These tests prove US-022 "Gate interp on native sema only": the interpreter's
// run path validates its input solely through internal/sema (native AST checks)
// and refuses a program that violates a static guarantee BEFORE evaluation, with
// a located diagnostic. A non-exhaustive `match` is the canonical violation. A
// dependency test pins the native-only envelope: internal/interp must not depend
// on internal/typecheck or go/types (REWRITE-ARCHITECTURE.md §3.2 — types are
// checked statically and erased at runtime, so goscript runs with no Go
// toolchain).

import (
	"os/exec"
	"strings"
	"testing"

	"goal/internal/parser"
	"goal/internal/sema"
)

// runProgram parses + resolves src and runs it through the interpreter, returning
// Run's error (the gate refusal, a runtime error, or nil).
func runProgram(t *testing.T, src string) error {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := sema.Resolve(file)
	return New(file, info).Run()
}

// nonExhaustiveProgram's main matches a three-variant in-file enum but handles
// only one variant with no `_` rest-arm — the exhaustiveness guarantee sema
// enforces (feature 02). The gate must refuse it before evaluating main.
const nonExhaustiveProgram = `package main

enum Shape {
	Point
	Circle { radius: int }
	Square { side: int }
}

func main() {
	s := Shape.Point
	match s {
		Shape.Circle(c) => return
	}
}
`

func TestRunRefusesNonExhaustiveMatch(t *testing.T) {
	err := runProgram(t, nonExhaustiveProgram)
	if err == nil {
		t.Fatal("Run on a non-exhaustive match: want a refusal error, got nil")
	}
	msg := err.Error()
	// Located: the refusal carries a source position (line:col) and the stable
	// diagnostic code, proving the gate refused via the native sema check rather
	// than failing somewhere during evaluation.
	if !strings.Contains(msg, "non-exhaustive-match") {
		t.Fatalf("refusal %q does not name the diagnostic code non-exhaustive-match", msg)
	}
	if !strings.Contains(msg, "refused before run") {
		t.Fatalf("refusal %q is not the pre-run gate refusal", msg)
	}
	// A located diagnostic: the enum decl + match sit on lines >1, so the
	// rendered position must contain a colon-separated line:col, not 0:0.
	if !strings.Contains(msg, ":") || strings.Contains(msg, " 0:0:") {
		t.Fatalf("refusal %q is not located (missing line:col)", msg)
	}
}

// exhaustiveProgram covers every variant, so the gate passes and main runs to a
// clean no-op completion — no false refusal.
const exhaustiveProgram = `package main

enum Shape {
	Point
	Circle { radius: int }
	Square { side: int }
}

func main() {
	s := Shape.Point
	match s {
		Shape.Point => {}
		Shape.Circle(c) => {}
		Shape.Square(q) => {}
	}
}
`

func TestRunAllowsExhaustiveMatch(t *testing.T) {
	if err := runProgram(t, exhaustiveProgram); err != nil {
		t.Fatalf("Run on an exhaustive match: unexpected error: %v", err)
	}
}

// warningProgram matches an enum NOT declared in this file: sema cannot prove
// exhaustiveness, so it emits a WARNING (a located deferral), never an Error. The
// gate must NOT refuse on a warning — only violated guarantees block.
const warningProgram = `package main

func main() {
	s := External.A
	match s {
		External.A => return
	}
}
`

func TestRunDoesNotBlockOnWarning(t *testing.T) {
	// Sanity-check the fixture actually produces a Warning and no Error, so this
	// test exercises the warning path rather than a clean program.
	file, err := parser.ParseFile(warningProgram)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := sema.Resolve(file)
	sawWarning, sawError := false, false
	for _, d := range sema.Check(file, info) {
		switch d.Severity {
		case sema.Warning:
			sawWarning = true
		case sema.Error:
			sawError = true
		}
	}
	if !sawWarning {
		t.Fatalf("fixture produced no Warning; cannot exercise the warning path")
	}
	if sawError {
		t.Fatalf("fixture unexpectedly produced an Error; not a pure-warning case")
	}
	// The gate must not refuse on a warning. main reads `External.A`, which the
	// interpreter cannot resolve, so Run may still fail at eval time — but the
	// failure must NOT be the pre-run gate refusal.
	runErr := New(file, info).Run()
	if runErr != nil && strings.Contains(runErr.Error(), "refused before run") {
		t.Fatalf("gate refused a warning-only program: %v", runErr)
	}
}

// TestInterpHasNoGoTypesOrTypecheckDep pins the native-only dependency envelope:
// the interpreter validates exclusively through internal/sema, so its transitive
// dependencies must include neither the Go depth checker (go/types) nor the
// lowered-Go typecheck package.
func TestInterpHasNoGoTypesOrTypecheckDep(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "goal/internal/interp").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	for _, dep := range strings.Fields(string(out)) {
		if dep == "go/types" {
			t.Errorf("internal/interp transitively depends on go/types — the interpreter must validate via native sema only")
		}
		if dep == "goal/internal/typecheck" {
			t.Errorf("internal/interp transitively depends on goal/internal/typecheck — the interpreter must validate via native sema only")
		}
	}
}

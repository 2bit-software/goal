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

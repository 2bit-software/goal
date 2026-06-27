package sema

import (
	"strings"
	"testing"

	"goal/internal/parser"
)

// analyzeExhaustive parses src, resolves it, and returns the exhaustiveness
// diagnostics — the unit harness for the AST exhaustiveness check.
func analyzeExhaustive(t *testing.T, src string) []Diagnostic {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return CheckExhaustive(file, Resolve(file))
}

func TestExhaustiveCleanNoDiagnostics(t *testing.T) {
	const src = `package p

enum Status { Pending Active Cancelled }

func handle(s Status) {
	match s {
		Status.Pending => f()
		Status.Active => f()
		Status.Cancelled => f()
	}
}
`
	if d := analyzeExhaustive(t, src); len(d) != 0 {
		t.Fatalf("exhaustive match produced diagnostics: %+v", d)
	}
}

func TestNonExhaustiveSingleMissing(t *testing.T) {
	const src = `package p

enum Status { Pending Active Cancelled }

func handle(s Status) {
	match s {
		Status.Pending => f()
		Status.Active => f()
	}
}
`
	d := analyzeExhaustive(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if d[0].Severity != Error {
		t.Errorf("want Error severity, got %v", d[0].Severity)
	}
	if d[0].Code != "non-exhaustive-match" {
		t.Errorf("want code non-exhaustive-match, got %q", d[0].Code)
	}
	if !strings.Contains(d[0].Message, "missing variant `Status.Cancelled`") {
		t.Errorf("message does not name the missing variant: %q", d[0].Message)
	}
}

func TestNonExhaustiveDeclarationOrder(t *testing.T) {
	const src = `package p

enum Status { Pending Active Cancelled }

func label(s Status) string {
	return match s {
		Status.Pending => "p"
	}
}
`
	d := analyzeExhaustive(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if !strings.Contains(d[0].Message, "missing variants `Status.Active`, `Status.Cancelled`") {
		t.Errorf("missing variants not listed in declaration order: %q", d[0].Message)
	}
}

func TestRestArmOptOut(t *testing.T) {
	const src = `package p

enum Status { Pending Active Cancelled }

func handle(s Status) {
	match s {
		Status.Active => f()
		_ => f()
	}
}
`
	if d := analyzeExhaustive(t, src); len(d) != 0 {
		t.Fatalf("rest-arm match should opt out of exhaustiveness, got: %+v", d)
	}
}

func TestUnknownEnumDeferred(t *testing.T) {
	const src = `package p

func route(c Color) string {
	return match c {
		Color.Red => "r"
		Color.Green => "g"
	}
}
`
	d := analyzeExhaustive(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if d[0].Severity != Warning {
		t.Errorf("deferral should be a Warning, got %v", d[0].Severity)
	}
	if !strings.Contains(d[0].Message, "exhaustiveness deferred") {
		t.Errorf("deferral message unexpected: %q", d[0].Message)
	}
}

func TestResultMatchSkipped(t *testing.T) {
	const src = `package p

func load(r Result[int, error]) int {
	return match r {
		Result.Ok(v) => v
		Result.Err(e) => 0
	}
}
`
	if d := analyzeExhaustive(t, src); len(d) != 0 {
		t.Fatalf("Result match is not exhaustiveness' concern, got: %+v", d)
	}
}

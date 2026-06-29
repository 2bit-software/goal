package sema

import (
	"strings"
	"testing"

	"goal/internal/parser"
)

// analyzeConvert parses src, resolves it, and returns the derive-convert totality
// diagnostics.
func analyzeConvert(t *testing.T, src string) []Diagnostic {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return CheckConvert(file, Resolve(file))
}

func TestConvertUnsourcedField(t *testing.T) {
	const src = `package conv

type EventExecution struct {
	ID       string
	Railroad string
}

type StoredEvent struct {
	ID         string
	Railroad   string
	ExternalID string
}

derive func toStorage(e EventExecution) StoredEvent
`
	d := analyzeConvert(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if SeverityLabel(d[0].Severity) != "error" || d[0].Code != "unsourced-field" {
		t.Errorf("want Error unsourced-field, got %v %q", d[0].Severity, d[0].Code)
	}
	if !strings.Contains(d[0].Message, "target field `StoredEvent.ExternalID` has no same-named source field") {
		t.Errorf("message mismatch: %q", d[0].Message)
	}
}

func TestConvertUnbridgedField(t *testing.T) {
	const src = `package conv

type UUID struct{ s string }

type EventExecution struct {
	ID       UUID
	Railroad string
}

type StoredEvent struct {
	ID       string
	Railroad string
}

derive func toStorage(e EventExecution) StoredEvent
`
	d := analyzeConvert(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if SeverityLabel(d[0].Severity) != "error" || d[0].Code != "unbridged-field" {
		t.Errorf("want Error unbridged-field, got %v %q", d[0].Severity, d[0].Code)
	}
	if !strings.Contains(d[0].Message, "no `from func` converts `UUID`→`string`") {
		t.Errorf("message mismatch: %q", d[0].Message)
	}
}

func TestConvertFallibleInTotalDerive(t *testing.T) {
	const src = `package conv

type UUID struct{ s string }

from func parseUUID(s string) (UUID, error) {
	return UUID{s: s}, nil
}

type StoredEvent struct {
	ID       string
	Railroad string
}

type EventExecution struct {
	ID       UUID
	Railroad string
}

derive func toDomain(s StoredEvent) EventExecution
`
	d := analyzeConvert(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if SeverityLabel(d[0].Severity) != "error" || d[0].Code != "fallible-in-total-derive" {
		t.Errorf("want Error fallible-in-total-derive, got %v %q", d[0].Severity, d[0].Code)
	}
	if !strings.Contains(d[0].Message, "this derive is total — declare it returning `(EventExecution, error)`") {
		t.Errorf("message mismatch: %q", d[0].Message)
	}
}

func TestConvertIdentityFieldsClean(t *testing.T) {
	const src = `package conv

type A struct {
	ID    string
	Count int
}

type B struct {
	ID    string
	Count int
}

derive func toB(a A) B
`
	if d := analyzeConvert(t, src); len(d) != 0 {
		t.Fatalf("an all-identity derive is total and must draw nothing, got: %+v", d)
	}
}

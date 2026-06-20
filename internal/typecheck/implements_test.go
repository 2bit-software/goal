package typecheck

import (
	"strings"
	"testing"
)

func diagsImplements(t *testing.T, src string) []Diagnostic {
	t.Helper()
	p, err := Load(pkgOf(map[string]string{"x.goal": src}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return CheckImplements(p)
}

// The payoff case: the interface method takes `int`, the concrete method takes `ID`
// (a type alias for int). The lexical 07 check compares signature text and would call
// this a mismatch; go/types resolves the alias and sees them identical.
func TestImplementsAliasEqualSignatureIsNotAMismatch(t *testing.T) {
	const src = `package demo

type ID = int

type Store interface {
    Get(id int) string
}

type Mem struct implements Store {
}

func (m Mem) Get(id ID) string {
    return ""
}
`
	if d := diagsImplements(t, src); len(d) != 0 {
		t.Errorf("alias-equal signature wrongly flagged: %v", d)
	}
}

func TestImplementsMissingMethod(t *testing.T) {
	const src = `package demo

type Store interface {
    Get(id int) string
}

type Mem struct implements Store {
}
`
	d := diagsImplements(t, src)
	if len(d) != 1 || d[0].Code != "unimplemented-method" {
		t.Fatalf("want one unimplemented-method diagnostic, got %v", d)
	}
	if !strings.Contains(d[0].Message, "Get") {
		t.Errorf("message should name the missing method Get: %q", d[0].Message)
	}
	if !strings.HasSuffix(d[0].Pos.Filename, ".goal") {
		t.Errorf("diagnostic not located in .goal: %s", d[0].Pos)
	}
}

// A qualified interface (io.Writer) the lexical check defers as out-of-package: go/types
// resolves it through the imports and confirms satisfaction.
func TestImplementsQualifiedInterfaceSatisfied(t *testing.T) {
	const src = `package demo

import "io"

type Sink struct implements io.Writer {
}

func (s Sink) Write(p []byte) (int, error) {
    return len(p), nil
}

var _ = io.Discard
`
	if d := diagsImplements(t, src); len(d) != 0 {
		t.Errorf("qualified interface satisfaction wrongly flagged: %v", d)
	}
}

func TestImplementsQualifiedInterfaceMissing(t *testing.T) {
	const src = `package demo

import "io"

type Sink struct implements io.Writer {
}

var _ = io.Discard
`
	// The lowered Go won't compile (the assertion fails), but the check still resolves
	// io.Writer and reports the missing Write at the clause.
	d := diagsImplements(t, src)
	if len(d) != 1 || !strings.Contains(d[0].Message, "Write") {
		t.Fatalf("want a missing-Write diagnostic for io.Writer, got %v", d)
	}
}

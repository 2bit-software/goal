package sema

import (
	"strings"
	"testing"

	"goal/internal/parser"
)

func analyzeImplements(t *testing.T, src string) []Diagnostic {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return CheckImplements(file, Resolve(file))
}

func TestImplementsSatisfiedValueClean(t *testing.T) {
	const src = `package p

type Stringer interface {
	String() string
}

type Point struct implements Stringer {
	X int
	Y int
}

func (p Point) String() string {
	return "point"
}
`
	if d := analyzeImplements(t, src); len(d) != 0 {
		t.Fatalf("satisfied value receiver should be clean, got: %+v", d)
	}
}

func TestImplementsSatisfiedPointerClean(t *testing.T) {
	const src = `package p

type Resetter interface {
	Reset()
}

type Counter struct implements Resetter {
	n int
}

func (c *Counter) Reset() {
	c.n = 0
}
`
	if d := analyzeImplements(t, src); len(d) != 0 {
		t.Fatalf("pointer-receiver method satisfies the interface, got: %+v", d)
	}
}

func TestImplementsMissingMethod(t *testing.T) {
	const src = `package p

type Shaper interface {
	Area() float64
	Perimeter() float64
}

type Square struct implements Shaper {
	side float64
}

func (s Square) Perimeter() float64 {
	return 4 * s.side
}
`
	d := analyzeImplements(t, src)
	if len(d) != 1 || d[0].Code != "unimplemented-method" {
		t.Fatalf("want 1 unimplemented-method, got: %+v", d)
	}
	if !strings.Contains(d[0].Message, "missing method `Area()") {
		t.Errorf("message should name Area(): %q", d[0].Message)
	}
}

func TestImplementsWrongSignature(t *testing.T) {
	const src = `package p

type Writer interface {
	Write(p []byte) (int, error)
}

type Encoder struct implements Writer {
	buf []byte
}

func (e Encoder) Write(s string) int {
	return len(s)
}
`
	d := analyzeImplements(t, src)
	if len(d) != 1 || d[0].Code != "method-signature-mismatch" {
		t.Fatalf("want 1 method-signature-mismatch, got: %+v", d)
	}
	if !strings.Contains(d[0].Message, "method `Write` has signature") {
		t.Errorf("message should report the signature: %q", d[0].Message)
	}
}

func TestImplementsEmbeddedMissing(t *testing.T) {
	const src = `package p

type Reader interface {
	Read(p []byte) (int, error)
}

type ReadWriter interface {
	Reader
	Write(p []byte) (int, error)
}

type Sink struct implements ReadWriter {
	n int
}

func (s Sink) Write(p []byte) (int, error) {
	return len(p), nil
}
`
	d := analyzeImplements(t, src)
	if len(d) != 1 || d[0].Code != "unimplemented-method" {
		t.Fatalf("want 1 unimplemented-method (embedded Read), got: %+v", d)
	}
	if !strings.Contains(d[0].Message, "missing method `Read(") {
		t.Errorf("message should name the embedded Read: %q", d[0].Message)
	}
}

func TestImplementsSealedTrivial(t *testing.T) {
	const src = `package p

sealed interface Shape {}

type Circle struct implements Shape {
	r float64
}
`
	if d := analyzeImplements(t, src); len(d) != 0 {
		t.Fatalf("sealed interface is trivially satisfied, got: %+v", d)
	}
}

func TestImplementsQualifiedDefers(t *testing.T) {
	const src = `package p

import "io"

type Discard struct implements io.Writer {}

func (Discard) Write(p []byte) (int, error) {
	return len(p), nil
}
`
	d := analyzeImplements(t, src)
	if len(d) != 1 || SeverityLabel(d[0].Severity) != "warning" || d[0].Code != "unresolved-interface" {
		t.Fatalf("qualified interface should defer with a Warning, got: %+v", d)
	}
}

func TestImplementsUndeclaredDefers(t *testing.T) {
	const src = `package p

type Plugin struct implements Handler {
	id int
}

func (p Plugin) Handle() {}
`
	d := analyzeImplements(t, src)
	if len(d) != 1 || SeverityLabel(d[0].Severity) != "warning" || d[0].Code != "unresolved-interface" {
		t.Fatalf("undeclared interface should defer with a Warning, got: %+v", d)
	}
}

package backendtest

import (
	"strings"
	"testing"

	"goal/internal/backend"
)

// TestQuestionArityFallbackWarnsOnUnresolvableImport asserts that a `?` on a
// package-qualified callee whose package cannot be imported during arity resolution
// surfaces a `question-arity-fallback` warning naming the package and the fallback
// used (US-022). The import path is guaranteed unresolvable so the test does not
// depend on the machine's module cache.
func TestQuestionArityFallbackWarnsOnUnresolvableImport(t *testing.T) {
	src := `package p

import "example.invalid/nope"

func do() Result[int, error] {
	x := nope.Do()?
	return Result.Ok(x)
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	var got *string
	for i := range out.Warnings {
		if out.Warnings[i].Code == "question-arity-fallback" {
			m := out.Warnings[i].Message
			got = &m
		}
	}
	if got == nil {
		t.Fatalf("expected a question-arity-fallback warning, got %#v", out.Warnings)
	}
	if !strings.Contains(*got, "nope") {
		t.Errorf("warning should name the package qualifier; got %q", *got)
	}
	// The warning is out-of-band: it must never leak into the generated Go.
	if strings.Contains(out.Go, "question-arity-fallback") {
		t.Errorf("warning code leaked into generated Go")
	}
}

// TestQuestionArityCleanStdlibNoWarn asserts that a `?` on a resolvable stdlib callee
// (os.WriteFile) produces no arity-fallback warning: real resolution succeeded, so
// the generated code is machine-independent and nothing new is emitted.
func TestQuestionArityCleanStdlibNoWarn(t *testing.T) {
	src := `package p

import "os"

func do() Result[int, error] {
	os.WriteFile("f", nil, 0)?
	return Result.Ok(1)
}
`
	out, err := backend.Transpile(src)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	for _, w := range out.Warnings {
		if w.Code == "question-arity-fallback" {
			t.Fatalf("unexpected arity-fallback warning for a resolvable stdlib import: %q", w.Message)
		}
	}
}

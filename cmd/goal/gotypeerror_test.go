package main

import (
	"bytes"
	"strings"
	"testing"
)

// goal check surfaces the plain-Go type errors the typed depth stage already
// observes, so `ok` means the package actually compiles (US-011). A type
// mismatch (`int + string`) and a call to an undefined function each render as a
// located `[go-type-error]` diagnostic and the command exits non-zero.
func TestCheckSurfacesGoTypeErrors(t *testing.T) {
	const bad = `package demo

func add(a int, b string) int {
	return a + b
}

func run() {
	missingFn()
}
`
	dir := goalModule(t, map[string]string{"bad.goal": bad})

	var out, errOut bytes.Buffer
	err := run([]string{"check", dir}, &out, &errOut)
	if err == nil {
		t.Fatalf("expected check to fail on the Go type errors\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	got := errOut.String()
	if !strings.Contains(got, "[go-type-error]") {
		t.Errorf("go type error not surfaced with the [go-type-error] code:\n%s", got)
	}
	// The mismatched-types error is located at the `return a + b` line.
	if !strings.Contains(got, "bad.goal:4") {
		t.Errorf("mismatched-types diagnostic not located at bad.goal:4:\n%s", got)
	}
	// The undefined function is surfaced too, at its call site.
	if !strings.Contains(got, "missingFn") {
		t.Errorf("undefined-function diagnostic not surfaced:\n%s", got)
	}
}

// A package that compiles cleanly still reports ok with exit 0 — the new check
// must not false-positive.
func TestCheckCleanGoTypesPasses(t *testing.T) {
	const clean = `package demo

func add(a int, b int) int {
	return a + b
}
`
	dir := goalModule(t, map[string]string{"main.goal": clean})

	var out, errOut bytes.Buffer
	if err := run([]string{"check", dir}, &out, &errOut); err != nil {
		t.Fatalf("clean check failed: %v\nstderr: %s", err, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "ok" {
		t.Errorf("want ok, got stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

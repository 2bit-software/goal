package check

import "testing"

// hasDiagCode reports whether any diagnostic carries the given code. (Named distinctly
// from fields_test.go's hasCode so this file stands alone.)
func hasDiagCode(diags []Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// A closed-E match that re-wraps the bound error of a same-E scrutinee
// (`Result.Err(e) => { return Result.Err(e) }`) is a valid passthrough: neither the arm
// pattern nor the re-wrap is a closedness problem, so neither the pattern nor the return
// may produce an `unresolved-err-value` deferral.
func TestClosedPassthroughNotDeferred(t *testing.T) {
	const src = `package demo

enum E {
	Bad { why: string }
	Worse
}

type T struct{ v int }

func inner() Result[T, E] { return Result.Ok(T{v: 1}) }

func outer() Result[T, E] {
	match inner() {
		Result.Ok(x)  => { return Result.Ok(x) }
		Result.Err(e) => { return Result.Err(e) }
	}
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze errored: %v", err)
	}
	if hasDiagCode(diags, "unresolved-err-value") {
		t.Errorf("passthrough re-wrap should not defer; got unresolved-err-value:\n%v", diags)
	}
	for _, d := range diags {
		if d.Severity == Error {
			t.Errorf("unexpected error on valid passthrough: [%s] %s", d.Code, d.Message)
		}
	}
}

// The passthrough suppression must not weaken real closedness checking: a foreign-enum
// Err and an unknown-variant Err are still hard errors, including inside a match arm body.
func TestClosedRealViolationsStillCaught(t *testing.T) {
	const src = `package demo

enum E { Bad { why: string }  Worse }
enum Other { Nope }

type T struct{ v int }

func crossEnum() Result[T, E] {
	return Result.Err(Other.Nope)
}

func badVariant() Result[T, E] {
	return Result.Err(E.Typo)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze errored: %v", err)
	}
	if !hasDiagCode(diags, "err-outside-closed-enum") {
		t.Errorf("foreign-enum Err no longer flagged:\n%v", diags)
	}
	if !hasDiagCode(diags, "unknown-error-variant") {
		t.Errorf("unknown-variant Err no longer flagged:\n%v", diags)
	}
}

// Passthrough is suppressed only when the scrutinee's error enum matches the function's.
// A re-wrap of an error bound from a different-E scrutinee cannot be confirmed closed, so
// it must stay deferred (a warning) rather than be silently suppressed.
func TestClosedPassthroughDifferentEStaysDeferred(t *testing.T) {
	const src = `package demo

enum E { Bad { why: string }  Worse }
enum Other { Nope }

type T struct{ v int }

func wrongScrutinee() Result[T, Other] { return Result.Ok(T{v: 2}) }

func passthroughWrongE() Result[T, E] {
	match wrongScrutinee() {
		Result.Ok(x)  => { return Result.Ok(x) }
		Result.Err(e) => { return Result.Err(e) }
	}
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze errored: %v", err)
	}
	if !hasDiagCode(diags, "unresolved-err-value") {
		t.Errorf("different-E re-wrap should remain deferred, but no unresolved-err-value:\n%v", diags)
	}
}

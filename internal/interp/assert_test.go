package interp

// These tests prove US-019 "Eval assert at runtime": the interpreter evaluates
// goal's `assert` statement — a false condition panics with the located
// assertion message; a true condition is a no-op and evaluation continues. The
// programs are modeled on features/10-assert/examples/{bank,message}.goal: the
// bare form (`assert amount > 0`) and the printf-message form
// (`assert age >= 0, "...", age`). stdlib testing, no testify.

import (
	"errors"
	"strings"
	"testing"

	"goal/internal/ast"
	"goal/internal/token"
)

const assertProgram = `package bank

func goodWithdraw() int {
	balance := 100
	amount := 10
	assert amount > 0
	return balance - amount
}

func badWithdraw() int {
	balance := 100
	amount := 0
	assert amount > 0
	return balance - amount
}

func goodAge() int {
	age := 5
	assert age >= 0, "age must be non-negative, got %d", age
	return age
}

func badAge() int {
	age := -3
	assert age >= 0, "age must be non-negative, got %d", age
	return age
}

func main() {}
`

// TestAssertTrueIsNoOp: a true condition lets evaluation continue to the
// following statements (FR-1).
func TestAssertTrueIsNoOp(t *testing.T) {
	ip := newInterp(t, assertProgram)
	got := evalFn(t, ip, "goodWithdraw")
	if got.Kind != KindInt || got.Int != 90 {
		t.Fatalf("goodWithdraw = %v, want 90", got)
	}
}

// TestAssertFalseBarePanicsLocated: a false bare assertion raises a panicSignal
// whose message marks an assertion failure, includes the condition text, and is
// located (FR-2).
func TestAssertFalseBarePanicsLocated(t *testing.T) {
	ip := newInterp(t, assertProgram)
	err := evalFnErr(t, ip, "badWithdraw")
	var p panicSignal
	if !errors.As(err, &p) {
		t.Fatalf("badWithdraw: error %v is not a panicSignal", err)
	}
	msg := p.value.Str
	if p.value.Kind != KindString || !strings.Contains(msg, "assertion failed") {
		t.Fatalf("panic message = %q, want it to contain \"assertion failed\"", msg)
	}
	if !strings.Contains(msg, "amount > 0") {
		t.Fatalf("panic message = %q, want it to contain the condition \"amount > 0\"", msg)
	}
	// Located: the message carries a source position (line:col) prefix.
	if !strings.Contains(msg, ":") {
		t.Fatalf("panic message = %q, want a located \"<pos>:\" prefix", msg)
	}
}

// TestAssertFalseMessageFormFormatsMessage: a false printf-message assertion
// includes the formatted message text in the panic (FR-3).
func TestAssertFalseMessageFormFormatsMessage(t *testing.T) {
	ip := newInterp(t, assertProgram)
	err := evalFnErr(t, ip, "badAge")
	var p panicSignal
	if !errors.As(err, &p) {
		t.Fatalf("badAge: error %v is not a panicSignal", err)
	}
	msg := p.value.Str
	if !strings.Contains(msg, "assertion failed") || !strings.Contains(msg, "age >= 0") {
		t.Fatalf("panic message = %q, want \"assertion failed\" + condition", msg)
	}
	if !strings.Contains(msg, "age must be non-negative, got -3") {
		t.Fatalf("panic message = %q, want the formatted message \"age must be non-negative, got -3\"", msg)
	}
}

// TestAssertMessageFormTrueIsNoOp: the message form is likewise a no-op when the
// condition holds (FR-1).
func TestAssertMessageFormTrueIsNoOp(t *testing.T) {
	ip := newInterp(t, assertProgram)
	got := evalFn(t, ip, "goodAge")
	if got.Kind != KindInt || got.Int != 5 {
		t.Fatalf("goodAge = %v, want 5", got)
	}
}

// TestAssertNonBoolConditionIsRefused: a non-bool condition is a descriptive
// refusal, not a silent no-op (FR-4). The AssertStmt is hand-built so a non-bool
// condition reaches execAssert directly.
func TestAssertNonBoolConditionIsRefused(t *testing.T) {
	ip := newInterp(t, assertProgram)
	stmt := &ast.AssertStmt{Cond: &ast.BasicLit{Kind: token.INT, Value: "3"}}
	err := ip.execAssert(stmt, ip.root)
	if err == nil {
		t.Fatalf("execAssert on a non-bool condition: expected an error, got none")
	}
	if _, isPanic := err.(panicSignal); isPanic {
		t.Fatalf("non-bool condition should be a descriptive refusal, not a panic: %v", err)
	}
	if !strings.Contains(err.Error(), "must be bool") {
		t.Fatalf("error = %q, want it to mention \"must be bool\"", err.Error())
	}
}

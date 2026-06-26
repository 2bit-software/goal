package check

import "testing"

// TestDiscardQuestionConsumesResult guards the §3 must-use check against a false positive:
// `_ := f(...)?` is not a whole-Result discard — the `?` consumes the Result by
// propagation and the `_` discards only the unwrapped Ok value, an ordinary value discard.
// (Regression: it reported `unresolved-result-discard` though the Result is used; the
// statement-leading `f(...)?` form was already recognized as consumed, but the
// `_ :=`-discard path ignored the trailing `?`.)
func TestDiscardQuestionConsumesResult(t *testing.T) {
	const src = `package x

func Write(p string) Result[int, error] { return Result.Ok(0) }

func run() Result[bool, error] {
	_ := Write("a")?
	return Result.Ok(true)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if hasCode(diags, "unresolved-result-discard") {
		for _, d := range diags {
			t.Logf("diag: [%s] %s", d.Code, d.Message)
		}
		t.Errorf("`_ := f(...)?` was flagged as an unresolved Result discard, but `?` consumes the Result")
	}
}

// TestInterfaceMethodSigNotCall guards the §3 must-use check against a false positive: an
// interface method signature whose name matches a concrete Result-returning func/method
// registered in FuncSignatures wears the `IDENT(params) ReturnType` shape of a
// statement-leading call. (Regression: `Provision(spec) Result[…]` inside an
// `interface{…}` was reported as a dropped/continued Result call because the token before
// the method name is the interface's `{`.)
func TestInterfaceMethodSigNotCall(t *testing.T) {
	const src = `package x

type Env struct { id string }

type Sandbox interface {
	Provision(spec int) Result[Env, error]
	Teardown(env Env) Result[int, error]
}

type Real struct {}

func (r Real) Provision(spec int) Result[Env, error] {
	return Result.Ok(Env{id: "x"})
}

func (r Real) Teardown(env Env) Result[int, error] {
	return Result.Ok(0)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	for _, code := range []string{"unresolved-result-use", "dropped-result", "unresolved-result-discard"} {
		if hasCode(diags, code) {
			for _, d := range diags {
				t.Logf("diag: [%s] %s", d.Code, d.Message)
			}
			t.Errorf("interface method signature was misread as a call site (%s)", code)
		}
	}
}

// TestUnderscoreDiscardWithoutQuestionStillDeferred confirms the fix above did not blind
// the check to a genuine whole-Result discard: `_ := f(...)` with no `?` must still defer.
func TestUnderscoreDiscardWithoutQuestionStillDeferred(t *testing.T) {
	const src = `package x

func Write(p string) Result[int, error] { return Result.Ok(0) }

func run() {
	_ := Write("a")
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "unresolved-result-discard") {
		t.Errorf("`_ := f(...)` (no `?`) should still defer with unresolved-result-discard")
	}
}

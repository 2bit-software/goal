package check

import "testing"

// A `?` on a void callee (nothing to propagate) is rejected: `?` needs a trailing error.
func TestQuestionVoidCalleeRejected(t *testing.T) {
	const src = `package x

func noth() {}

func f() Result[bool, error] {
	noth()?
	return Result.Ok(true)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "question-callee-no-error") {
		t.Fatalf("expected question-callee-no-error for a void `?` callee, got %+v", diags)
	}
}

// A `?` on a callee whose last result is not `error` is rejected.
func TestQuestionNonErrorCalleeRejected(t *testing.T) {
	const src = `package x

func now() int { return 0 }

func f() Result[bool, error] {
	now()?
	return Result.Ok(true)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "question-callee-no-error") {
		t.Fatalf("expected question-callee-no-error for a non-error `?` callee, got %+v", diags)
	}
}

// Binding a value from an error-only callee (`x := mkdirLike()?`) is rejected — there is no
// value to bind; the author wants the bare `…?` form.
func TestQuestionBindValueFromErrorOnlyRejected(t *testing.T) {
	const src = `package x

func clean() error { return nil }

func f() Result[bool, error] {
	x := clean()?
	_ = x
	return Result.Ok(true)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "question-binds-nonvalue") {
		t.Fatalf("expected question-binds-nonvalue for value-bind on an error-only callee, got %+v", diags)
	}
}

// A `?` on an `Option`-returning callee inside a `Result` function is rejected: an `Option`
// has no `error` to propagate (its failure is a `None`).
func TestQuestionOptionCalleeRejected(t *testing.T) {
	const src = `package x

func find() Option[int] { return Option.Some(1) }

func f() Result[bool, error] {
	find()?
	return Result.Ok(true)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "question-callee-no-error") {
		t.Fatalf("expected question-callee-no-error for an Option `?` callee in a Result fn, got %+v", diags)
	}
}

// A `?` on a closed-E `Result[T, E]` callee inside an open-E `Result[_, error]` function is
// rejected: its failure is an `Err[T, E]` sum, not a plain `error`.
func TestQuestionClosedECalleeRejected(t *testing.T) {
	const src = `package x

enum MyErr { Boom }

func sub() Result[int, MyErr] { return Result.Ok(0) }

func f() Result[bool, error] {
	sub()?
	return Result.Ok(true)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "question-callee-no-error") {
		t.Fatalf("expected question-callee-no-error for a closed-E `?` callee in an open-E fn, got %+v", diags)
	}
}

// An unresolvable `?` callee (a method whose receiver type is not inferred) in the discard
// form is surfaced as a Warning, not silently lowered to the two-value form.
func TestQuestionUnresolvedMethodWarns(t *testing.T) {
	const src = `package x

func f(w Closer) Result[bool, error] {
	w.Close()?
	return Result.Ok(true)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !hasCode(diags, "question-callee-unresolved") {
		t.Fatalf("expected question-callee-unresolved warning for a method `?` callee, got %+v", diags)
	}
	for _, d := range diags {
		if d.Code == "question-callee-unresolved" && d.Severity != Warning {
			t.Fatalf("question-callee-unresolved should be a Warning, got %v", d.Severity)
		}
	}
}

// A valid `?` — bare on an in-file error-only call, and value-bind on a Result call — produces
// no question diagnostics (guards against false positives).
func TestQuestionValidCalleesClean(t *testing.T) {
	const src = `package x

func mkdir(p string) error { return nil }

func lookup(id string) Result[string, error] { return Result.Ok(id) }

func f(id string) Result[string, error] {
	mkdir(id)?
	u := lookup(id)?
	return Result.Ok(u)
}
`
	diags, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	for _, d := range diags {
		if d.Feature == "05-question-prop" {
			t.Fatalf("unexpected question diagnostic on valid `?`: [%s] %s", d.Code, d.Message)
		}
	}
}

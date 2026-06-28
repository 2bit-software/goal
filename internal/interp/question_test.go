package interp

// These tests prove US-017 "Eval question-mark unwinding": the interpreter
// evaluates the postfix `?` operator (the second of the two genuinely non-Go
// runtime mechanics) as non-local early return — on Result.Err / Option.None it
// unwinds to the enclosing function's own error / none return; on Result.Ok /
// Option.Some it yields the unwrapped value and execution continues. Closed-E
// `from` conversions are applied to the error during propagation.
//
// The 05-question-prop / 06-error-e corpus fixtures return Ok / None
// unconditionally, so they never exercise the Err / None / conversion branches.
// These tests therefore use programs MODELED ON those shapes but adapted so each
// branch actually fires. They drive real parsed+resolved goal programs through
// newInterp (call_test.go) and the standard direct-evalExpr seam. stdlib testing
// only; no testify.

import (
	"strings"
	"testing"

	"goal/internal/ast"
)

// 05-question-prop open-E shape, adapted so readFile can fail and loadConfig has
// a post-`?` statement that only runs when the `?` does NOT early-return.
const questionResultProgram = `package config

import "errors"

type Config struct {
	Raw string
}

func readFile(s string) Result[Config, error] {
	if s == "" {
		return Result.Err(errors.New("missing file"))
	}
	return Result.Ok(Config{Raw: s})
}

func loadConfig(s string) Result[Config, error] {
	cfg := readFile(s)?
	return Result.Ok(Config{Raw: cfg.Raw + "!"})
}

func main() {}
`

// 05-question-prop Option shape, adapted so find can return None and grandparent
// has a post-`?` statement.
const questionOptionProgram = `package users

type User struct {
	Name string
}

func find(name string) Option[User] {
	if name == "" {
		return Option.None
	}
	return Option.Some(User{Name: name})
}

func grandparent(name string) Option[User] {
	u := find(name)?
	return Option.Some(User{Name: u.Name + "-gp"})
}

func main() {}
`

// 06-error-e `from` shape (qclosed_prop_from), adapted so parse genuinely errs so
// the ParseError -> AppError `from func` conversion actually fires during `?`.
const questionClosedFromProgram = `package app

enum ParseError {
	Empty
}

enum AppError {
	Wrapped { cause: ParseError }
}

type Config struct {
	Raw string
}

func parse(s string) Result[Config, ParseError] {
	if s == "" {
		return Result.Err(ParseError.Empty)
	}
	return Result.Ok(Config{Raw: s})
}

from func toApp(e ParseError) AppError {
	return AppError.Wrapped(cause: e)
}

func load(s string) Result[Config, AppError] {
	cfg := parse(s)?
	return Result.Ok(cfg)
}

func main() {}
`

// 06-error-e same-E shape (qclosed_prop_same): callee and caller share the error
// type, so `?` propagates the Err unchanged (no conversion).
const questionClosedSameProgram = `package parse

enum ParseError {
	Empty
	BadKey { key: string }
}

type Config struct {
	Raw string
}

func parse(s string) Result[Config, ParseError] {
	if s == "" {
		return Result.Err(ParseError.Empty)
	}
	return Result.Ok(Config{Raw: s})
}

func loadFirst(a string) Result[Config, ParseError] {
	cfg := parse(a)?
	return Result.Ok(cfg)
}

func main() {}
`

// TestQuestionResultOkContinues: `?` on Result.Ok yields the unwrapped Config and
// the post-`?` statement runs, producing Result.Ok with the transformed payload.
func TestQuestionResultOkContinues(t *testing.T) {
	ip := newInterp(t, questionResultProgram)
	got, err := ip.evalExpr(call("loadConfig", strArg("cfg")), ip.root)
	if err != nil {
		t.Fatalf("loadConfig(\"cfg\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil || got.Variant.TypeID != "Result" || got.Variant.Tag != "Ok" {
		t.Fatalf("loadConfig(\"cfg\") = %v, want Result.Ok", got)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok || payload.Kind != KindStruct || payload.Struct == nil {
		t.Fatalf("Ok payload = %v, want a Config struct", payload)
	}
	// "cfg!" proves the unwrapped value flowed into the post-`?` statement.
	if raw := payload.Struct.Fields["Raw"]; raw.Kind != KindString || raw.Str != "cfg!" {
		t.Fatalf("Ok payload Raw = %v, want \"cfg!\"", raw)
	}
}

// TestQuestionResultErrEarlyReturns: `?` on Result.Err early-returns the enclosing
// function's Result.Err — the post-`?` Result.Ok statement never runs.
func TestQuestionResultErrEarlyReturns(t *testing.T) {
	ip := newInterp(t, questionResultProgram)
	got, err := ip.evalExpr(call("loadConfig", strArg("")), ip.root)
	if err != nil {
		t.Fatalf("loadConfig(\"\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil || got.Variant.TypeID != "Result" || got.Variant.Tag != "Err" {
		t.Fatalf("loadConfig(\"\") = %v, want Result.Err (early return, the post-? Ok must not run)", got)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok || payload.Kind != KindStruct || payload.Struct == nil {
		t.Fatalf("Err payload = %v, want the propagated error value", payload)
	}
	if msg := payload.Struct.Fields["message"]; msg.Kind != KindString || msg.Str != "missing file" {
		t.Fatalf("Err payload message = %v, want \"missing file\"", msg)
	}
}

// TestQuestionOptionSomeContinues: `?` on Option.Some yields the unwrapped User
// and the post-`?` statement runs, producing Option.Some.
func TestQuestionOptionSomeContinues(t *testing.T) {
	ip := newInterp(t, questionOptionProgram)
	got, err := ip.evalExpr(call("grandparent", strArg("ann")), ip.root)
	if err != nil {
		t.Fatalf("grandparent(\"ann\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil || got.Variant.TypeID != "Option" || got.Variant.Tag != "Some" {
		t.Fatalf("grandparent(\"ann\") = %v, want Option.Some", got)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok || payload.Kind != KindStruct || payload.Struct == nil {
		t.Fatalf("Some payload = %v, want a User struct", payload)
	}
	if name := payload.Struct.Fields["Name"]; name.Kind != KindString || name.Str != "ann-gp" {
		t.Fatalf("Some payload Name = %v, want \"ann-gp\"", name)
	}
}

// TestQuestionOptionNoneEarlyReturns: `?` on Option.None early-returns the
// enclosing function's Option.None — the post-`?` Some statement never runs.
func TestQuestionOptionNoneEarlyReturns(t *testing.T) {
	ip := newInterp(t, questionOptionProgram)
	got, err := ip.evalExpr(call("grandparent", strArg("")), ip.root)
	if err != nil {
		t.Fatalf("grandparent(\"\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil || got.Variant.TypeID != "Option" || got.Variant.Tag != "None" {
		t.Fatalf("grandparent(\"\") = %v, want Option.None (early return)", got)
	}
}

// TestQuestionClosedEAppliesFromConversion: a closed-E `?` whose callee error type
// (ParseError) differs from the enclosing function's (AppError) applies the
// registered `from func` conversion (toApp) to the error during propagation.
func TestQuestionClosedEAppliesFromConversion(t *testing.T) {
	ip := newInterp(t, questionClosedFromProgram)
	got, err := ip.evalExpr(call("load", strArg("")), ip.root)
	if err != nil {
		t.Fatalf("load(\"\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil || got.Variant.Tag != "Err" {
		t.Fatalf("load(\"\") = %v, want Result.Err", got)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok || payload.Kind != KindVariant || payload.Variant == nil {
		t.Fatalf("Err payload = %v, want an AppError variant", payload)
	}
	// The conversion produced AppError.Wrapped, not the raw ParseError.
	if payload.Variant.TypeID != "AppError" || payload.Variant.Tag != "Wrapped" {
		t.Fatalf("Err payload = %s.%s, want AppError.Wrapped (from conversion applied)", payload.Variant.TypeID, payload.Variant.Tag)
	}
	cause, ok := payload.Field("cause")
	if !ok || cause.Kind != KindVariant || cause.Variant == nil {
		t.Fatalf("AppError.Wrapped cause = %v, want the wrapped ParseError", cause)
	}
	if cause.Variant.TypeID != "ParseError" || cause.Variant.Tag != "Empty" {
		t.Fatalf("cause = %s.%s, want ParseError.Empty", cause.Variant.TypeID, cause.Variant.Tag)
	}
}

// TestQuestionClosedEOkContinues: the same closed-E `from` program continues on Ok
// (the conversion path is not taken).
func TestQuestionClosedEOkContinues(t *testing.T) {
	ip := newInterp(t, questionClosedFromProgram)
	got, err := ip.evalExpr(call("load", strArg("raw")), ip.root)
	if err != nil {
		t.Fatalf("load(\"raw\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil || got.Variant.Tag != "Ok" {
		t.Fatalf("load(\"raw\") = %v, want Result.Ok", got)
	}
}

// TestQuestionClosedESameENoConversion: a closed-E `?` whose callee and caller
// share the error type propagates the Err UNCHANGED (the payload is the same
// ParseError variant, not wrapped).
func TestQuestionClosedESameENoConversion(t *testing.T) {
	ip := newInterp(t, questionClosedSameProgram)
	got, err := ip.evalExpr(call("loadFirst", strArg("")), ip.root)
	if err != nil {
		t.Fatalf("loadFirst(\"\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil || got.Variant.Tag != "Err" {
		t.Fatalf("loadFirst(\"\") = %v, want Result.Err", got)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok || payload.Kind != KindVariant || payload.Variant == nil {
		t.Fatalf("Err payload = %v, want the unchanged ParseError variant", payload)
	}
	if payload.Variant.TypeID != "ParseError" || payload.Variant.Tag != "Empty" {
		t.Fatalf("Err payload = %s.%s, want ParseError.Empty (no conversion)", payload.Variant.TypeID, payload.Variant.Tag)
	}
}

// TestQuestionOutsidePropagatingFunctionRefused: `?` on a Result inside a function
// that does not return Result is a located, descriptive refusal, never a silent
// value.
func TestQuestionOutsidePropagatingFunctionRefused(t *testing.T) {
	const program = `package config

type Config struct {
	Raw string
}

func parse(s string) Result[Config, error] {
	return Result.Ok(Config{Raw: s})
}

func bad(s string) string {
	cfg := parse(s)?
	return cfg.Raw
}

func main() {}
`
	ip := newInterp(t, program)
	_, err := ip.evalExpr(call("bad", strArg("x")), ip.root)
	if err == nil {
		t.Fatalf("bad(...) did not refuse a ? in a non-Result function")
	}
	if !strings.Contains(err.Error(), "?") {
		t.Fatalf("error = %q, want it to name the ? operator", err.Error())
	}
}

// TestQuestionOnNonVariantRefused: `?` on a value that is not a Result or Option
// variant is a located, descriptive refusal.
func TestQuestionOnNonVariantRefused(t *testing.T) {
	ip := newInterp(t, questionResultProgram)
	bad := &ast.UnwrapExpr{X: intLit("7")}
	// Evaluate inside a Result-returning function's frame so the refusal is the
	// non-variant operand, not the enclosing-function check.
	ip.fnStack = append(ip.fnStack, ip.sigFor("loadConfig"))
	defer func() { ip.fnStack = ip.fnStack[:len(ip.fnStack)-1] }()
	_, err := ip.evalExpr(bad, ip.root)
	if err == nil {
		t.Fatalf("7? did not refuse a non-variant operand")
	}
	if !strings.Contains(err.Error(), "Result or Option") {
		t.Fatalf("error = %q, want it to mention the operand is not a Result or Option", err.Error())
	}
}

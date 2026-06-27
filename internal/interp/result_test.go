package interp

// These tests prove US-015 "Eval Result as tagged union": the interpreter
// represents Result.Ok / Result.Err as the universal tagged-union Value —
// uniformly for both open-E (`Result[T, error]`) and closed-E (`Result[T, Enum]`),
// with no (T, error) optimization — and matches over them, binding the unwrapped
// Ok payload and the unwrapped Err error.
//
// Tests drive real parsed+resolved goal programs through newInterp (call_test.go)
// and the standard direct-evalExpr seam (ip.evalExpr(call(...), ip.root)). stdlib
// testing only; no testify.

import (
	"strconv"
	"strings"
	"testing"

	"goal/internal/ast"
	"goal/internal/token"
)

// strArg builds a string-literal argument expression.
func strArg(s string) ast.Expr {
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(s)}
}

// 03-result shape: open-E, Err carries a host `error` via errors.New.
const resultOpenEProgram = `package config

import (
	"errors"
	"fmt"
)

type Config struct {
	Raw string
}

func parse(s string) Result[Config, error] {
	if s == "" {
		return Result.Err(errors.New("empty input"))
	}
	return Result.Ok(Config{Raw: s})
}

func describe(s string) string {
	return match parse(s) {
		Result.Ok(cfg) => cfg.Raw
		Result.Err(e) => fmt.Sprintf("%v", e)
	}
}

func main() {}
`

// 06-error-e shape: closed-E, Err carries an enum variant (same Result encoding).
const resultClosedEProgram = `package parse

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
	if s == "?" {
		return Result.Err(ParseError.BadKey(key: "q"))
	}
	return Result.Ok(Config{Raw: s})
}

func describe(s string) string {
	return match parse(s) {
		Result.Ok(cfg) => cfg.Raw
		Result.Err(e) => match e {
			ParseError.Empty => "empty"
			ParseError.BadKey(b) => b.key
		}
	}
}

func main() {}
`

// TestResultOkConstruction: Result.Ok(x) evaluates to a tagged-union value tagged
// "Ok" under TypeID "Result", carrying the success value as its single payload.
func TestResultOkConstruction(t *testing.T) {
	ip := newInterp(t, resultOpenEProgram)
	got, err := ip.evalExpr(call("parse", strArg("hi")), ip.root)
	if err != nil {
		t.Fatalf("parse(\"hi\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil {
		t.Fatalf("parse(\"hi\") kind = %s, want variant", got.Kind)
	}
	if got.Variant.TypeID != "Result" || got.Variant.Tag != "Ok" {
		t.Fatalf("parse(\"hi\") = %s.%s, want Result.Ok", got.Variant.TypeID, got.Variant.Tag)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok {
		t.Fatalf("Result.Ok carries no single payload: %v", got)
	}
	if payload.Kind != KindStruct || payload.Struct == nil || payload.Struct.TypeID != "Config" {
		t.Fatalf("Ok payload = %v, want a Config struct", payload)
	}
	raw, ok := payload.Struct.Fields["Raw"]
	if !ok || raw.Kind != KindString || raw.Str != "hi" {
		t.Fatalf("Ok payload Raw = %v, want \"hi\"", raw)
	}
}

// TestResultErrConstruction: Result.Err(e) evaluates to a tagged-union value
// tagged "Err" under TypeID "Result", carrying the error value as its payload.
func TestResultErrConstruction(t *testing.T) {
	ip := newInterp(t, resultOpenEProgram)
	got, err := ip.evalExpr(call("parse", strArg("")), ip.root)
	if err != nil {
		t.Fatalf("parse(\"\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil {
		t.Fatalf("parse(\"\") kind = %s, want variant", got.Kind)
	}
	if got.Variant.TypeID != "Result" || got.Variant.Tag != "Err" {
		t.Fatalf("parse(\"\") = %s.%s, want Result.Err", got.Variant.TypeID, got.Variant.Tag)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok {
		t.Fatalf("Result.Err carries no single payload: %v", got)
	}
	// The host errors.New shim encodes an error as a struct carrying its message.
	if payload.Kind != KindStruct || payload.Struct == nil {
		t.Fatalf("Err payload = %v, want an error value", payload)
	}
	if msg := payload.Struct.Fields["message"]; msg.Kind != KindString || msg.Str != "empty input" {
		t.Fatalf("Err payload message = %v, want \"empty input\"", msg)
	}
}

// TestResultMatchOpenE: a value-position match over an open-E Result binds the
// unwrapped Ok payload (read as cfg.Raw) and the unwrapped Err error (rendered via
// fmt.Sprintf), proving the arm binding is the inner value/error, not the variant.
func TestResultMatchOpenE(t *testing.T) {
	ip := newInterp(t, resultOpenEProgram)
	cases := []struct {
		arg, want string
	}{
		{"hello", "hello"},  // Ok arm binds Config; cfg.Raw == "hello"
		{"", "empty input"}, // Err arm binds the error; fmt %v renders its message
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call("describe", strArg(c.arg)), ip.root)
		if err != nil {
			t.Fatalf("describe(%q): %v", c.arg, err)
		}
		if got.Kind != KindString || got.Str != c.want {
			t.Fatalf("describe(%q) = %v, want %q", c.arg, got, c.want)
		}
	}
}

// TestResultMatchClosedE: a closed-E Result (E is the enum ParseError) uses the
// SAME tagged-union representation — the Err arm binds the enum variant, which a
// nested match consumes by tag. The Ok arm binds the Config. This proves open-E
// and closed-E share one runtime encoding (no (T, error) optimization).
func TestResultMatchClosedE(t *testing.T) {
	ip := newInterp(t, resultClosedEProgram)
	cases := []struct {
		arg, want string
	}{
		{"cfg", "cfg"}, // Result.Ok(Config) -> cfg.Raw
		{"", "empty"},  // Result.Err(ParseError.Empty) -> nested match label
		{"?", "q"},     // Result.Err(ParseError.BadKey(key:"q")) -> b.key
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call("describe", strArg(c.arg)), ip.root)
		if err != nil {
			t.Fatalf("describe(%q): %v", c.arg, err)
		}
		if got.Kind != KindString || got.Str != c.want {
			t.Fatalf("describe(%q) = %v, want %q", c.arg, got, c.want)
		}
	}
}

// TestResultErrCarriesEnumValueClosedE: directly inspect a closed-E Result.Err to
// confirm its payload is the enum variant value (ParseError.Empty), not a host
// error — i.e. the Err payload is whatever value E denotes, uniformly.
func TestResultErrCarriesEnumValueClosedE(t *testing.T) {
	ip := newInterp(t, resultClosedEProgram)
	got, err := ip.evalExpr(call("parse", strArg("")), ip.root)
	if err != nil {
		t.Fatalf("parse(\"\"): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil || got.Variant.Tag != "Err" {
		t.Fatalf("parse(\"\") = %v, want Result.Err", got)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok {
		t.Fatalf("Result.Err carries no single payload: %v", got)
	}
	if payload.Kind != KindVariant || payload.Variant == nil {
		t.Fatalf("Err payload = %v, want an enum variant", payload)
	}
	if payload.Variant.TypeID != "ParseError" || payload.Variant.Tag != "Empty" {
		t.Fatalf("Err payload = %s.%s, want ParseError.Empty", payload.Variant.TypeID, payload.Variant.Tag)
	}
}

// TestResultUnknownCtorIsRefused: a Result constructor other than Ok/Err is a
// located, descriptive error rather than a silent value.
func TestResultUnknownCtorIsRefused(t *testing.T) {
	ip := newInterp(t, resultOpenEProgram)
	bad := &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: &ast.Ident{Name: "Result"}, Sel: &ast.Ident{Name: "Maybe"}},
		Args: []ast.Expr{strArg("x")},
	}
	_, err := ip.evalExpr(bad, ip.root)
	if err == nil {
		t.Fatalf("Result.Maybe(...) did not error")
	}
	if !strings.Contains(err.Error(), "unknown Result constructor") || !strings.Contains(err.Error(), "Maybe") {
		t.Fatalf("error = %q, want it to name the unknown constructor Maybe", err.Error())
	}
}

// TestResultCtorArityIsRefused: Result.Ok / Result.Err with other than one
// argument is a located, descriptive error.
func TestResultCtorArityIsRefused(t *testing.T) {
	ip := newInterp(t, resultOpenEProgram)
	bad := &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: &ast.Ident{Name: "Result"}, Sel: &ast.Ident{Name: "Ok"}},
		Args: nil, // zero args
	}
	_, err := ip.evalExpr(bad, ip.root)
	if err == nil {
		t.Fatalf("Result.Ok() with no args did not error")
	}
	if !strings.Contains(err.Error(), "expects 1 argument") {
		t.Fatalf("error = %q, want an arity complaint", err.Error())
	}
}

package sema

import (
	"strings"
	"testing"

	"goal/internal/parser"
)

func analyzeQuestion(t *testing.T, src string) []Diagnostic {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return CheckQuestion(file, Resolve(file))
}

func analyzeClosed(t *testing.T, src string) []Diagnostic {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return CheckClosed(file, Resolve(file))
}

func TestQuestionOpenEResultCalleeClean(t *testing.T) {
	const src = `package p

func parse(s string) Result[Config, error] {
	return Result.Ok(Config{})
}

func questionUse(input string) Result[Config, error] {
	cfg := parse(input)?
	return Result.Ok(cfg)
}
`
	if d := analyzeQuestion(t, src); len(d) != 0 {
		t.Fatalf("open-E `?` over a Result callee should be clean, got: %+v", d)
	}
}

func TestClosedMissingFrom(t *testing.T) {
	const src = `package app

enum ParseError { Empty }
enum AppError { Boom }

func parse(s string) Result[Config, ParseError] {
	return Result.Ok(Config{})
}

func load(s string) Result[Config, AppError] {
	cfg := parse(s)?
	return Result.Ok(cfg)
}
`
	d := analyzeClosed(t, src)
	if len(d) != 1 || d[0].Code != "missing-from-conversion" {
		t.Fatalf("want 1 missing-from-conversion, got: %+v", d)
	}
	if !strings.Contains(d[0].Message, "no `from func` converts `ParseError` to `AppError`") {
		t.Errorf("message mismatch: %q", d[0].Message)
	}
}

func TestClosedFromPresentClean(t *testing.T) {
	const src = `package app

enum ParseError { Empty }
enum AppError { Wrapped { cause: ParseError } }

func parse(s string) Result[Config, ParseError] {
	return Result.Ok(Config{})
}

from func toApp(e ParseError) AppError {
	return AppError.Wrapped(cause: e)
}

func load(s string) Result[Config, AppError] {
	cfg := parse(s)?
	return Result.Ok(cfg)
}
`
	if d := analyzeClosed(t, src); len(d) != 0 {
		t.Fatalf("registered from func makes From-totality hold, got: %+v", d)
	}
}

func TestClosedSameEnumClean(t *testing.T) {
	const src = `package parse

enum ParseError { Empty }

func parse(s string) Result[Config, ParseError] {
	if s == "" {
		return Result.Err(ParseError.Empty)
	}
	return Result.Ok(Config{})
}

func loadFirst(a string) Result[Config, ParseError] {
	cfg := parse(a)?
	return Result.Ok(cfg)
}
`
	if d := analyzeClosed(t, src); len(d) != 0 {
		t.Fatalf("same error enum needs no conversion and Err is a known variant, got: %+v", d)
	}
}

func TestClosedUnknownCalleeDefers(t *testing.T) {
	const src = `package app

enum AppError { Boom }

func load(s string) Result[Config, AppError] {
	cfg := external(s)?
	return Result.Ok(cfg)
}
`
	d := analyzeClosed(t, src)
	if len(d) != 1 || SeverityLabel(d[0].Severity) != "warning" || d[0].Code != "unresolved-question-error" {
		t.Fatalf("unknown callee should defer with a Warning, got: %+v", d)
	}
}

func TestClosedErrOutsideEnum(t *testing.T) {
	const src = `package app

enum ParseError { Empty }
enum OtherError { Nope }

func parse(s string) Result[Config, ParseError] {
	if s == "" {
		return Result.Err(OtherError.Nope)
	}
	return Result.Ok(Config{})
}
`
	d := analyzeClosed(t, src)
	if len(d) != 1 || d[0].Code != "err-outside-closed-enum" {
		t.Fatalf("want 1 err-outside-closed-enum, got: %+v", d)
	}
	if !strings.Contains(d[0].Message, "escapes the closed error type") {
		t.Errorf("message mismatch: %q", d[0].Message)
	}
}

func TestClosedUnknownVariant(t *testing.T) {
	const src = `package app

enum ParseError {
	Empty
	BadKey { key: string }
}

func parse(s string) Result[Config, ParseError] {
	if s == "" {
		return Result.Err(ParseError.Missing)
	}
	return Result.Ok(Config{})
}
`
	d := analyzeClosed(t, src)
	if len(d) != 1 || d[0].Code != "unknown-error-variant" {
		t.Fatalf("want 1 unknown-error-variant, got: %+v", d)
	}
	if !strings.Contains(d[0].Message, "not a variant of enum `ParseError`") {
		t.Errorf("message mismatch: %q", d[0].Message)
	}
}

func TestClosedErrBoundValueDefers(t *testing.T) {
	const src = `package app

enum ParseError { Empty }

func rewrap(e ParseError) Result[Config, ParseError] {
	return Result.Err(e)
}
`
	d := analyzeClosed(t, src)
	if len(d) != 1 || SeverityLabel(d[0].Severity) != "warning" || d[0].Code != "unresolved-err-value" {
		t.Fatalf("a bound-variable Err arg should defer with a Warning, got: %+v", d)
	}
}

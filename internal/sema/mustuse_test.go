package sema

import (
	"strings"
	"testing"

	"goal/internal/parser"
)

// analyzeMustUse parses src, resolves it, and returns the must-use diagnostics.
func analyzeMustUse(t *testing.T, src string) []Diagnostic {
	t.Helper()
	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return CheckMustUse(file, Resolve(file))
}

func TestMustUseDroppedResult(t *testing.T) {
	const src = `package p

func parse(s string) Result[Config, error] {
	return Result.Ok(Config{})
}

func run(input string) {
	parse(input)
}
`
	d := analyzeMustUse(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if d[0].Severity != Error || d[0].Code != "dropped-result" {
		t.Errorf("want Error dropped-result, got %v %q", d[0].Severity, d[0].Code)
	}
	if !strings.Contains(d[0].Message, "the `Result` returned by `parse(…)` is dropped") {
		t.Errorf("message mismatch: %q", d[0].Message)
	}
}

func TestMustUseConsumedFormsClean(t *testing.T) {
	const src = `package p

func parse(s string) Result[Config, error] {
	return Result.Ok(Config{})
}

func bindUse(input string) {
	r := parse(input)
	_ = r
}

func returnUse(input string) Result[Config, error] {
	return parse(input)
}

func questionUse(input string) Result[Config, error] {
	cfg := parse(input)?
	return Result.Ok(cfg)
}

func argUse(input string) {
	consume(parse(input))
}
`
	if d := analyzeMustUse(t, src); len(d) != 0 {
		t.Fatalf("consumed Result forms should be clean, got: %+v", d)
	}
}

func TestMustUseUnderscoreDiscardDefers(t *testing.T) {
	const src = `package p

func parse(s string) Result[Config, error] {
	return Result.Ok(Config{})
}

func discard(input string) {
	_ := parse(input)
}
`
	d := analyzeMustUse(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if d[0].Severity != Warning || d[0].Code != "unresolved-result-discard" {
		t.Errorf("want Warning unresolved-result-discard, got %v %q", d[0].Severity, d[0].Code)
	}
}

func TestMustUsePlainCallIsClean(t *testing.T) {
	const src = `package p

func log(msg string) {}

func emit() {
	log("hello")
}
`
	if d := analyzeMustUse(t, src); len(d) != 0 {
		t.Fatalf("dropping a void (ModeNone) call is ordinary Go, got: %+v", d)
	}
}

func TestMustUseClosedResultDropped(t *testing.T) {
	const src = `package p

enum ParseError { Empty }

func load(s string) Result[Config, ParseError] {
	return Result.Ok(Config{})
}

func boot(input string) {
	load(input)
}
`
	d := analyzeMustUse(t, src)
	if len(d) != 1 || d[0].Code != "dropped-result" {
		t.Fatalf("closed-E Result drop should fire dropped-result, got: %+v", d)
	}
}

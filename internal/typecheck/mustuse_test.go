package typecheck

import (
	"strings"
	"testing"

	"goal/internal/sema"
)

func diagsMustUse(t *testing.T, src string) []Diagnostic {
	t.Helper()
	p, err := Load(pkgOf(map[string]string{"x.goal": src}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return CheckMustUse(p)
}

// hasCode reports whether some diagnostic has the given code, and returns it.
func hasCode(diags []Diagnostic, code string) (Diagnostic, bool) {
	for _, d := range diags {
		if d.Code == code {
			return d, true
		}
	}
	return Diagnostic{}, false
}

// --- discarded-result-error (open-E Result, error dropped with `_`) ---

func TestMustUseDiscardedError(t *testing.T) {
	const src = `package demo

func parse(s string) Result[int, error] {
    return Result.Ok(1)
}

func use() {
    v, _ := parse("x")
    _ = v
}
`
	d, ok := hasCode(diagsMustUse(t, src), "discarded-result-error")
	if !ok {
		t.Fatalf("want discarded-result-error, got %v", diagsMustUse(t, src))
	}
	if !strings.Contains(d.Message, "parse") {
		t.Errorf("message should name parse: %q", d.Message)
	}
	if !strings.HasSuffix(d.Pos.Filename, ".goal") {
		t.Errorf("diagnostic not located in .goal: %s", d.Pos)
	}
}

// The whole-Result discard via `=` (`_, _ = f()`, which `:=` would reject as "no new
// variables") still drops the error.
func TestMustUseDiscardWholeResult(t *testing.T) {
	const src = `package demo

func parse(s string) Result[int, error] {
    return Result.Ok(1)
}

func use() {
    _, _ = parse("x")
}
`
	if _, ok := hasCode(diagsMustUse(t, src), "discarded-result-error"); !ok {
		t.Fatalf("want discarded-result-error, got %v", diagsMustUse(t, src))
	}
}

// Negative: binding the error and consulting it is the correct handling — no Error.
func TestMustUseErrorConsultedClean(t *testing.T) {
	const src = `package demo

func parse(s string) Result[int, error] {
    return Result.Ok(1)
}

func use() {
    v, err := parse("x")
    _ = v
    _ = err
}
`
	if d := diagsMustUse(t, src); len(d) != 0 {
		t.Errorf("clean error handling wrongly flagged: %v", d)
	}
}

// Negative: discarding the success value while consulting the error is allowed — the
// must-use guarantee protects the failure channel, not the success value.
func TestMustUseDiscardValueKeepErrorClean(t *testing.T) {
	const src = `package demo

func parse(s string) Result[int, error] {
    return Result.Ok(1)
}

func use() {
    _, err := parse("x")
    _ = err
}
`
	if d := diagsMustUse(t, src); len(d) != 0 {
		t.Errorf("value-discard-error-kept wrongly flagged: %v", d)
	}
}

// Negative: a plain (non-Result) function whose error is discarded is not goal's
// must-use construct, so it is not flagged.
func TestMustUsePlainGoNotFlagged(t *testing.T) {
	const src = `package demo

func raw(s string) (int, error) {
    return 1, nil
}

func use() {
    v, _ := raw("x")
    _ = v
}
`
	if d := diagsMustUse(t, src); len(d) != 0 {
		t.Errorf("plain (int, error) discard wrongly flagged as must-use: %v", d)
	}
}

// --- dropped-stored-result (Result/Option field never consulted) ---

func TestMustUseDroppedOptionField(t *testing.T) {
	const src = `package demo

type cache struct {
    last Option[int]
}

func find(k string) Option[int] {
    return Option.None
}

func use() {
    c := cache{last: find("x")}
    _ = c
}
`
	d, ok := hasCode(diagsMustUse(t, src), "dropped-stored-result")
	if !ok {
		t.Fatalf("want dropped-stored-result, got %v", diagsMustUse(t, src))
	}
	if !strings.Contains(d.Message, "cache.last") || !strings.Contains(d.Message, "Option") {
		t.Errorf("message should name cache.last and Option: %q", d.Message)
	}
	if !strings.HasSuffix(d.Pos.Filename, ".goal") {
		t.Errorf("diagnostic not located in .goal: %s", d.Pos)
	}
}

func TestMustUseDroppedClosedResultField(t *testing.T) {
	const src = `package demo

enum DBErr {
    NotFound
    Timeout
}

type query struct {
    result Result[int, DBErr]
}

func get(k string) Result[int, DBErr] {
    return Result.Ok(1)
}

func use() {
    q := query{result: get("x")}
    _ = q
}
`
	d, ok := hasCode(diagsMustUse(t, src), "dropped-stored-result")
	if !ok {
		t.Fatalf("want dropped-stored-result, got %v", diagsMustUse(t, src))
	}
	if !strings.Contains(d.Message, "query.result") || !strings.Contains(d.Message, "Result") {
		t.Errorf("message should name query.result and Result: %q", d.Message)
	}
}

// Negative: a stored Option that IS read via a selector is consulted — no Error.
func TestMustUseStoredFieldReadClean(t *testing.T) {
	const src = `package demo

type cache struct {
    last Option[int]
}

func find(k string) Option[int] {
    return Option.None
}

func use() {
    c := cache{last: find("x")}
    _ = c.last
}
`
	if d := diagsMustUse(t, src); len(d) != 0 {
		t.Errorf("read field wrongly flagged: %v", d)
	}
}

// Deferral: an exported never-read-in-package must-use field cannot be proven dropped
// (another package may read it) — an honest Warning, never an Error.
func TestMustUseExportedFieldDeferred(t *testing.T) {
	const src = `package demo

type Cache struct {
    Last Option[int]
}

func find(k string) Option[int] {
    return Option.None
}

func use() {
    c := Cache{Last: find("x")}
    _ = c
}
`
	d := diagsMustUse(t, src)
	w, ok := hasCode(d, "unresolved-dropped-field")
	if !ok {
		t.Fatalf("want unresolved-dropped-field warning, got %v", d)
	}
	if sema.SeverityLabel(w.Severity) != "warning" {
		t.Errorf("exported dropped field should be a Warning, got %v", w.Severity)
	}
	if _, isErr := hasCode(d, "dropped-stored-result"); isErr {
		t.Errorf("exported field must not be a hard Error: %v", d)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// jsonDiagWire mirrors the `goal check --json` diagnostic shape (US-009 + US-030's
// suggestedFix) for decoding in tests.
type jsonDiagWire struct {
	File         string `json:"file"`
	Line         int    `json:"line"`
	Col          int    `json:"col"`
	Severity     string `json:"severity"`
	Code         string `json:"code"`
	Message      string `json:"message"`
	SuggestedFix *struct {
		Line    int    `json:"line"`
		Col     int    `json:"col"`
		NewText string `json:"newText"`
	} `json:"suggestedFix"`
}

// checkJSON runs `goal check --json <dir>` and decodes the diagnostic array.
func checkJSON(t *testing.T, dir string) []jsonDiagWire {
	t.Helper()
	var out, errOut bytes.Buffer
	// A diagnostic case returns a non-nil error (checker errors), but stdout still holds
	// the JSON array; a clean case returns nil and an empty array. Either way, decode out.
	_ = run([]string{"check", "--json", dir}, &out, &errOut)
	var diags []jsonDiagWire
	if err := json.Unmarshal(out.Bytes(), &diags); err != nil {
		t.Fatalf("check --json did not emit valid JSON: %v\nstdout: %s\nstderr: %s", err, out.String(), errOut.String())
	}
	return diags
}

// applyInsertion inserts text at the 1-based (line, col) source position, treating col as
// a byte column (the fixtures are ASCII). It mirrors how a consumer applies a suggestedFix.
func applyInsertion(src string, line, col int, text string) string {
	lines := strings.Split(src, "\n")
	off := 0
	for i := 0; i < line-1 && i < len(lines); i++ {
		off += len(lines[i]) + 1 // +1 for the stripped '\n'
	}
	off += col - 1
	if off < 0 || off > len(src) {
		return src
	}
	return src[:off] + text + src[off:]
}

// assertCheckClean runs plain `goal check <dir>` and fails unless it reports ok.
func assertCheckClean(t *testing.T, dir string) {
	t.Helper()
	var out, errOut bytes.Buffer
	if err := run([]string{"check", dir}, &out, &errOut); err != nil {
		t.Fatalf("expected clean check after applying the fix, got error: %v\nstdout: %s\nstderr: %s", err, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "ok") {
		t.Fatalf("expected `ok` after applying the fix, stdout: %s\nstderr: %s", out.String(), errOut.String())
	}
}

const nonExhaustiveGoal = `package demo

enum Color {
	Red
	Green
	Blue
}

func red()   {}
func green() {}

func handle(c Color) {
	match c {
		Color.Red => red()
		Color.Green => green()
	}
}
`

// A non-exhaustive-match diagnostic carries a suggestedFix; applying its insertion makes
// the package check clean (US-030 AC1).
func TestSuggestedFixExhaustiveMatch(t *testing.T) {
	dir := goalModule(t, map[string]string{"color.goal": nonExhaustiveGoal})

	diags := checkJSON(t, dir)
	var fix *jsonDiagWire
	for i := range diags {
		if diags[i].Code == "non-exhaustive-match" {
			fix = &diags[i]
		}
	}
	if fix == nil {
		t.Fatalf("no non-exhaustive-match diagnostic: %+v", diags)
	}
	if fix.SuggestedFix == nil {
		t.Fatal("non-exhaustive-match diagnostic carried no suggestedFix")
	}
	if !strings.Contains(fix.SuggestedFix.NewText, "Color.Blue") {
		t.Errorf("suggestedFix does not add the missing variant: %q", fix.SuggestedFix.NewText)
	}

	path := filepath.Join(dir, "color.goal")
	patched := applyInsertion(nonExhaustiveGoal, fix.SuggestedFix.Line, fix.SuggestedFix.Col, fix.SuggestedFix.NewText)
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCheckClean(t, dir)
}

const safeOmissionGoal = `package demo

type Point struct {
	X int
	Y int
}

func mk() Point {
	return Point{X: 1}
}
`

const unsafeOmissionGoal = `package demo

type Cache struct {
	name    string
	entries map[string]int
}

func mk() Cache {
	return Cache{name: "c"}
}
`

// The feature-08 lexical check is safety-only (US-001): omitting a field whose zero is
// SAFE (here `Y int`) is accepted with exit 0, while omitting a field whose zero is
// UNSAFE (a nil map that panics on write) is a located `[unsafe-zero]` error carrying no
// suggested fix. The `...defaults` unsafe path is covered separately by
// TestSuggestedFixOmittedForUnrepairable.
func TestSafetyOnlyFieldCheck(t *testing.T) {
	// (a) A struct literal omitting only a safe-zero field checks clean.
	safeDir := goalModule(t, map[string]string{"point.goal": safeOmissionGoal})
	assertCheckClean(t, safeDir)

	// (b) A struct literal omitting an unsafe-zero (nil map) field is rejected with a
	//     located [unsafe-zero] error and no suggested fix.
	unsafeDir := goalModule(t, map[string]string{"cache.goal": unsafeOmissionGoal})
	diags := checkJSON(t, unsafeDir)
	var uz *jsonDiagWire
	for i := range diags {
		if diags[i].Code == "unsafe-zero" {
			uz = &diags[i]
		}
	}
	if uz == nil {
		t.Fatalf("no unsafe-zero diagnostic for an omitted nil-map field: %+v", diags)
	}
	if !strings.Contains(uz.Message, "entries") {
		t.Errorf("unsafe-zero diagnostic does not name the omitted field: %q", uz.Message)
	}
	if uz.SuggestedFix != nil {
		t.Errorf("unsafe-zero diagnostic unexpectedly carried a suggestedFix: %+v", uz.SuggestedFix)
	}
}

const unsafeDefaultGoal = `package demo

type Box struct {
	p *int
}

func mk() Box {
	return Box{...defaults}
}
`

// A diagnostic with no known repair omits the suggestedFix field entirely (US-030 AC3).
func TestSuggestedFixOmittedForUnrepairable(t *testing.T) {
	dir := goalModule(t, map[string]string{"box.goal": unsafeDefaultGoal})

	diags := checkJSON(t, dir)
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic")
	}
	for _, d := range diags {
		if d.SuggestedFix != nil {
			t.Errorf("diagnostic %q unexpectedly carried a suggestedFix: %+v", d.Code, d.SuggestedFix)
		}
	}
}

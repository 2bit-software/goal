package pass

import (
	"strings"
	"testing"

	"goal/internal/analyze"
)

const closedSrc = `package demo

enum E {
    Bad
}

func f(x int) Result[int, E] {
    if x < 0 {
        return Result.Err(E.Bad)
    }
    return Result.Ok(x)
}
`

const preludeMarker = "type Result[T, E any] interface"

func TestNeedsResultPrelude(t *testing.T) {
	if !NeedsResultPrelude(analyze.Build(closedSrc)) {
		t.Error("closed-E program should need the prelude")
	}
	if NeedsResultPrelude(analyze.Build("package demo\n\nfunc g() int { return 0 }\n")) {
		t.Error("plain program should not need the prelude")
	}
}

func TestResultClosedInjectsPreludeInline(t *testing.T) {
	tables := analyze.Build(closedSrc)
	out, err := ResultClosed(closedSrc, tables)
	if err != nil {
		t.Fatalf("ResultClosed: %v", err)
	}
	if !strings.Contains(out, preludeMarker) {
		t.Error("single-file mode must inject the prelude inline")
	}
	if !strings.Contains(out, "Ok[int, E]{Value:") {
		t.Error("expected the closed-E constructor rewrite")
	}
}

func TestResultClosedSuppressedPrelude(t *testing.T) {
	tables := analyze.Build(closedSrc)
	tables.SuppressResultPrelude = true
	out, err := ResultClosed(closedSrc, tables)
	if err != nil {
		t.Fatalf("ResultClosed: %v", err)
	}
	if strings.Contains(out, preludeMarker) {
		t.Error("suppressed mode must NOT inject the prelude inline")
	}
	// The lowering itself still happens — only the prelude emission is suppressed.
	if !strings.Contains(out, "Ok[int, E]{Value:") {
		t.Error("suppressing the prelude must not suppress the constructor rewrite")
	}
}

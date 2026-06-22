package typecheck

import (
	"strings"
	"testing"

	"goal/internal/check"
)

func diagsNoZero(t *testing.T, src string) []Diagnostic {
	t.Helper()
	p, err := Load(pkgOf(map[string]string{"x.goal": src}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return CheckNoZeroValue(p)
}

const innerDecl = `package demo

type Inner struct {
    a int
    b int
}

`

// --- positive: elided element literals omitting a field, resolved via go/types ---

// In a slice literal the element type is elided; go/types infers `Inner`, so the omitted
// `b` is a feature-08 violation the lexical scan cannot see (it misreads `Inner{` and
// reports the wrong field set). The depth check reports the field-accurate set.
func TestNoZeroSliceElement(t *testing.T) {
	src := innerDecl + `func f() []Inner {
    return []Inner{{a: 1}}
}
`
	d, ok := hasCode(diagsNoZero(t, src), "elided-missing-field")
	if !ok {
		t.Fatalf("want elided-missing-field, got %v", diagsNoZero(t, src))
	}
	if !strings.Contains(d.Message, "`b`") || strings.Contains(d.Message, "`a`") {
		t.Errorf("message should name only the omitted field `b`: %q", d.Message)
	}
	if !strings.Contains(d.Message, "Inner") {
		t.Errorf("message should name the inferred type Inner: %q", d.Message)
	}
	if !strings.HasSuffix(d.Pos.Filename, ".goal") {
		t.Errorf("diagnostic not located in .goal: %s", d.Pos)
	}
}

func TestNoZeroMapValue(t *testing.T) {
	src := innerDecl + `func f() map[string]Inner {
    return map[string]Inner{"k": {a: 1}}
}
`
	if _, ok := hasCode(diagsNoZero(t, src), "elided-missing-field"); !ok {
		t.Fatalf("want elided-missing-field for map value, got %v", diagsNoZero(t, src))
	}
}

func TestNoZeroArrayElement(t *testing.T) {
	src := innerDecl + `func f() [2]Inner {
    return [2]Inner{{a: 1}, {a: 2, b: 3}}
}
`
	diags := diagsNoZero(t, src)
	if len(diags) != 1 {
		t.Fatalf("want exactly one diagnostic (the first element only), got %d: %v", len(diags), diags)
	}
	if diags[0].Code != "elided-missing-field" || !strings.Contains(diags[0].Message, "`b`") {
		t.Errorf("want missing-`b` on the first element, got %v", diags[0])
	}
}

// An empty elided literal `{}` omits every field.
func TestNoZeroEmptyElement(t *testing.T) {
	src := innerDecl + `func f() []Inner {
    return []Inner{{}}
}
`
	d, ok := hasCode(diagsNoZero(t, src), "elided-missing-field")
	if !ok {
		t.Fatalf("want elided-missing-field for `{}`, got %v", diagsNoZero(t, src))
	}
	if !strings.Contains(d.Message, "`a`") || !strings.Contains(d.Message, "`b`") {
		t.Errorf("empty literal should report both fields missing: %q", d.Message)
	}
}

// --- positive: generic-instantiation literal omitting a field ---

const boxDecl = `package demo

type Box[T any] struct {
    val T
    tag string
}

`

// `Box[int]{val: 1}` omits `tag`. The lexical scan keys on `IDENT {` but a `]` sits before
// the brace, so it misses this entirely (and the analyze tables don't register generic
// structs); go/types resolves `Box[int]` and reports the omission.
func TestNoZeroGenericLiteral(t *testing.T) {
	src := boxDecl + `func f() Box[int] {
    return Box[int]{val: 1}
}
`
	d, ok := hasCode(diagsNoZero(t, src), "generic-missing-field")
	if !ok {
		t.Fatalf("want generic-missing-field, got %v", diagsNoZero(t, src))
	}
	if !strings.Contains(d.Message, "`tag`") {
		t.Errorf("message should name the omitted field `tag`: %q", d.Message)
	}
	if !strings.Contains(d.Message, "Box[int]") {
		t.Errorf("message should spell the instantiated type Box[int]: %q", d.Message)
	}
	if !strings.HasSuffix(d.Pos.Filename, ".goal") {
		t.Errorf("diagnostic not located in .goal: %s", d.Pos)
	}
}

// A complete generic literal supplies every field — no diagnostic.
func TestNoZeroGenericComplete(t *testing.T) {
	src := boxDecl + `func f() Box[int] {
    return Box[int]{val: 1, tag: "x"}
}
`
	if diags := diagsNoZero(t, src); len(diags) != 0 {
		t.Fatalf("complete generic literal must produce no diagnostic, got %v", diags)
	}
}

// An injected generic prelude struct (Ok/Err, built by the Result lowering) must never be
// flagged: it is not goal-authored (no .goal declaration position). Result.Ok(1) lowers to
// a construction of the injected Ok struct, which the depth check must leave alone.
func TestNoZeroInjectedTypeNotFlagged(t *testing.T) {
	src := `package demo

func mk() Result[int, error] {
    return Result.Ok(1)
}
`
	for _, d := range diagsNoZero(t, src) {
		if d.Feature == "08-no-zero-value" {
			t.Errorf("must not flag an injected prelude struct: %v", d)
		}
	}
}

// --- negative: must NOT emit an Error ---

// A complete elided element supplies every field.
func TestNoZeroCompleteElement(t *testing.T) {
	src := innerDecl + `func f() []Inner {
    return []Inner{{a: 1, b: 2}}
}
`
	if diags := diagsNoZero(t, src); len(diags) != 0 {
		t.Fatalf("complete element must produce no diagnostic, got %v", diags)
	}
}

// A literal whose type IS named at the site is the lexical stage's job, not the depth
// stage's — the depth check skips it (cl.Type != nil) so the two stages don't double-fire.
func TestNoZeroTypedAtSiteSkipped(t *testing.T) {
	src := innerDecl + `func f() []Inner {
    return []Inner{Inner{a: 1}}
}
`
	if diags := diagsNoZero(t, src); len(diags) != 0 {
		t.Fatalf("typed-at-site literal is the lexical stage's; depth must skip, got %v", diags)
	}
}

// An elided literal of a non-struct element type (here `int`) is not a feature-08 site.
func TestNoZeroNonStructElement(t *testing.T) {
	src := `package demo

func f() []int {
    return []int{1, 2}
}
`
	if diags := diagsNoZero(t, src); len(diags) != 0 {
		t.Fatalf("non-struct element must produce no diagnostic, got %v", diags)
	}
}

// --- deferral: an unresolved elided literal is already a Go error; the depth check must
// not pile a (wrong) feature-08 Error on top of it. Struct-field-value elision is invalid
// Go ("missing type in composite literal"), so the literal's type is unresolved here. ---

func TestNoZeroUnresolvedDeferred(t *testing.T) {
	src := `package demo

type Inner struct {
    a int
    b int
}

type Outer struct {
    inner Inner
}

func f() Outer {
    return Outer{inner: {a: 1}}
}
`
	p, err := Load(pkgOf(map[string]string{"x.goal": src}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// The invalid elision is collected as a Go type error (proof it is genuinely
	// unresolved) ...
	if len(p.Errors) == 0 {
		t.Fatal("expected a collected Go error for the invalid struct-field elision")
	}
	// ... and the depth check emits no feature-08 Error of its own for it.
	for _, d := range CheckNoZeroValue(p) {
		if d.Severity == check.Error {
			t.Errorf("must not emit an Error on an unresolved elided literal: %v", d)
		}
	}
}

package backendtest

import (
	"strings"
	"testing"
)

// resultOptionMethodFixture is the goal source that declares a sum type carrying
// an Option[int]-returning and a Result[int, error]-returning value-receiver
// method (US-002). It lives beside the test so the lowered-source assertions and
// the build-and-run parity check read the same program.
const resultOptionMethodFixture = "../backend/testdata/resultoptionmethods/main.goal"

// resultOptionMethodReference is a hand-authored Go program that models the same
// behaviour with an explicit interface + type switch, Option lowered to *int and
// the Result method to the native (int, error) pair. TestResultOptionMethodBehaviour
// asserts the transpiled program's stdout matches this reference's stdout,
// covering the Some/None and Ok/Err paths.
const resultOptionMethodReference = `package main

import "fmt"

type Grade interface{ isGrade() }
type Pass struct{ score int }
type Fail struct{ reason string }

func (Pass) isGrade() {}
func (Fail) isGrade() {}

func points(g Grade) *int {
	switch v := g.(type) {
	case Pass:
		s := v.score
		return &s
	case Fail:
		return nil
	}
	panic("unreachable")
}

func validate(g Grade) (int, error) {
	switch v := g.(type) {
	case Pass:
		return v.score, nil
	case Fail:
		return 0, fmt.Errorf("failed: %s", v.reason)
	}
	panic("unreachable")
}

func main() {
	grades := []Grade{Pass{90}, Fail{"late"}}
	for _, g := range grades {
		if p := points(g); p != nil {
			fmt.Printf("points=%d\n", *p)
		} else {
			fmt.Println("points=none")
		}
		v, err := validate(g)
		if err != nil {
			fmt.Printf("validate err=%v\n", err)
		} else {
			fmt.Printf("validate ok=%d\n", v)
		}
	}
}
`

// TestResultOptionMethodLowering asserts the lowered Go for an enum whose methods
// return Option[T] and Result[T, error]: both signatures fold into the enum
// interface, each method body lives in exactly one shared free function with the
// native lowered return shape (Option -> *T, open-E Result -> (ok T, err error)),
// each variant forwards to that free function, and — critically for US-002 — the
// Result.Ok/Err constructors are lowered inside the enum-match return arms rather
// than emitted verbatim.
func TestResultOptionMethodLowering(t *testing.T) {
	goSrc := mustTranspile(t, readFixture(t, resultOptionMethodFixture))

	// The interface method set folds in both method signatures plus the marker.
	iface := interfaceBlock(goSrc, "Grade")
	if iface == "" {
		t.Fatalf("no `type Grade interface` block in lowered source:\n%s", goSrc)
	}
	for _, m := range []string{"Points() *int", "Validate() (", "isGrade()"} {
		if !strings.Contains(iface, m) {
			t.Errorf("enum interface is missing method %q; got:\n%s", m, iface)
		}
	}

	// Exactly one shared free function per method, with the native lowered result.
	if got := strings.Count(goSrc, "func Grade_Points(g Grade) *int"); got != 1 {
		t.Errorf("expected exactly one `func Grade_Points(g Grade) *int`, found %d:\n%s", got, goSrc)
	}
	// The open-E Result free function lowers to a native (ok T, err error) pair;
	// the error return name is a scope-aware gensym, so match the shape loosely.
	if !strings.Contains(goSrc, "func Grade_Validate(g Grade) (ok int, err") {
		t.Errorf("Grade_Validate must lower to native `(ok int, err ...error)` returns; got:\n%s", goSrc)
	}

	// US-002 core: no verbatim Result constructor survives in the emitted Go.
	if strings.Contains(goSrc, "Result.Ok") || strings.Contains(goSrc, "Result.Err") {
		t.Errorf("emitted Go still contains a verbatim Result.Ok/Result.Err constructor (not lowered):\n%s", goSrc)
	}
	// The Ok arm lowers to `return <value>, nil`; the None arm of the Option method
	// lowers to `return nil`.
	if !strings.Contains(goSrc, "return v1.Score, nil") {
		t.Errorf("Result.Ok arm did not lower to `return v1.Score, nil`; got:\n%s", goSrc)
	}

	// One forwarding method per variant per method, delegating to the free function.
	forwarders := map[string]string{
		"func (g Grade_Pass) Points() *int": "return Grade_Points(g)",
		"func (g Grade_Fail) Points() *int": "return Grade_Points(g)",
		"Validate() (ok int, err":           "return Grade_Validate(g)",
	}
	for sig, body := range forwarders {
		if !strings.Contains(goSrc, sig) {
			t.Errorf("expected forwarding method signature %q in lowered source:\n%s", sig, goSrc)
		}
		if !strings.Contains(goSrc, body) {
			t.Errorf("forwarding method body %q not found in lowered source:\n%s", body, goSrc)
		}
	}
}

// TestResultOptionMethodBehaviour builds and runs the transpiled fixture and
// asserts its stdout equals the hand-authored type-switch reference, exercising
// both the Some/None (Option) and Ok/Err (Result) paths.
func TestResultOptionMethodBehaviour(t *testing.T) {
	goSrc := mustTranspile(t, readFixture(t, resultOptionMethodFixture))

	got := runGoMain(t, goSrc)
	want := runGoMain(t, resultOptionMethodReference)
	if got != want {
		t.Errorf("transpiled output differs from reference type-switch:\ntranspiled:\n%s\nreference:\n%s", got, want)
	}
}

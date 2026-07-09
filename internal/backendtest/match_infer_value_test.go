package backendtest

import (
	"strings"
	"testing"

	"goal/internal/backend"
)

// A value-position `x := match` lowers to `var x T; switch { … }`, so the backend
// must know T. These tests pin that T is inferred for every arm shape whose type is
// derivable — a builtin conversion with untyped-literal siblings, stdlib/local calls
// resolved through the go/types probe, and a pass-through parameter identifier — and
// that a genuinely heterogeneous match still keeps its located deferral.

// TestInferValueMatchInt64 pins the numeric case: untyped integer-literal arms
// coerce to a sibling `int64(...)` conversion arm, so the bind infers int64.
func TestInferValueMatchInt64(t *testing.T) {
	const src = `package main

import "fmt"

enum Value {
	Str { s: string }
	Int { n: int64 }
	List { items: []Value }
}

func arity(v Value) int64 {
	x := match v {
		Value.Str(a) => 1
		Value.Int(a) => 1
		Value.List(a) => int64(len(a.items))
	}
	return x
}

func main() {
	fmt.Println(arity(Value.List(items: []Value{Value.Int(n: 1), Value.Int(n: 2)})))
}
`
	if got := strings.TrimSpace(runGoMain(t, mustTranspile(t, src))); got != "2" {
		t.Errorf("arity = %q, want %q", got, "2")
	}
}

// TestInferValueMatchStringViaProbe pins the probe case: arms that are stdlib calls
// (fmt.Sprintf / fmt.Sprint) and a local call all resolve to string — a type the AST
// alone cannot see, recovered by type-checking the probe.
func TestInferValueMatchStringViaProbe(t *testing.T) {
	const src = `package main

import "fmt"

enum Value {
	Str { s: string }
	Int { n: int64 }
	Bool { b: bool }
}

func boolStr(b bool) string {
	if b {
		return "t"
	}
	return "f"
}

func label(v Value) string {
	x := match v {
		Value.Str(a) => fmt.Sprintf("%q", a.s)
		Value.Int(a) => fmt.Sprint(a.n)
		Value.Bool(a) => boolStr(a.b)
	}
	return x
}

func main() {
	fmt.Println(label(Value.Int(n: 7)))
}
`
	if got := strings.TrimSpace(runGoMain(t, mustTranspile(t, src))); got != "7" {
		t.Errorf("label = %q, want %q", got, "7")
	}
}

// TestInferValueMatchEnumIdent pins that arms that are a local call and a bare
// pass-through parameter identifier infer the enum result type.
func TestInferValueMatchEnumIdent(t *testing.T) {
	const src = `package main

import "fmt"

enum Value {
	Str { s: string }
	Int { n: int64 }
}

func norm(v Value) Value { return v }

func swap(v Value) Value {
	x := match v {
		Value.Str(a) => norm(v)
		Value.Int(a) => v
	}
	return x
}

func main() {
	fmt.Println(swap(Value.Int(n: 3)))
}
`
	if got := strings.TrimSpace(runGoMain(t, mustTranspile(t, src))); got != "{3}" {
		t.Errorf("swap = %q, want %q", got, "{3}")
	}
}

// TestInferValueMatchDefersDisagreeingArms is the regression guard against
// over-accepting: arms of genuinely disagreeing types (string vs int64) must keep the
// located deferral rather than adopt one arm's type.
func TestInferValueMatchDefersDisagreeingArms(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"string_vs_int64", `package main

enum Value {
	Str { s: string }
	Int { n: int64 }
}

func bad(v Value) string {
	x := match v {
		Value.Str(a) => a.s
		Value.Int(a) => int64(a.n)
	}
	return x
}
`},
		{"all_untyped_literals", `package main

enum Color {
	Red
	Green
}

func pick(c Color) int {
	x := match c {
		Color.Red => 1
		Color.Green => 2
	}
	return x
}
`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := backend.Transpile(c.src)
			if err == nil {
				t.Fatalf("expected a transpile error for disagreeing/ambiguous arms, got none")
			}
			if !strings.Contains(err.Error(), "needs an inferable result type") {
				t.Errorf("want the located deferral, got: %v", err)
			}
		})
	}
}

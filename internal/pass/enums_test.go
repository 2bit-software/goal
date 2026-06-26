package pass

import (
	"strings"
	"testing"

	"goal/internal/analyze"
)

// TestEnumsNestedConstruction covers the recursive lowering of variant constructions
// nested inside another construction's payload. Pass B's outer scan resumes past the
// closing ")", so without recursion the inner construction would survive verbatim as
// invalid Go (`Enum.V(field: …)`); construct lowers each argument expression in turn.
func TestEnumsNestedConstruction(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string // substrings the lowered output must contain
	}{
		{
			name: "single nesting",
			src: "package p\n" +
				"enum Inner { Leaf { v: int } }\n" +
				"enum Outer { Node { in: Inner } }\n" +
				"func f() Outer { return Outer.Node(in: Inner.Leaf(v: 7)) }\n",
			want: []string{"Outer(Outer_Node{In: Inner(Inner_Leaf{V: 7})})"},
		},
		{
			name: "triple nesting",
			src: "package p\n" +
				"enum A { Wrap { b: B } }\n" +
				"enum B { Wrap { c: C } }\n" +
				"enum C { Leaf { v: int } }\n" +
				"func f() A { return A.Wrap(b: B.Wrap(c: C.Leaf(v: 1))) }\n",
			want: []string{"A(A_Wrap{B: B(B_Wrap{C: C(C_Leaf{V: 1})})})"},
		},
		{
			name: "nested among siblings",
			src: "package p\n" +
				"enum Inner { Leaf { v: int } }\n" +
				"enum Outer { Node { left: Inner, n: int } }\n" +
				"func f() Outer { return Outer.Node(left: Inner.Leaf(v: 2), n: 5) }\n",
			want: []string{"Outer(Outer_Node{Left: Inner(Inner_Leaf{V: 2}), N: 5})"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Enums(c.src, analyze.Build(c.src))
			if err != nil {
				t.Fatalf("Enums: %v", err)
			}
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Errorf("lowered output missing %q\n--- got ---\n%s", w, got)
				}
			}
			if strings.Contains(got, "(v:") || strings.Contains(got, "(c:") || strings.Contains(got, "(in:") {
				t.Errorf("an inner construction was left unlowered (label: form survived)\n--- got ---\n%s", got)
			}
		})
	}
}

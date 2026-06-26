package pipeline

import (
	"strings"
	"testing"
)

// The `?` lowering refuses a non-propagatable callee rather than emitting a destructure that
// will not compile: a void callee (nothing to propagate) and a callee whose last result is not
// `error` are both transpile errors, so `goal build` fails loudly at the source instead of at
// `go build` on the generated Go.
func TestQuestionLoweringRefusesNonPropagatable(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "void callee",
			src: `package x

func noth() {}

func f() Result[bool, error] {
	noth()?
	return Result.Ok(true)
}
`,
			want: "returns nothing",
		},
		{
			name: "non-error callee",
			src: `package x

func now() int { return 0 }

func f() Result[bool, error] {
	now()?
	return Result.Ok(true)
}
`,
			want: "does not return an `error`",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Transpile(c.src)
			if err == nil {
				t.Fatalf("expected a transpile error for a %s `?`, got none", c.name)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not mention %q", err.Error(), c.want)
			}
		})
	}
}

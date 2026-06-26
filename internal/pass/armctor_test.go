package pass

import (
	"strings"
	"testing"

	"goal/internal/analyze"
)

// TestOptionArmBodyConstructor covers the §8.4 lowering of `return Option.Some/None`
// inside a match arm body — the regression where an arm-body constructor was emitted
// verbatim (`undefined: Option`) because its replacement was dropped by the enclosing
// match replacement (Splice drops overlaps).
func TestOptionArmBodyConstructor(t *testing.T) {
	cases := []struct {
		name       string
		src        string
		want       []string
		notContain string
	}{
		{
			name: "identifier payload passthrough",
			src: "package p\n" +
				"func mk() Option[int] { return Option.Some(1) }\n" +
				"func f() Option[int] {\n\tmatch mk() {\n\t\tOption.Some(n) => { return Option.Some(n) }\n\t\tOption.None => { return Option.None }\n\t}\n}\n",
			want:       []string{"return &n", "return nil"},
			notContain: "Option.Some",
		},
		{
			name: "non-identifier payload is boxed",
			src: "package p\n" +
				"func mk() Option[int] { return Option.Some(1) }\n" +
				"func f() Option[int] {\n\tmatch mk() {\n\t\tOption.Some(n) => { return Option.Some(n + 1) }\n\t\tOption.None => { return Option.None }\n\t}\n}\n",
			want:       []string{":= n + 1", "return &"},
			notContain: "Option.Some",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := lowerThrough(t, c.src)
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Errorf("missing %q\n--- got ---\n%s", w, got)
				}
			}
			if c.notContain != "" && strings.Contains(got, c.notContain) {
				t.Errorf("arm-body constructor left unlowered (%q present)\n--- got ---\n%s", c.notContain, got)
			}
		})
	}
}

// TestResultArmBodyConstructor covers the §8.3 lowering of `return Result.Ok/Err` inside
// a match arm body, including that it composes with the arm payload-binding rename: the
// Ok arm's `n` must become the renamed value, not a dangling identifier.
func TestResultArmBodyConstructor(t *testing.T) {
	const src = "package p\n" +
		"func mk() Result[int, error] { return Result.Ok(7) }\n" +
		"func f() Result[int, error] {\n\tmatch mk() {\n\t\tResult.Ok(n) => { return Result.Ok(n) }\n\t\tResult.Err(e) => { return Result.Err(e) }\n\t}\n}\n"
	got := lowerThrough(t, src)
	for _, w := range []string{valName + ", nil", okName + ", " + errName} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q\n--- got ---\n%s", w, got)
		}
	}
	if strings.Contains(got, "Result.Ok") || strings.Contains(got, "Result.Err") {
		t.Errorf("arm-body constructor left unlowered\n--- got ---\n%s", got)
	}
	// The Ok payload binding `n` was renamed; a bare `n` must not survive in the body.
	if strings.Contains(got, "return n,") {
		t.Errorf("binding rename did not compose with constructor lowering\n--- got ---\n%s", got)
	}
}

// lowerThrough runs the open-E Result and Option passes in pipeline order and returns the
// combined output, the two passes responsible for arm-body constructor lowering.
func lowerThrough(t *testing.T, src string) string {
	t.Helper()
	tables := analyze.Build(src)
	out, err := Result(src, tables)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	out, err = Option(out, tables)
	if err != nil {
		t.Fatalf("Option: %v", err)
	}
	return out
}

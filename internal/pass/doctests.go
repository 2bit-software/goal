package pass

import (
	"fmt"
	"go/format"
	"strings"
)

// Doctests extracts runnable doctests (spec §4.1) from a source's `///` doc comments
// and returns a generated `_test.go` file, or "" when there are none. Unlike the
// lowering passes it does not transform the main source — `///` is a valid Go line
// comment, so the source compiles as-is. The side output is the product: turning a
// doctest into a real test guarantees it cannot silently not-run.
//
// A doctest is a `>>> <expr>` line in a `///` block followed by an expected-output
// line; it attaches to the free function declared immediately below the block. It is
// read from the original source (before any pass), so the pipeline must run it on the
// untouched input. Methods, multi-line expected output, and goscript's runner are out
// of scope.
func Doctests(src string) (string, error) {
	tests := extractDoctests(src)
	if len(tests) == 0 {
		return "", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", packageName(src))
	b.WriteString("import \"testing\"\n")
	counts := map[string]int{}
	for _, d := range tests {
		counts[d.fn]++
		fmt.Fprintf(&b, "\nfunc TestDoctest_%s_%d(t *testing.T) {\n", d.fn, counts[d.fn])
		fmt.Fprintf(&b, "\tgot := %s\n", d.expr)
		fmt.Fprintf(&b, "\twant := %s\n", d.expected)
		fmt.Fprintf(&b, "\tif got != want {\n")
		fmt.Fprintf(&b, "\t\tt.Errorf(\"doctest %s: got %%v, want %%v\", got, want)\n", d.fn)
		fmt.Fprintf(&b, "\t}\n}\n")
	}

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", fmt.Errorf("generated test file did not parse: %w\n--- generated ---\n%s", err, b.String())
	}
	return string(formatted), nil
}

// doctest is one extracted `>>> expr` / expected pair and the function it documents.
type doctest struct {
	fn       string
	expr     string
	expected string
}

// extractDoctests walks the source, finds each run of `///` doc lines, attaches it to
// the function declared just below, and parses its `>>> expr` / expected pairs.
func extractDoctests(src string) []doctest {
	lines := strings.Split(src, "\n")
	var out []doctest
	for i := 0; i < len(lines); i++ {
		if !isDocLine(lines[i]) {
			continue
		}
		j := i
		var content []string
		for j < len(lines) && isDocLine(lines[j]) {
			content = append(content, docContent(lines[j]))
			j++
		}
		if fn := funcNameBelow(lines, j); fn != "" {
			out = append(out, parseBlock(fn, content)...)
		}
		i = j
	}
	return out
}

// parseBlock turns a doc block's content lines into doctests: each `>>> expr` line
// pairs with the immediately following content line as the expected output.
func parseBlock(fn string, content []string) []doctest {
	var out []doctest
	for k := 0; k < len(content); k++ {
		rest, ok := strings.CutPrefix(content[k], ">>>")
		if !ok {
			continue
		}
		expr := strings.TrimSpace(rest)
		if expr == "" || k+1 >= len(content) {
			continue
		}
		expected := strings.TrimSpace(content[k+1])
		if expected == "" || strings.HasPrefix(expected, ">>>") {
			continue // no expected-output line — malformed, skip
		}
		out = append(out, doctest{fn: fn, expr: expr, expected: expected})
		k++ // consume the expected line
	}
	return out
}

// isDocLine reports whether a line is a `///` doc comment.
func isDocLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "///")
}

// docContent strips the leading whitespace and `///` from a doc line.
func docContent(line string) string {
	t := strings.TrimSpace(line)
	return strings.TrimSpace(strings.TrimPrefix(t, "///"))
}

// funcNameBelow returns the name of the free function declared at the next non-blank
// line at or after idx, or "" if that line is not a `func NAME(`. A leading
// declaration modifier (`from`/`derive func`) is stripped first, so doctests attach to
// modifier-prefixed functions too — a composition the standalone feature never saw.
func funcNameBelow(lines []string, idx int) string {
	for ; idx < len(lines); idx++ {
		t := strings.TrimSpace(lines[idx])
		if t == "" {
			continue
		}
		for _, mod := range []string{"from ", "derive "} {
			if after, ok := strings.CutPrefix(t, mod); ok {
				t = strings.TrimSpace(after)
				break
			}
		}
		rest, ok := strings.CutPrefix(t, "func ")
		if !ok {
			return "" // doc block not attached to a function
		}
		name, _, found := strings.Cut(strings.TrimSpace(rest), "(")
		name = strings.TrimSpace(name)
		if !found || name == "" {
			return "" // a method `func (r R) m(` — out of scope for v1
		}
		return name
	}
	return ""
}

// packageName returns the package declared in src, or "main" if none is found.
func packageName(src string) string {
	for line := range strings.SplitSeq(src, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "package "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return "main"
}

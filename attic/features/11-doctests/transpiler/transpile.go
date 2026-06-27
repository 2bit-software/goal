// Package main is a standalone reference transpiler for goal feature
// 11-doctests: runnable doctests co-located with code (§4.1). A doctest lives in
// a `///` doc comment as a `>>> <expr>` line followed by an expected-output line;
// it is EXTRACTED into a generated `_test.go` so it runs under `go test`. The hard
// design rule (§4.1): there is no way for a doctest to silently not-run — turning
// it into a real test file is exactly what guarantees that.
//
//	/// Adds two ints.
//	/// >>> add(2, 3)
//	/// 5
//	func add(a int, b int) int { return a + b }
//
// generates (§8.6):
//
//	func TestDoctest_add_1(t *testing.T) {
//	    got := add(2, 3)
//	    want := 5
//	    if got != want { t.Errorf("doctest add: got %v, want %v", got, want) }
//	}
//
// transpile() RETURNS THE GENERATED `_test.go` — that test file is this feature's
// product. The original code file needs no transformation: `///` is a valid Go
// line comment, so the source compiles as-is and is left untouched (a real build
// emits it verbatim alongside the generated `<base>_doctest_test.go`).
//
// Scope: doctests attach to the free function declared immediately below the
// `///` block; the expected output is written as a Go expression/literal (so it
// lowers to `want := <expected>`); one expected line per `>>>`. Methods, multi-line
// expected output, and goscript's own runner are out of scope. Comments are read
// straight from the source (the lexer skips them). Malformed input is UB.
package main

import (
	"fmt"
	"go/format"
	"strings"
)

// doctest is one extracted `>>> expr` / expected pair and the function it documents.
type doctest struct {
	fn       string
	expr     string
	expected string
}

// transpile extracts every doctest and returns the generated `_test.go` source.
func transpile(src string) (string, error) {
	pkg := packageName(src)
	tests := extractDoctests(src)

	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n", pkg)
	if len(tests) > 0 {
		b.WriteString("\nimport \"testing\"\n")
	}

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

// extractDoctests walks the source, finds each run of `///` doc lines, attaches it
// to the function declared just below, and parses its `>>> expr` / expected pairs.
func extractDoctests(src string) []doctest {
	lines := strings.Split(src, "\n")
	var out []doctest
	for i := 0; i < len(lines); i++ {
		if !isDocLine(lines[i]) {
			continue
		}
		// Gather the contiguous `///` block [i, j).
		j := i
		var content []string
		for j < len(lines) && isDocLine(lines[j]) {
			content = append(content, docContent(lines[j]))
			j++
		}
		fn := funcNameBelow(lines, j)
		if fn != "" {
			out = append(out, parseBlock(fn, content)...)
		}
		i = j // skip past the block
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

// funcNameBelow returns the name of the free function declared at the next
// non-blank line at or after idx, or "" if that line is not a `func NAME(`.
func funcNameBelow(lines []string, idx int) string {
	for ; idx < len(lines); idx++ {
		t := strings.TrimSpace(lines[idx])
		if t == "" {
			continue
		}
		rest, ok := strings.CutPrefix(t, "func ")
		if !ok {
			return "" // doc block not attached to a function
		}
		rest = strings.TrimSpace(rest)
		name, _, found := strings.Cut(rest, "(")
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

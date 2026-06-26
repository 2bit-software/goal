package pass

import (
	"fmt"
	"strings"
)

// Doctest is one extracted `>>> expr` / expected pair and the function it documents.
// Expr and Expected are copied verbatim from the `///` block, so they are goal
// expressions, not Go: a doctest may reference enum variants (`Enum.V(field: …)`),
// keyed struct literals (`T{field: …}`), or Result/Option constructors. They must be
// run through the lowering passes — alongside the function bodies — before they are
// valid Go. RenderDoctests emits them into a goal-shaped test source for the pipeline
// to lower; see pipeline.Transpile.
type Doctest struct {
	Fn       string
	Expr     string
	Expected string
}

// ExtractDoctests pulls runnable doctests (spec §4.1) from a source's `///` doc
// comments. A doctest is a `>>> <expr>` line in a `///` block followed by an
// expected-output line; it attaches to the free function declared immediately below the
// block. It is read from the original source (before any pass), so the pipeline must run
// it on the untouched input. Methods, multi-line expected output, and goscript's runner
// are out of scope.
func ExtractDoctests(src string) []Doctest {
	lines := strings.Split(src, "\n")
	var out []Doctest
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

// RenderDoctests emits a `_test.go`-shaped source from extracted doctests, or "" when
// there are none. The output is GOAL, not Go: each Expr/Expected is copied verbatim, so
// a doctest over goal-specific values renders here unchanged for the caller's pass
// pipeline to lower. It is deliberately not gofmt-formatted — the goal expressions are
// not yet valid Go. Turning a doctest into a real test is the product: it guarantees the
// doctest cannot silently not-run.
func RenderDoctests(src string, tests []Doctest) string {
	if len(tests) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", packageName(src))
	b.WriteString("import \"testing\"\n")
	counts := map[string]int{}
	for _, d := range tests {
		counts[d.Fn]++
		fmt.Fprintf(&b, "\nfunc TestDoctest_%s_%d(t *testing.T) {\n", d.Fn, counts[d.Fn])
		fmt.Fprintf(&b, "\tgot := %s\n", d.Expr)
		fmt.Fprintf(&b, "\twant := %s\n", d.Expected)
		fmt.Fprintf(&b, "\tif got != want {\n")
		fmt.Fprintf(&b, "\t\tt.Errorf(\"doctest %s: got %%v, want %%v\", got, want)\n", d.Fn)
		fmt.Fprintf(&b, "\t}\n}\n")
	}
	return b.String()
}

// parseBlock turns a doc block's content lines into doctests: each `>>> expr` line
// pairs with the immediately following content line as the expected output.
func parseBlock(fn string, content []string) []Doctest {
	var out []Doctest
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
		out = append(out, Doctest{Fn: fn, Expr: expr, Expected: expected})
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

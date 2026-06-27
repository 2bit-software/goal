package backend

import (
	"fmt"
	"strings"

	"goal/internal/ast"
	"goal/internal/parser"
	"goal/internal/sema"
)

// emitDoctests extracts the `///` doctests the parser attached to each
// declaration (ast.FuncDecl.Doc.Doctests, US-023) and emits the `_test.go`
// sidecar, lowered through the SAME emit path as the function bodies. It renders
// a goal-shaped test source from the structured doctests, parses it back to an
// *ast.File, and runs it through emitFile with the ORIGINAL file's resolved info
// — so a doctest over enum variants, keyed struct literals, or Result/Option
// constructors lowers to exactly the Go the documented function body would (the
// emitter's variant/selector lowering reads info.Enums by name). It returns ""
// (the un-formatted sidecar) when the file has no doctests; the caller formats.
//
// This is the AST-native analogue of the splice path's pipeline.doctestFile: it
// reads the structured Doctest nodes rather than re-lexing the source text, but
// produces the same `TestDoctest_<fn>_<n>` shape so the doctest tier (and its
// goldens) line up with the splice engine.
func emitDoctests(f *ast.File, info *sema.Info) (string, error) {
	goalTest := renderDoctests(f)
	if goalTest == "" {
		return "", nil
	}
	testFile, err := parser.ParseFile(goalTest)
	if err != nil {
		return "", fmt.Errorf("doctest sidecar parse: %w\n--- rendered ---\n%s", err, goalTest)
	}
	return emitFile(testFile, info)
}

// renderDoctests builds a goal-shaped `_test.go` source from the structured
// doctests on f's declarations, or "" when none exist. Each doctest becomes a
// `TestDoctest_<fn>_<n>` function that evaluates the input expression and
// compares it to the expected value — the input/expected text is copied verbatim
// (it is goal, not yet Go), so the construction in a doctest body lowers when the
// rendered source is parsed and emitted. The output is GOAL: it is not gofmt-able
// until emitDoctests lowers it. Shape mirrors internal/pass.RenderDoctests so the
// AST and splice engines produce the same sidecar.
func renderDoctests(f *ast.File) string {
	pkg := "main"
	if f != nil && f.Name != nil {
		pkg = f.Name.Name
	}

	var body strings.Builder
	counts := map[string]int{}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name == nil || fd.Doc == nil || len(fd.Doc.Doctests) == 0 {
			continue
		}
		fn := fd.Name.Name
		for _, dt := range fd.Doc.Doctests {
			expr := strings.TrimSpace(dt.Input)
			want := strings.TrimSpace(strings.Join(dt.Expected, "\n"))
			if expr == "" || want == "" {
				continue // malformed (no input or no expected line) — skip
			}
			counts[fn]++
			fmt.Fprintf(&body, "\nfunc TestDoctest_%s_%d(t *testing.T) {\n", fn, counts[fn])
			fmt.Fprintf(&body, "\tgot := %s\n", expr)
			fmt.Fprintf(&body, "\twant := %s\n", want)
			fmt.Fprintf(&body, "\tif got != want {\n")
			fmt.Fprintf(&body, "\t\tt.Errorf(\"doctest %s: got %%v, want %%v\", got, want)\n", fn)
			fmt.Fprintf(&body, "\t}\n}\n")
		}
	}
	if body.Len() == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\nimport \"testing\"\n", pkg)
	b.WriteString(body.String())
	return b.String()
}

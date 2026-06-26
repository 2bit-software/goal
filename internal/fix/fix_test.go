package fix

import (
	"strings"
	"testing"
)

// Converting a plain-Go (T, error) helper with manual propagation produces the idiomatic
// Result + `?` form, and the conversion ripples to a same-package caller.
func TestConvertTupleToResult(t *testing.T) {
	before := "package app\n\n" +
		"import \"os\"\n\n" +
		"func load(p string) ([]byte, error) {\n" +
		"\tf, err := os.ReadFile(p)\n" +
		"\tif err != nil {\n" +
		"\t\treturn nil, err\n" +
		"\t}\n" +
		"\treturn f, nil\n" +
		"}\n\n" +
		"func describe(p string) (int, error) {\n" +
		"\tdata, err := load(p)\n" +
		"\tif err != nil {\n" +
		"\t\treturn 0, err\n" +
		"\t}\n" +
		"\treturn len(data), nil\n" +
		"}\n"
	want := "package app\n\n" +
		"import \"os\"\n\n" +
		"func load(p string) Result[[]byte, error] {\n" +
		"\tf := os.ReadFile(p)?\n" +
		"\treturn Result.Ok(f)\n" +
		"}\n\n" +
		"func describe(p string) Result[int, error] {\n" +
		"\tdata := load(p)?\n" +
		"\treturn Result.Ok(len(data))\n" +
		"}\n"

	got, changes, _ := File(before)
	if got != want {
		t.Fatalf("fix mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if len(changes) == 0 {
		t.Fatal("expected changes to be recorded")
	}

	// Idempotence: fixing the fixed output changes nothing.
	again, ch2, _ := File(got)
	if again != got {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", got, again)
	}
	if len(ch2) != 0 {
		t.Fatalf("idempotent run still reported %d changes", len(ch2))
	}
}

// A function that already returns Result but still propagates manually collapses to `?`.
func TestCollapseInsideResultFunc(t *testing.T) {
	before := "package app\n\n" +
		"func describe(id string) Result[string, error] {\n" +
		"\tu, err := lookup(id)\n" +
		"\tif err != nil {\n" +
		"\t\treturn Result.Err(err)\n" +
		"\t}\n" +
		"\treturn Result.Ok(u)\n" +
		"}\n"
	want := "package app\n\n" +
		"func describe(id string) Result[string, error] {\n" +
		"\tu := lookup(id)?\n" +
		"\treturn Result.Ok(u)\n" +
		"}\n"
	got, _, _ := File(before)
	if got != want {
		t.Fatalf("collapse mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// A decorated error return blocks the signature conversion: the function is left untouched
// and reported, never half-converted.
func TestDecoratedErrorNotConverted(t *testing.T) {
	before := "package app\n\n" +
		"import \"fmt\"\n\n" +
		"func load(p string) ([]byte, error) {\n" +
		"\tf, err := read(p)\n" +
		"\tif err != nil {\n" +
		"\t\treturn nil, fmt.Errorf(\"load %s: %w\", p, err)\n" +
		"\t}\n" +
		"\treturn f, nil\n" +
		"}\n"
	got, changes, reports := File(before)
	if got != before {
		t.Fatalf("decorated error should not be converted, but source changed:\n%s", got)
	}
	if len(changes) != 0 {
		t.Fatalf("expected no changes, got %d", len(changes))
	}
	if !hasReport(reports, "result-sig") {
		t.Fatalf("expected a result-sig skip report, got %+v", reports)
	}
}

// A comment inside a propagation block prevents the collapse (the comment would be lost).
func TestCommentBlocksCollapse(t *testing.T) {
	before := "package app\n\n" +
		"func describe(id string) Result[string, error] {\n" +
		"\tu, err := lookup(id)\n" +
		"\tif err != nil {\n" +
		"\t\t// give up early\n" +
		"\t\treturn Result.Err(err)\n" +
		"\t}\n" +
		"\treturn Result.Ok(u)\n" +
		"}\n"
	got, _, reports := File(before)
	if strings.Contains(got, "?") {
		t.Fatalf("comment block should not collapse to `?`:\n%s", got)
	}
	if !hasReport(reports, "propagate") {
		t.Fatalf("expected a propagate skip report, got %+v", reports)
	}
}

// An Option function's manual nil-check collapses to `?`, with `*o` dereferences rewritten.
func TestOptionCollapse(t *testing.T) {
	before := "package app\n\n" +
		"func first(xs []int) Option[int] {\n" +
		"\to := lookup(xs)\n" +
		"\tif o == nil {\n" +
		"\t\treturn Option.None\n" +
		"\t}\n" +
		"\treturn Option.Some(*o)\n" +
		"}\n"
	want := "package app\n\n" +
		"func first(xs []int) Option[int] {\n" +
		"\to := lookup(xs)?\n" +
		"\treturn Option.Some(o)\n" +
		"}\n"
	got, _, _ := File(before)
	if got != want {
		t.Fatalf("option collapse mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// A function returning multiple non-error values is out of scope: left untouched, reported.
func TestMultiValueNotConverted(t *testing.T) {
	before := "package app\n\n" +
		"func split(s string) (string, string, error) {\n" +
		"\ta, b, err := parse(s)\n" +
		"\tif err != nil {\n" +
		"\t\treturn \"\", \"\", err\n" +
		"\t}\n" +
		"\treturn a, b, nil\n" +
		"}\n"
	got, changes, reports := File(before)
	if got != before || len(changes) != 0 {
		t.Fatalf("multi-value function should be left untouched:\n%s", got)
	}
	if !hasReport(reports, "result-sig") {
		t.Fatalf("expected a result-sig skip report, got %+v", reports)
	}
}

// A switch over an in-file enum is reported as a match candidate (detection, not rewrite).
func TestSwitchOverEnumReported(t *testing.T) {
	before := "package app\n\n" +
		"enum Status {\n\tActive\n\tDone\n}\n\n" +
		"func label(s Status) string {\n" +
		"\tswitch s {\n" +
		"\tcase Status.Active:\n\t\treturn \"a\"\n" +
		"\tcase Status.Done:\n\t\treturn \"d\"\n" +
		"\t}\n" +
		"\treturn \"\"\n" +
		"}\n"
	got, _, reports := File(before)
	if got != before {
		t.Fatalf("switch should not be rewritten, only reported:\n%s", got)
	}
	if !hasReport(reports, "match") {
		t.Fatalf("expected a match suggestion, got %+v", reports)
	}
}

func hasReport(reports []Report, rule string) bool {
	for _, r := range reports {
		if r.Rule == rule {
			return true
		}
	}
	return false
}

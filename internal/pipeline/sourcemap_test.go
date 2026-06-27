package pipeline

import (
	"strings"
	"testing"
)

func TestAddLineDirectivesAnchorsUserDecls(t *testing.T) {
	goalSrc := "package demo\n\nfunc f() int {\n\treturn 0\n}\n"
	genGo := "package demo\n\nfunc f() int {\n\treturn 0\n}\n"
	got := AddLineDirectives(goalSrc, genGo, "f.goal", "f.go")
	// f is declared on line 3 of the source; a directive must anchor it there.
	if !strings.Contains(got, "//line f.goal:3\n") {
		t.Errorf("missing source anchor for func f:\n%s", got)
	}
}

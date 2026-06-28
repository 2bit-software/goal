package typecheck

import (
	"strings"
	"testing"

	"goal/internal/project"
)

func pkgOf(files map[string]string) *project.Package {
	p := &project.Package{Dir: "demo", Name: "demo"}
	for name, src := range files {
		p.Files = append(p.Files, project.File{Path: "demo/" + name, Name: name, Src: src})
	}
	return p
}

// A package that exercises an import, an interface + implementer, and a local — the
// shapes the depth checks query.
const harnessSrc = `package demo

import "fmt"

type Speaker interface {
    Speak() string
}

type Dog struct implements Speaker {
}

func (d Dog) Speak() string {
    return "woof"
}

func greet(d Dog) string {
    msg := fmt.Sprintf("%s!", d.Speak())
    return msg
}
`

func TestLoadTypedView(t *testing.T) {
	p, err := Load(pkgOf(map[string]string{"zoo.goal": harnessSrc}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.Errors) != 0 {
		t.Fatalf("clean program had type errors: %v", p.Errors)
	}
	if p.Types == nil {
		t.Fatal("no *types.Package")
	}
	if len(p.Info.Defs) == 0 || len(p.Info.Uses) == 0 {
		t.Errorf("Info not populated: Defs=%d Uses=%d", len(p.Info.Defs), len(p.Info.Uses))
	}
	// Merged sema facts carry what goal said (so a check knows which question to ask).
	if _, ok := p.Sema.FuncSignatures["greet"]; !ok {
		t.Error("merged sema facts missing greet")
	}
	// A user declaration is found by its goal name, and its position maps to .goal.
	dog := p.Lookup("Dog")
	if dog == nil {
		t.Fatal("Dog not found in package scope")
	}
	if pos := p.Fset.Position(dog.Pos()); !strings.HasSuffix(pos.Filename, ".goal") {
		t.Errorf("Dog position %s is not in a .goal file", pos)
	}
}

func TestLoadErrorTolerant(t *testing.T) {
	// A genuine Go type error in passed-through code must not make Load fail; it should
	// surface as a collected type error with a usable (partial) package.
	const bad = "package demo\n\nfunc f() int {\n\tvar x int = \"nope\"\n\treturn x\n}\n"
	p, err := Load(pkgOf(map[string]string{"bad.goal": bad}))
	if err != nil {
		t.Fatalf("Load should tolerate type errors, got: %v", err)
	}
	if len(p.Errors) == 0 {
		t.Fatal("expected a collected type error")
	}
	// The collected error is goal-located via //line.
	if !strings.Contains(p.Errors[0].Error(), "bad.goal:4") {
		t.Errorf("type error not mapped to bad.goal:4: %v", p.Errors[0])
	}
}

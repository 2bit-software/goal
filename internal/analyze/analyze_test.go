package analyze

import "testing"

// fileA declares an enum; fileB references it in a function in the same package. With
// per-file tables fileB cannot see Shape; with package tables it must.
const fileA = `package demo

enum Shape {
    Circle { r: float64 }
    Square { s: float64 }
}
`

const fileB = `package demo

func area(sh Shape) float64 {
    return match sh {
        Shape.Circle(c) => c.r
        Shape.Square(q) => q.s
    }
}
`

func TestBuildPackageResolvesCrossFileEnum(t *testing.T) {
	// Control: file B alone does not know Shape.
	if e := Build(fileB).Enums["Shape"]; e != nil {
		t.Fatal("Build(fileB) unexpectedly knows the cross-file enum Shape")
	}

	pkg := BuildPackage([]string{fileA, fileB})
	e := pkg.Enums["Shape"]
	if e == nil {
		t.Fatal("BuildPackage did not resolve cross-file enum Shape")
	}
	if len(e.Variants) != 2 || e.Variants[0].Name != "Circle" || e.Variants[1].Name != "Square" {
		t.Errorf("Shape variants = %+v, want [Circle Square]", e.Variants)
	}
	// The union also carries file B's own facts (its function signature).
	if _, ok := pkg.FuncSignatures["area"]; !ok {
		t.Error("BuildPackage dropped area's signature from file B")
	}
}

func TestMergeUnionsDistinctTableKinds(t *testing.T) {
	a := Build("package demo\n\ntype Point struct {\n\tx int\n\ty int\n}\n")
	b := Build("package demo\n\nsealed interface Drawable {}\n")
	a.Merge(b)

	if _, ok := a.Structs["Point"]; !ok {
		t.Error("merge lost the struct Point from the receiver")
	}
	if !a.Sealed["Drawable"] {
		t.Error("merge dropped the sealed interface Drawable from the argument")
	}
}

func TestBuildPackageLastWins(t *testing.T) {
	// Two declarations of the same name: the later source wins (deterministic with a
	// stable file order). The genuine redeclaration is left for the Go compiler.
	first := "package demo\n\ntype S struct {\n\ta int\n}\n"
	second := "package demo\n\ntype S struct {\n\ta int\n\tb int\n}\n"

	pkg := BuildPackage([]string{first, second})
	if got := len(pkg.Structs["S"]); got != 2 {
		t.Errorf("last-wins merge: S has %d fields, want 2 (the second declaration)", got)
	}
}

func TestBuildPackageEmpty(t *testing.T) {
	// No sources -> initialized, empty tables (maps non-nil, safe to read/merge).
	pkg := BuildPackage(nil)
	if pkg == nil || pkg.Enums == nil || pkg.FuncSignatures == nil {
		t.Fatal("BuildPackage(nil) must return initialized empty tables")
	}
	if len(pkg.Enums) != 0 {
		t.Errorf("BuildPackage(nil) has %d enums, want 0", len(pkg.Enums))
	}
}

package project

import (
	"os"
	"path/filepath"
	"testing"
)

// write creates dir/name with the given contents under root, making parents as needed.
func write(t *testing.T, root, rel, src string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverGroupsByDirectory(t *testing.T) {
	root := t.TempDir()
	write(t, root, "main.goal", "package app\n\nfunc main() {}\n")
	write(t, root, "util.goal", "package app\n\nfunc helper() {}\n")
	write(t, root, "sub/lib.goal", "package lib\n\nfunc F() {}\n")

	pkgs, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("want 2 packages, got %d", len(pkgs))
	}
	// Sorted by Dir: root ("app") then root/sub ("lib").
	if pkgs[0].Name != "app" || len(pkgs[0].Files) != 2 {
		t.Errorf("pkg[0] = %q with %d files, want app with 2", pkgs[0].Name, len(pkgs[0].Files))
	}
	if pkgs[1].Name != "lib" || len(pkgs[1].Files) != 1 {
		t.Errorf("pkg[1] = %q with %d files, want lib with 1", pkgs[1].Name, len(pkgs[1].Files))
	}
	// Files within a package are sorted by path.
	if pkgs[0].Files[0].Name != "main.goal" || pkgs[0].Files[1].Name != "util.goal" {
		t.Errorf("files not sorted: %q, %q", pkgs[0].Files[0].Name, pkgs[0].Files[1].Name)
	}
	if pkgs[0].Files[0].Src == "" {
		t.Error("file source not read")
	}
}

func TestDiscoverConflictingPackageNames(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.goal", "package app\n")
	write(t, root, "b.goal", "package other\n")

	if _, err := Discover(root); err == nil {
		t.Fatal("want error for two package names in one directory, got nil")
	}
}

func TestDiscoverMissingPackageClause(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.goal", "func main() {}\n") // no package clause

	if _, err := Discover(root); err == nil {
		t.Fatal("want error for missing package clause, got nil")
	}
}

func TestDiscoverSkipsReservedDirs(t *testing.T) {
	root := t.TempDir()
	write(t, root, "main.goal", "package app\n")
	write(t, root, "testdata/x.goal", "package ignored\n")
	write(t, root, ".hidden/y.goal", "package ignored\n")
	write(t, root, "_cut/z.goal", "package ignored\n")

	pkgs, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "app" {
		t.Fatalf("want only the app package, got %d packages", len(pkgs))
	}
}

func TestDiscoverEmptyTreeYieldsNoPackages(t *testing.T) {
	root := t.TempDir()
	write(t, root, "README.md", "not goal\n")

	pkgs, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("want 0 packages, got %d", len(pkgs))
	}
}

func TestPackageClauseIgnoresStringsAndComments(t *testing.T) {
	// A `package` word inside a comment or string must not be read as the clause.
	src := "// package fake is not the clause\npackage real\n\nvar s = \"package alsofake\"\n"
	if got := PackageClause(src); got != "real" {
		t.Errorf("PackageClause = %q, want real", got)
	}
	if got := PackageClause("func main() {}\n"); got != "" {
		t.Errorf("PackageClause with no clause = %q, want empty", got)
	}
}

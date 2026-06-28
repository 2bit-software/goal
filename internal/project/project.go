// Package project models a goal project as packages of source files — the unit the
// build model transpiles and type-checks together (Phase A of BUILD-MODEL-TODO).
//
// A package is one directory of `.goal` files that share a `package` clause, mirroring
// Go's one-package-per-directory rule. This is the discovery layer beneath the
// cross-file table merge (U2) and the package transpile driver (U4): it locates and
// groups source, reads each file's package name, and does no lowering itself.
//
// Discovery is deliberately offset-free and name-oriented, consistent with the rest of
// the front-end: a File carries its raw source; a Package carries its shared name and
// its files. Everything downstream keys off names, not the order files were found.
package project

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"goal/internal/scan"
	"goal/internal/textedit"
)

// Ext is the goal source-file extension.
const Ext = ".goal"

// File is one `.goal` source file: its path, base name, and contents. Src is read at
// discovery time so later phases need not re-read the disk.
type File struct {
	Path string // path as discovered (relative to the walk root or absolute)
	Name string // base name, e.g. "main.goal"
	Src  string // file contents
}

// Package is a directory of goal files sharing one `package` clause — the granularity
// the build model transpiles and type-checks as a unit. Files is sorted by path for
// deterministic output.
type Package struct {
	Dir   string // directory holding the package
	Name  string // the `package` name every file in Dir declares
	Files []File // the package's .goal files, sorted by Path
}

// Discover walks root recursively (the `./...` sense), reads every `.goal` file, and
// groups them into packages by directory. It returns the packages sorted by directory.
//
// One directory is one package: every file in it must declare the same `package` name,
// or Discover returns an error naming the conflict (the same rule `go build` enforces).
// Hidden directories and Go-convention "_"-prefixed directories are skipped, as is
// anything matching a vendored/testdata-style ignore. A directory with no `.goal` files
// yields no package rather than an error.
func Discover(root string) ([]*Package, error) {
	byDir := map[string][]File{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), Ext) {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		dir := filepath.Dir(path)
		byDir[dir] = append(byDir[dir], File{Path: path, Name: d.Name(), Src: string(src)})
		return nil
	})
	if err != nil {
		return nil, err
	}

	pkgs := make([]*Package, 0, len(byDir))
	for dir, files := range byDir {
		sort.Slice(files, func(a, b int) bool { return files[a].Path < files[b].Path })
		name, err := packageName(dir, files)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, &Package{Dir: dir, Name: name, Files: files})
	}
	sort.Slice(pkgs, func(a, b int) bool { return pkgs[a].Dir < pkgs[b].Dir })
	return pkgs, nil
}

// packageName reads the shared `package` clause of a directory's files, erroring if a
// file omits the clause or if two files disagree (the one-package-per-directory rule).
func packageName(dir string, files []File) (string, error) {
	name := ""
	for _, f := range files {
		got := PackageClause(f.Src)
		if got == "" {
			return "", fmt.Errorf("%s: missing package clause", f.Path)
		}
		if name == "" {
			name = got
			continue
		}
		if got != name {
			return "", fmt.Errorf("%s: found packages %q and %q in the same directory %s",
				f.Path, name, got, dir)
		}
	}
	return name, nil
}

// PackageClause returns the name in a file's leading `package <name>` clause, or "" if
// the file has no package clause. It lexes (rather than regexping) so a `package`
// keyword inside a string or comment is never mistaken for the clause.
func PackageClause(src string) string {
	toks := scan.Lex(src)
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text == "package" && textedit.IsIdent(toks[i+1].Text) {
			return toks[i+1].Text
		}
	}
	return ""
}

// skipDir reports whether a directory should be pruned from discovery: hidden dirs
// (".git", ".github"), Go-convention "_"-prefixed dirs (e.g. features/_cut), and
// "testdata" (Go's reserved non-buildable directory).
func skipDir(name string) bool {
	if name == "testdata" {
		return true
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return true
	}
	return false
}

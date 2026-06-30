// Package selfhost is the self-host verification harness: it transpiles the goal
// compiler's own packages through the goal front-end and proves the generated Go
// compiles. The static checker is silent on a class of transpile defects (the
// US-001 iota miscompile was found only because the generated Go failed to build),
// so `go build` over the generated output is the real gate. Each later port story
// (US-005+) reuses this harness to validate its package.
package selfhost

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"goal/internal/backend"
	"goal/internal/parser"
	"goal/internal/pipeline"
	"goal/internal/project"
)

// InScope lists the compiler packages the smoke gate covers, by directory name
// under internal/. Their non-test internal dependency closure is contained within
// this set, so a temp module holding the transpiled output of all of them builds
// with no extra wiring (stdlib imports pass through).
var InScope = []string{
	"token", "lexer", "ast", "parser", "sema", "project", "pipeline", "backend",
}

// ReadPackage reads the non-test *.go files in dir as goal source (the compiler is
// written in goal, a Go superset) and returns a project.Package ready for the
// front-end. Dir is set to dir so backend.TranspilePackage's import resolver finds
// the enclosing go.mod; Name is taken from the package clause. Each file is named
// "<base>.goal" so the generated Go file names come out clean ("<base>.go").
func ReadPackage(dir string) (*project.Package, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	var files []project.File
	name := ""
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		src, err := os.ReadFile(m)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", m, err)
		}
		if name == "" {
			if f, perr := parser.ParseFile(string(src)); perr == nil && f.Name != nil {
				name = f.Name.Name
			}
		}
		base := strings.TrimSuffix(filepath.Base(m), ".go") + project.Ext
		files = append(files, project.File{Path: m, Name: base, Src: string(src)})
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .go source files in %s", dir)
	}
	if name == "" {
		return nil, fmt.Errorf("could not determine package name for %s", dir)
	}
	return &project.Package{Dir: dir, Name: name, Files: files}, nil
}

// BuildTranspiled transpiles each package in layout through the goal front-end,
// writes the generated Go into a throwaway temp module under each entry's
// module-relative key directory (e.g. "internal/token"), and runs `go build ./...`
// over the lot. It returns a descriptive, package-identified error on any transpile
// failure, invalid generated Go, or build failure; nil when everything compiles.
//
// The temp module is declared `module goal` (matching the real go.mod) so that the
// transpiled packages' in-module imports (goal/internal/<pkg>) resolve against each
// other rather than the real source tree.
func BuildTranspiled(layout map[string]*project.Package) error {
	dir, err := os.MkdirTemp("", "selfhost-gate-*")
	if err != nil {
		return fmt.Errorf("temp module: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module goal\n\ngo 1.26\n"), 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}

	for relDir, pkg := range layout {
		out, err := backend.TranspilePackage(pkg)
		if err != nil {
			return fmt.Errorf("%s: transpile: %w", relDir, err)
		}
		if err := writePackage(dir, relDir, pkg.Name, out); err != nil {
			return err
		}
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("transpiled packages failed to build:\n%s", b)
	}
	return nil
}

// BuildAndTest transpiles pkg into a throwaway `module goal` temp module at
// relDir (e.g. "internal/token"), copies each path in testFiles into that same
// package directory, and runs `go test ./<relDir>`. Because the temp module is
// declared `module goal` (matching the real go.mod), the copied tests' import
// path and the package's in-module imports resolve exactly as they do in the
// real tree.
//
// deps holds any in-module dependency packages the package under test imports
// (keyed by their module-relative dir, e.g. "internal/token"); each is
// transpiled into the same temp module first so those imports resolve. Pass nil
// when the package has no in-module dependencies (e.g. the leaf token package).
//
// It is the behavioral half of the self-host port gate (BuildTranspiled is the
// compile half): the existing white-box (same-package) tests compile and run
// against the *transpiled* source, proving the ported package behaves
// identically to the trusted one. It returns a descriptive, package-identified
// error on transpile failure, invalid generated Go, a missing test file, or a
// test failure; nil when the tests pass. Reused by every later port story
// (US-005+).
func BuildAndTest(relDir string, pkg *project.Package, testFiles []string, deps map[string]*project.Package) error {
	dir, err := os.MkdirTemp("", "selfhost-port-*")
	if err != nil {
		return fmt.Errorf("temp module: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module goal\n\ngo 1.26\n"), 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}

	// Transpile in-module dependencies into the temp module first so the package
	// under test (and its tests) can import them.
	for depRel, dep := range deps {
		depOut, err := backend.TranspilePackage(dep)
		if err != nil {
			return fmt.Errorf("%s: transpile dependency %s: %w", relDir, depRel, err)
		}
		if err := writePackage(dir, depRel, dep.Name, depOut); err != nil {
			return err
		}
	}

	out, err := backend.TranspilePackage(pkg)
	if err != nil {
		return fmt.Errorf("%s: transpile: %w", relDir, err)
	}
	if err := writePackage(dir, relDir, pkg.Name, out); err != nil {
		return err
	}

	dest := filepath.Join(dir, filepath.FromSlash(relDir))
	for _, tf := range testFiles {
		src, err := os.ReadFile(tf)
		if err != nil {
			return fmt.Errorf("%s: read test file %s: %w", relDir, tf, err)
		}
		// The canonical white-box test files live beside the goal compiler
		// sources at internal/<pkg> and import their siblings as goal/internal/<pkg>;
		// the transpiled package under test is written at the same import path, so no
		// import rewrite is needed — the test's siblings and the package's siblings
		// resolve to one package instance.
		if err := os.WriteFile(filepath.Join(dest, filepath.Base(tf)), src, 0o644); err != nil {
			return fmt.Errorf("%s: write test file %s: %w", relDir, tf, err)
		}
	}

	cmd := exec.Command("go", "test", "./"+relDir)
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: transpiled package failed its tests:\n%s", relDir, b)
	}
	return nil
}

// writePackage writes a transpiled package's generated Go files into the temp
// module at relDir, checking each is valid Go first so a malformed emission is
// reported per file rather than as an opaque build error.
func writePackage(moduleDir, relDir, pkgName string, out pipeline.PackageOutput) error {
	if len(out.Files) == 0 {
		return fmt.Errorf("%s: package %q produced no output files", relDir, pkgName)
	}
	dest := filepath.Join(moduleDir, filepath.FromSlash(relDir))
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("%s: mkdir: %w", relDir, err)
	}
	for _, gf := range out.Files {
		if _, err := format.Source([]byte(gf.Go)); err != nil {
			return fmt.Errorf("%s: generated %s is not valid Go: %w", relDir, gf.Name, err)
		}
		if err := os.WriteFile(filepath.Join(dest, gf.Name), []byte(gf.Go), 0o644); err != nil {
			return fmt.Errorf("%s: write %s: %w", relDir, gf.Name, err)
		}
	}
	return nil
}

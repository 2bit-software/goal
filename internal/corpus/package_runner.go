package corpus

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"goal/internal/pipeline"
	"goal/internal/project"
)

// PackageTranspiler lowers a multi-file goal package to its Go output. It is the
// package-mode counterpart of [Transpiler]: the seam that lets the corpus judge
// any front-end's package driver by the same yardstick.
// backend.TranspilePackage satisfies it via [PackageTranspilerFunc].
type PackageTranspiler interface {
	TranspilePackage(pkg *project.Package) (pipeline.PackageOutput, error)
}

// PackageTranspilerFunc adapts a plain package-transpile function to the
// [PackageTranspiler] interface, so a free function such as
// backend.TranspilePackage can be passed where a PackageTranspiler is expected.
type PackageTranspilerFunc func(pkg *project.Package) (pipeline.PackageOutput, error)

// TranspilePackage calls the wrapped function.
func (f PackageTranspilerFunc) TranspilePackage(pkg *project.Package) (pipeline.PackageOutput, error) {
	return f(pkg)
}

// RunPackage executes one [ModePackage] Case against pt. It reads the package's
// .goal files (relative to root), transpiles them as one package through the
// seam, asserts every generated file is valid Go, and then proves the package
// coheres by building it: the generated files plus every declared foreign import
// are written into an isolated temp module and compiled with `go build ./...`.
//
// Import resolution at transpile time is handled by the package driver itself
// (it walks up from the package directory to the module's go.mod), so the
// fixture directory — passed as the package Dir — must sit inside the module.
// The case's declared import map is used to wire each foreign Go package into the
// temp build module under its import-path tail, so the generated code links.
//
// It returns a descriptive, case-identified error on any read failure, a
// transpile failure, invalid generated Go, or a build failure; nil on pass.
func RunPackage(root string, c Case, pt PackageTranspiler) error {
	if c.Mode != ModePackage {
		return fmt.Errorf("corpus: RunPackage: case %q is mode %q, not %q", c.ID, c.Mode, ModePackage)
	}
	if c.Package == nil {
		return fmt.Errorf("corpus: RunPackage: case %q has no package spec", c.ID)
	}
	spec := c.Package

	files := make([]project.File, 0, len(spec.Files))
	for _, rel := range spec.Files {
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("corpus: case %q: reading %s: %w", c.ID, rel, err)
		}
		name := filepath.Base(filepath.FromSlash(rel))
		files = append(files, project.File{Path: rel, Name: name, Src: string(src)})
	}
	if len(files) == 0 {
		return fmt.Errorf("corpus: case %q: package has no files", c.ID)
	}

	pkg := &project.Package{
		// Dir is the fixture directory inside the module, so the package driver's
		// resolver finds the enclosing go.mod and resolves in-module imports.
		Dir:   filepath.Join(root, filepath.FromSlash(c.Input)),
		Name:  spec.Name,
		Files: files,
	}

	out, err := pt.TranspilePackage(pkg)
	if err != nil {
		return fmt.Errorf("corpus: case %q: transpile package: %w", c.ID, err)
	}

	gen := append(append([]pipeline.GoFile{}, out.Files...), out.Tests...)
	if len(gen) == 0 {
		return fmt.Errorf("corpus: case %q: package produced no output files", c.ID)
	}
	for _, f := range gen {
		if _, err := format.Source([]byte(f.Go)); err != nil {
			return fmt.Errorf("corpus: case %q: generated %s is not valid Go: %w", c.ID, f.Name, err)
		}
	}

	if err := buildPackageOutput(root, c, spec, out); err != nil {
		return err
	}
	return nil
}

// buildPackageOutput writes the generated package and every declared foreign
// import into a throwaway module and runs `go build ./...`, the real proof that
// the cross-file lowering, the single prelude, and the foreign-import wiring all
// cohere.
func buildPackageOutput(root string, c Case, spec *PackageSpec, out pipeline.PackageOutput) error {
	modPath, err := moduleName(root)
	if err != nil {
		return fmt.Errorf("corpus: case %q: %w", c.ID, err)
	}

	dir, err := os.MkdirTemp("", "corpus-pkg-*")
	if err != nil {
		return fmt.Errorf("corpus: case %q: temp dir: %w", c.ID, err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+modPath+"\n\ngo 1.26\n"), 0o644); err != nil {
		return fmt.Errorf("corpus: case %q: write go.mod: %w", c.ID, err)
	}

	// The package under test lives in its own subdirectory; its files already
	// declare the package clause.
	underTest := filepath.Join(dir, "pkgundertest")
	if err := os.MkdirAll(underTest, 0o755); err != nil {
		return fmt.Errorf("corpus: case %q: mkdir: %w", c.ID, err)
	}
	for _, f := range out.Files {
		if err := os.WriteFile(filepath.Join(underTest, f.Name), []byte(f.Go), 0o644); err != nil {
			return fmt.Errorf("corpus: case %q: write %s: %w", c.ID, f.Name, err)
		}
	}

	// Wire each declared foreign import in at its import-path tail so the
	// generated code's imports resolve within the temp module.
	for importPath, srcDirRel := range spec.Imports {
		tail, ok := strings.CutPrefix(importPath, modPath+"/")
		if !ok {
			return fmt.Errorf("corpus: case %q: import %q is outside module %q", c.ID, importPath, modPath)
		}
		destDir := filepath.Join(dir, filepath.FromSlash(tail))
		if err := copyGoFiles(filepath.Join(root, filepath.FromSlash(srcDirRel)), destDir); err != nil {
			return fmt.Errorf("corpus: case %q: wiring import %q: %w", c.ID, importPath, err)
		}
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("corpus: case %q: package failed to build:\n%s", c.ID, b)
	}
	return nil
}

// copyGoFiles copies every *.go file directly under src into dst (created as
// needed). Subdirectories are not recursed; foreign fixtures used here are flat.
func copyGoFiles(src, dst string) error {
	goFiles, err := filepath.Glob(filepath.Join(src, "*.go"))
	if err != nil {
		return err
	}
	if len(goFiles) == 0 {
		return fmt.Errorf("no .go files in %s", src)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, f := range goFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, filepath.Base(f)), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// moduleName reads the module path declared by root/go.mod.
func moduleName(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("no module directive in go.mod")
}

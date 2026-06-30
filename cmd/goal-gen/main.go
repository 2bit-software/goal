// Command goal-gen is a thin Go entrypoint over the goal-SOURCED compiler: it
// imports goal/internal/compiler/{backend,pipeline,project} (the committed Go that
// `task generate` transpiles from the .goal compiler source) and exposes them as a
// runnable binary. Building it with `go build ./cmd/goal-gen` therefore proves the
// goal-written compiler compiles end to end, and running it drives the corpus
// behavioral tier through the goal-sourced front-end rather than the legacy Go one
// (US-005). It is transition scaffolding for the self-host flip, removed/folded in
// US-022.
//
// Two subcommands:
//
//	goal-gen transpile <file.goal | ->   # single file, lowered Go to stdout
//	goal-gen build --emit=<dir> <path>   # whole-package emit, mirroring `goal build`
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"goal/internal/compiler/backend"
	"goal/internal/compiler/pipeline"
	"goal/internal/compiler/project"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "goal-gen:", err)
		os.Exit(1)
	}
}

// run dispatches the subcommand. transpile lowers a single file (or stdin via "-")
// and writes the Go to out; build emits a whole package tree under --emit.
func run(args []string, stdin io.Reader, out io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: goal-gen <transpile|build> ...")
	}
	switch args[0] {
	case "transpile":
		return runTranspile(args[1:], stdin, out)
	case "build":
		return runBuild(args[1:])
	default:
		return fmt.Errorf("unknown subcommand %q (want transpile|build)", args[0])
	}
}

// runTranspile reads one .goal source (a file path, or stdin when the arg is "-"),
// lowers it through the goal-sourced backend, and writes the generated Go to out.
func runTranspile(args []string, stdin io.Reader, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: goal-gen transpile <file.goal | ->")
	}
	src, err := readInput(args[0], stdin)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	result, err := backend.Transpile(string(src))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(out, result.Go)
	return err
}

// readInput returns the bytes of name, or all of stdin when name is "-".
func readInput(name string, stdin io.Reader) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(name)
}

// runBuild discovers every goal package under <path> and emits its lowered Go under
// --emit=<dir>, preserving the module-relative directory layout. It mirrors the
// goal-sourced internal/compiler/main.go entrypoint so the binary exercises
// project.Discover and backend.TranspilePackage as well as the single-file path.
func runBuild(args []string) error {
	emitDir := ""
	root := ""
	gotPath := false
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--emit="):
			emitDir = strings.TrimPrefix(a, "--emit=")
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("unknown flag %q", a)
		default:
			if gotPath {
				return fmt.Errorf("expected a single path, got extra %q", a)
			}
			root, gotPath = a, true
		}
	}
	if emitDir == "" {
		return fmt.Errorf("goal-gen build requires --emit=<dir>")
	}
	if root == "" {
		root = "."
	}
	pkgs, err := project.Discover(root)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no .goal packages found under %s", root)
	}
	for _, pkg := range pkgs {
		out, err := backend.TranspilePackage(pkg)
		if err != nil {
			return err
		}
		if err := emitPackage(pkg, out, emitDir); err != nil {
			return err
		}
	}
	return nil
}

// emitPackage writes every generated Go file (sources and doctest sidecars) for one
// package under emitDir, preserving the package's module-relative directory.
func emitPackage(pkg *project.Package, out pipeline.PackageOutput, emitDir string) error {
	dir := filepath.Join(emitDir, pkg.Dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files := []pipeline.GoFile{}
	files = append(files, out.Files...)
	files = append(files, out.Tests...)
	for _, gf := range files {
		p := filepath.Join(dir, gf.Name)
		if err := os.WriteFile(p, []byte(gf.Go), 0o644); err != nil {
			return err
		}
	}
	return nil
}

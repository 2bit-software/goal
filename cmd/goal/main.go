// Command goal is the umbrella CLI for the goal language: it discovers the .goal
// packages under a path, transpiles them with the unified front-end, and drives the Go
// toolchain over the result.
//
//	goal build [--emit[=dir]] [path]   transpile and `go build` (default ./.)
//	goal run   [--emit[=dir]] [path]   transpile and `go run` the main package
//	goal check [path]                  run the static checker over the package(s)
//
// By default build/run are ephemeral: the generated Go is mapped into the module with
// `go build -overlay`, so nothing is written to the source tree and module/stdlib
// imports still resolve. --emit instead writes the generated .go beside each .goal (or
// into dir) for tooling and inspection. goalc remains the single-file primitive.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"goal/internal/check"
	"goal/internal/pipeline"
	"goal/internal/project"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "goal:", err)
		os.Exit(1)
	}
}

func run(args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: goal <build|run|check> [--emit[=dir]] [path]")
	}
	cmd, rest := args[0], args[1:]
	emit, emitDir, root, err := parseFlags(rest)
	if err != nil {
		return err
	}
	switch cmd {
	case "build":
		return cmdBuild(root, emit, emitDir, out, errOut)
	case "run":
		return cmdRun(root, emit, emitDir, out, errOut)
	case "check":
		return cmdCheck(root, out, errOut)
	default:
		return fmt.Errorf("unknown command %q (want build, run, or check)", cmd)
	}
}

// parseFlags pulls --emit[=dir] and a single optional path argument out of args. The
// path defaults to "." and a trailing "/..." (or bare "...") is stripped, since
// discovery is already recursive.
func parseFlags(args []string) (emit bool, emitDir, root string, err error) {
	root = "."
	gotPath := false
	for _, a := range args {
		switch {
		case a == "--emit":
			emit = true
		case strings.HasPrefix(a, "--emit="):
			emit, emitDir = true, strings.TrimPrefix(a, "--emit=")
		case strings.HasPrefix(a, "-"):
			return false, "", "", fmt.Errorf("unknown flag %q", a)
		default:
			if gotPath {
				return false, "", "", fmt.Errorf("expected a single path, got extra %q", a)
			}
			root, gotPath = a, true
		}
	}
	root = strings.TrimSuffix(strings.TrimSuffix(root, "..."), "/")
	if root == "" {
		root = "."
	}
	return emit, emitDir, root, nil
}

// transpiled pairs a package's directory with its in-memory Go output.
type transpiled struct {
	pkg *project.Package
	out pipeline.PackageOutput
}

// transpileAll discovers and transpiles every package under root.
func transpileAll(root string) ([]transpiled, error) {
	pkgs, err := project.Discover(root)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no .goal packages found under %s", root)
	}
	var ts []transpiled
	for _, pkg := range pkgs {
		out, err := pipeline.TranspilePackage(pkg)
		if err != nil {
			return nil, err
		}
		ts = append(ts, transpiled{pkg: pkg, out: out})
	}
	return ts, nil
}

func cmdBuild(root string, emit bool, emitDir string, out, errOut io.Writer) error {
	ts, err := transpileAll(root)
	if err != nil {
		return err
	}
	if emit {
		return emitFiles(ts, emitDir, out)
	}
	return goToolchain(root, ts, out, errOut, "build", "./...")
}

func cmdRun(root string, emit bool, emitDir string, out, errOut io.Writer) error {
	ts, err := transpileAll(root)
	if err != nil {
		return err
	}
	if emit {
		if err := emitFiles(ts, emitDir, out); err != nil {
			return err
		}
	}
	mainDir, err := soleMainDir(ts)
	if err != nil {
		return err
	}
	target := "."
	if rel := filepath.ToSlash(mustRel(root, mainDir)); rel != "." {
		target = "./" + rel
	}
	if emit {
		return runGo(root, nil, out, errOut, "run", target)
	}
	return goToolchain(root, ts, out, errOut, "run", target)
}

func cmdCheck(root string, out, errOut io.Writer) error {
	pkgs, err := project.Discover(root)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no .goal packages found under %s", root)
	}
	total := 0
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			diags, err := check.Analyze(f.Src)
			if err != nil {
				return fmt.Errorf("check %s: %w", f.Path, err)
			}
			for _, d := range diags {
				fmt.Fprintln(errOut, d.Render(f.Src, f.Path))
			}
			total += countErrors(diags)
		}
	}
	if total > 0 {
		return fmt.Errorf("%d checker error(s)", total)
	}
	fmt.Fprintln(out, "ok")
	return nil
}

// goToolchain runs `go <verb> <target>` over the package with the generated Go supplied
// as an overlay, so nothing is written to the source tree. Output (including any error,
// already mapped to .goal positions by the //line directives) is relayed verbatim.
func goToolchain(root string, ts []transpiled, out, errOut io.Writer, verb, target string) error {
	overlayPath, cleanup, err := writeOverlay(ts)
	if err != nil {
		return err
	}
	defer cleanup()
	return runGo(root, []string{"-overlay", overlayPath}, out, errOut, verb, target)
}

// runGo invokes the go tool with the given verb, flags, and target from dir.
func runGo(dir string, flags []string, out, errOut io.Writer, verb, target string) error {
	args := append([]string{verb}, flags...)
	args = append(args, target)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %s failed: %w", verb, err)
	}
	return nil
}

// writeOverlay materializes each package's generated Go into a temp dir and builds a
// `go build -overlay` map from the intended in-source path to the temp file. The .go
// files never touch the source tree; the .goal sources are ignored by the Go tool.
func writeOverlay(ts []transpiled) (path string, cleanup func(), err error) {
	tmp, err := os.MkdirTemp("", "goal-build-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(tmp) }
	replace := map[string]string{}
	n := 0
	for _, t := range ts {
		absDir, err := filepath.Abs(t.pkg.Dir)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		for _, gf := range t.out.Files {
			content := filepath.Join(tmp, fmt.Sprintf("%d_%s", n, gf.Name))
			n++
			if err := os.WriteFile(content, []byte(gf.Go), 0o644); err != nil {
				cleanup()
				return "", nil, err
			}
			replace[filepath.Join(absDir, gf.Name)] = content
		}
	}
	path = filepath.Join(tmp, "overlay.json")
	data, err := json.Marshal(struct{ Replace map[string]string }{replace})
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

// emitFiles writes the generated Go to disk: beside each .goal by default, or mirrored
// under emitDir when given. Test sidecars are written too so doctests can run.
func emitFiles(ts []transpiled, emitDir string, out io.Writer) error {
	for _, t := range ts {
		dir := t.pkg.Dir
		if emitDir != "" {
			dir = filepath.Join(emitDir, t.pkg.Dir)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		for _, gf := range append(append([]pipeline.GoFile{}, t.out.Files...), t.out.Tests...) {
			p := filepath.Join(dir, gf.Name)
			if err := os.WriteFile(p, []byte(gf.Go), 0o644); err != nil {
				return err
			}
			fmt.Fprintln(out, "emitted", p)
		}
	}
	return nil
}

// soleMainDir returns the directory of the one `package main` among the transpiled
// packages, erroring if there is not exactly one.
func soleMainDir(ts []transpiled) (string, error) {
	var dirs []string
	for _, t := range ts {
		if t.pkg.Name == "main" {
			dirs = append(dirs, t.pkg.Dir)
		}
	}
	switch len(dirs) {
	case 1:
		return dirs[0], nil
	case 0:
		return "", fmt.Errorf("no `package main` to run")
	default:
		return "", fmt.Errorf("multiple main packages: %s", strings.Join(dirs, ", "))
	}
}

func mustRel(base, target string) string {
	if rel, err := filepath.Rel(base, target); err == nil {
		return rel
	}
	return target
}

func countErrors(diags []check.Diagnostic) int {
	n := 0
	for _, d := range diags {
		if d.Severity == check.Error {
			n++
		}
	}
	return n
}

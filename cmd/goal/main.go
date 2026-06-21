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
	"sort"
	"strings"

	"goal/internal/check"
	"goal/internal/pipeline"
	"goal/internal/project"
	"goal/internal/typecheck"
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
		diags, err := checkPackage(pkg, errOut)
		if err != nil {
			return fmt.Errorf("check %s: %w", pkg.Dir, err)
		}
		sortDiags(diags)
		for _, d := range diags {
			fmt.Fprintln(errOut, d.render())
			if d.severity == check.Error {
				total++
			}
		}
	}
	if total > 0 {
		return fmt.Errorf("%d checker error(s)", total)
	}
	fmt.Fprintln(out, "ok")
	return nil
}

// checkPackage runs both checker stages over one package and returns their merged,
// deduplicated findings. The lexical stage (internal/check) runs on the original source,
// before lowering; the typed depth stage (internal/typecheck) runs on the lowered Go and
// answers what the lexical stage had to defer. When both flag the same construct (same
// file, line, and feature), the type-backed finding wins — it is grounded in real type
// information, whereas the lexical one may be a conservative deferral or, for an elided
// composite literal, an outright misfire. A depth-stage load failure (the program does
// not transpile) is reported as a note and the lexical findings still stand; goal build is
// the gate that hard-fails on non-transpiling source.
func checkPackage(pkg *project.Package, errOut io.Writer) ([]checkDiag, error) {
	srcs := make([]string, len(pkg.Files))
	for i, f := range pkg.Files {
		srcs[i] = f.Src
	}
	perFile, err := check.AnalyzePackage(srcs)
	if err != nil {
		return nil, err
	}

	depth, derr := runDepthChecks(pkg)
	if derr != nil {
		fmt.Fprintf(errOut, "goal check: depth stage unavailable for %s: %v\n", pkg.Dir, derr)
	}

	// A type-backed finding suppresses a lexical one for the same construct. Key on the
	// file basename (the two stages spell the path differently — full path vs. the //line
	// basename), line, and feature; within a package, basenames are unique.
	suppress := map[string]bool{}
	for _, d := range depth {
		suppress[dedupKey(d.Pos.Filename, d.Pos.Line, d.Feature)] = true
	}

	var diags []checkDiag
	for i, fileDiags := range perFile {
		path := pkg.Files[i].Path
		for _, d := range fileDiags {
			p := check.OffsetToPosition(pkg.Files[i].Src, d.Pos)
			if suppress[dedupKey(path, p.Line, d.Feature)] {
				continue // type-backed finding owns this construct
			}
			diags = append(diags, checkDiag{path, p.Line, p.Col, d.Severity, d.Code, d.Message})
		}
	}
	for _, d := range depth {
		diags = append(diags, checkDiag{
			depthFilePath(pkg, d.Pos.Filename), d.Pos.Line, d.Pos.Column,
			d.Severity, d.Code, d.Message,
		})
	}
	return diags, nil
}

// runDepthChecks loads the package's lowered Go into go/types and runs every depth check.
// It returns an error only when the package fails to transpile or parse (a goal-compiler
// problem); user type errors are tolerated inside Load.
func runDepthChecks(pkg *project.Package) ([]typecheck.Diagnostic, error) {
	p, err := typecheck.Load(pkg)
	if err != nil {
		return nil, err
	}
	var diags []typecheck.Diagnostic
	diags = append(diags, typecheck.CheckImplements(p)...)
	diags = append(diags, typecheck.CheckMustUse(p)...)
	diags = append(diags, typecheck.CheckNoZeroValue(p)...)
	return diags, nil
}

// checkDiag is a stage-agnostic rendered finding, so the two stages' diagnostics order
// and print uniformly.
type checkDiag struct {
	file          string
	line, col     int
	severity      check.Severity
	code, message string
}

// render formats the finding as `file:line:col: severity: [code] message`, matching both
// stages' native rendering.
func (d checkDiag) render() string {
	return fmt.Sprintf("%s:%d:%d: %s: [%s] %s", d.file, d.line, d.col, d.severity, d.code, d.message)
}

// sortDiags orders findings by file, then line, then column, for stable output.
func sortDiags(diags []checkDiag) {
	sort.SliceStable(diags, func(i, j int) bool {
		if diags[i].file != diags[j].file {
			return diags[i].file < diags[j].file
		}
		if diags[i].line != diags[j].line {
			return diags[i].line < diags[j].line
		}
		return diags[i].col < diags[j].col
	})
}

// dedupKey identifies a construct across the two stages by file basename, line, and the
// feature that flagged it. The basename normalizes the stages' differing path spellings.
func dedupKey(file string, line int, feature string) string {
	return fmt.Sprintf("%s:%d:%s", filepath.Base(file), line, feature)
}

// depthFilePath maps a depth diagnostic's filename (which may be a //line basename) back
// to the package's full source path, so depth findings render with the same path as
// lexical ones. It falls back to the reported name when no basename matches.
func depthFilePath(pkg *project.Package, name string) string {
	base := filepath.Base(name)
	for _, f := range pkg.Files {
		if filepath.Base(f.Path) == base {
			return f.Path
		}
	}
	return name
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

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

	"goal/internal/backend"
	"goal/internal/check"
	"goal/internal/fix"
	"goal/internal/goalfmt"
	"goal/internal/guide"
	"goal/internal/interp"
	"goal/internal/lsp"
	"goal/internal/parser"
	"goal/internal/pipeline"
	"goal/internal/project"
	"goal/internal/sema"
	"goal/internal/typecheck"
)

// guideCommands describes the binary's subcommands. It is the single source the
// dispatch in run() validates against and the one the AI guide's toolchain section
// lists, so the documented commands cannot drift from the real ones.
var guideCommands = []guide.Command{
	{
		Name:    "build",
		Summary: "transpile and `go build` the package(s)",
		Usage:   "goal build [--emit[=dir]] [path]",
		Flags: []guide.Flag{
			{Name: "--emit[=dir]", Summary: "also write generated .go beside each .goal (or under dir)"},
		},
	},
	{
		Name:    "run",
		Summary: "transpile and `go run` the sole main package",
		Usage:   "goal run [--engine=ast|interp] [--emit[=dir]] [path]",
		Flags: []guide.Flag{
			{Name: "--engine=ast|interp", Summary: "ast (default) transpiles and `go run`s; interp runs a single .goal file under the goscript tree-walking interpreter"},
			{Name: "--emit[=dir]", Summary: "also write generated .go beside each .goal (or under dir)"},
		},
	},
	{
		Name:    "check",
		Summary: "run the static checker over the package(s)",
		Usage:   "goal check [path]",
	},
	{
		Name:    "fix",
		Summary: "rewrite plain-Go patterns into idiomatic goal (Result + `?`)",
		Usage:   "goal fix [-inplace] [path]",
		Flags:   []guide.Flag{{Name: "-inplace", Summary: "write changes back to each file instead of printing to stdout"}},
	},
	{
		Name:    "fmt",
		Summary: "format .goal source into the canonical, comment-preserving layout",
		Usage:   "goal fmt [-w] [path]",
		Flags:   []guide.Flag{{Name: "-w", Summary: "write the formatted result back to each file instead of printing to stdout"}},
	},
	{
		Name:    "ai",
		Summary: "print the AI bootstrap guide (how to write goal) to stdout",
		Usage:   "goal ai [section]",
	},
	{
		Name:    "lsp",
		Summary: "run the language server (editor diagnostics) over stdio",
		Usage:   "goal lsp",
	},
}

// topUsage is the one-line usage listing every subcommand.
func topUsage() string {
	names := make([]string, len(guideCommands))
	for i, c := range guideCommands {
		names[i] = c.Name
	}
	return "usage: goal <" + strings.Join(names, "|") + "> [--emit[=dir]] [path]"
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "goal:", err)
		os.Exit(1)
	}
}

func run(args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", topUsage())
	}
	cmd, rest := args[0], args[1:]
	if cmd == "--ai" { // alias: `goal --ai [section]` == `goal ai [section]`
		cmd = "ai"
	}
	switch cmd {
	case "ai":
		return cmdAI(rest, out)
	case "lsp":
		return lsp.NewServer(out).Run(os.Stdin)
	case "fix":
		path, inplace, err := parseFixFlags(rest)
		if err != nil {
			return err
		}
		return cmdFix(path, inplace, out, errOut)
	case "fmt":
		path, write, err := parseFmtFlags(rest)
		if err != nil {
			return err
		}
		return cmdFmt(path, write, out, errOut)
	case "run":
		engine, emit, emitDir, root, err := parseRunFlags(rest)
		if err != nil {
			return err
		}
		if engine == engineInterp {
			return cmdRunInterp(root, out, errOut)
		}
		return cmdRun(root, emit, emitDir, out, errOut)
	case "build", "check":
		emit, emitDir, root, err := parseFlags(rest)
		if err != nil {
			return err
		}
		if cmd == "build" {
			return cmdBuild(root, emit, emitDir, out, errOut)
		}
		return cmdCheck(root, out, errOut)
	default:
		return fmt.Errorf("unknown command %q (%s)", cmd, topUsage())
	}
}

// cmdAI prints the AI bootstrap guide to out. With no argument it prints the whole
// guide; with one argument it prints only that named section.
func cmdAI(args []string, out io.Writer) error {
	section := ""
	switch len(args) {
	case 0:
	case 1:
		section = args[0]
	default:
		return fmt.Errorf("usage: goal ai [section] (sections: %s)", strings.Join(guide.SectionKeys(), ", "))
	}
	return guide.Render(out, section, guideCommands)
}

// parseFlags pulls --emit[=dir] and a single optional path argument out of args.
// The path defaults to "." and a trailing "/..." (or bare "...") is stripped,
// since discovery is already recursive.
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

// Engine names select which back-end `goal run` uses. ast (the default)
// transpiles to Go and drives the Go toolchain; interp runs a single .goal file
// directly under the goscript tree-walking interpreter (internal/interp).
const (
	engineAST    = "ast"
	engineInterp = "interp"
)

// parseRunFlags parses the `run` subcommand's flags: --engine=ast|interp
// (default ast), --emit[=dir], and a single optional path. It mirrors parseFlags
// for emit/path handling and adds the engine selector; an unknown engine value
// is a descriptive error so a typo never silently falls back to a different
// back-end.
func parseRunFlags(args []string) (engine string, emit bool, emitDir, root string, err error) {
	engine, root = engineAST, "."
	gotPath := false
	for _, a := range args {
		switch {
		case a == "--engine" || a == "-engine":
			return "", false, "", "", fmt.Errorf("flag %q requires a value (--engine=ast|interp)", a)
		case strings.HasPrefix(a, "--engine="):
			engine = strings.TrimPrefix(a, "--engine=")
		case strings.HasPrefix(a, "-engine="):
			engine = strings.TrimPrefix(a, "-engine=")
		case a == "--emit":
			emit = true
		case strings.HasPrefix(a, "--emit="):
			emit, emitDir = true, strings.TrimPrefix(a, "--emit=")
		case strings.HasPrefix(a, "-"):
			return "", false, "", "", fmt.Errorf("unknown flag %q", a)
		default:
			if gotPath {
				return "", false, "", "", fmt.Errorf("expected a single path, got extra %q", a)
			}
			root, gotPath = a, true
		}
	}
	if engine != engineAST && engine != engineInterp {
		return "", false, "", "", fmt.Errorf("unknown engine %q (want ast or interp)", engine)
	}
	root = strings.TrimSuffix(strings.TrimSuffix(root, "..."), "/")
	if root == "" {
		root = "."
	}
	return engine, emit, emitDir, root, nil
}

// cmdRunInterp runs a single .goal file under the goscript tree-walking
// interpreter: it parses the source through the shared front-end
// (internal/parser + internal/sema) and executes func main via internal/interp,
// with the program's standard-output effect routed to out and full host
// authority (the interpreter's GrantAll default). The interpreter consumes the
// shared AST + sema front-end directly — no Go transpilation, no Go toolchain —
// so it runs in a host with no Go installed. A missing/non-.goal path, a parse
// failure, a refused static guarantee (the native sema gate), a missing func
// main, or a runtime failure is a located, descriptive error (the command exits
// non-zero), never a silent success.
func cmdRunInterp(path string, out, errOut io.Writer) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("--engine=interp runs a single .goal file, not a directory: %s", path)
	}
	if !strings.HasSuffix(path, project.Ext) {
		return fmt.Errorf("not a .goal file: %s", path)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	file, err := parser.ParseFile(string(src))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	semaInfo := sema.Resolve(file)
	ip := interp.New(file, semaInfo, interp.WithStdout(out))
	if err := ip.Run(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// parseFixFlags pulls the -inplace flag and a single optional path (a .goal file or a
// directory, default ".") out of args.
func parseFixFlags(args []string) (path string, inplace bool, err error) {
	path = "."
	gotPath := false
	for _, a := range args {
		switch {
		case a == "-inplace" || a == "--inplace":
			inplace = true
		case strings.HasPrefix(a, "-"):
			return "", false, fmt.Errorf("unknown flag %q", a)
		default:
			if gotPath {
				return "", false, fmt.Errorf("expected a single path, got extra %q", a)
			}
			path, gotPath = a, true
		}
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, "..."), "/")
	if path == "" {
		path = "."
	}
	return path, inplace, nil
}

// cmdFix rewrites plain-Go patterns into idiomatic goal across a .goal file or every .goal
// file under a directory. By default it prints each rewritten file to stdout and writes
// nothing; with -inplace it writes changed files back in place and lists them. Suggestions
// and skip/warning reports always go to stderr; only an operational failure (bad path,
// unreadable/unwritable file) makes the command fail.
func cmdFix(path string, inplace bool, out, errOut io.Writer) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	type fileRef struct{ path, src string }
	var files []fileRef
	if info.IsDir() {
		pkgs, err := project.Discover(path)
		if err != nil {
			return err
		}
		if len(pkgs) == 0 {
			return fmt.Errorf("no .goal packages found under %s", path)
		}
		for _, pkg := range pkgs {
			for _, f := range pkg.Files {
				files = append(files, fileRef{f.Path, f.Src})
			}
		}
	} else {
		if !strings.HasSuffix(path, project.Ext) {
			return fmt.Errorf("not a .goal file: %s", path)
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, fileRef{path, string(src)})
	}

	for _, fr := range files {
		newSrc, _, reports := fix.File(fr.src)
		for _, r := range reports {
			fmt.Fprintf(errOut, "%s:%d: %s: [%s] %s\n", fr.path, r.Line, r.Level, r.Rule, r.Msg)
		}
		if inplace {
			if newSrc != fr.src {
				if err := os.WriteFile(fr.path, []byte(newSrc), 0o644); err != nil {
					return err
				}
				fmt.Fprintln(out, "fixed", fr.path)
			}
			continue
		}
		if _, err := io.WriteString(out, newSrc); err != nil {
			return err
		}
	}
	return nil
}

// parseFmtFlags pulls the -w (write-in-place) flag and a single optional path (a .goal
// file or a directory, default ".") out of args.
func parseFmtFlags(args []string) (path string, write bool, err error) {
	path = "."
	gotPath := false
	for _, a := range args {
		switch {
		case a == "-w" || a == "--write":
			write = true
		case strings.HasPrefix(a, "-"):
			return "", false, fmt.Errorf("unknown flag %q", a)
		default:
			if gotPath {
				return "", false, fmt.Errorf("expected a single path, got extra %q", a)
			}
			path, gotPath = a, true
		}
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, "..."), "/")
	if path == "" {
		path = "."
	}
	return path, write, nil
}

// cmdFmt formats a .goal file or every .goal file under a directory into the canonical,
// comment-preserving layout. By default it prints each formatted file to stdout; with -w
// it writes changed files back in place and lists them. A file that does not parse is a
// failure — goalfmt never reformats malformed source.
func cmdFmt(path string, write bool, out, errOut io.Writer) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	type fileRef struct{ path, src string }
	var files []fileRef
	if info.IsDir() {
		pkgs, err := project.Discover(path)
		if err != nil {
			return err
		}
		if len(pkgs) == 0 {
			return fmt.Errorf("no .goal packages found under %s", path)
		}
		for _, pkg := range pkgs {
			for _, f := range pkg.Files {
				files = append(files, fileRef{f.Path, f.Src})
			}
		}
	} else {
		if !strings.HasSuffix(path, project.Ext) {
			return fmt.Errorf("not a .goal file: %s", path)
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, fileRef{path, string(src)})
	}

	for _, fr := range files {
		formatted, err := goalfmt.Source(fr.src)
		if err != nil {
			return fmt.Errorf("%s: %w", fr.path, err)
		}
		if write {
			if formatted != fr.src {
				if err := os.WriteFile(fr.path, []byte(formatted), 0o644); err != nil {
					return err
				}
				fmt.Fprintln(out, "formatted", fr.path)
			}
			continue
		}
		if _, err := io.WriteString(out, formatted); err != nil {
			return err
		}
	}
	return nil
}

// transpiled pairs a package's directory with its in-memory Go output.
type transpiled struct {
	pkg *project.Package
	out pipeline.PackageOutput
}

// transpileAll discovers and transpiles every package under root through the AST
// backend: backend.TranspilePackage performs the cross-file fact merge plus a
// single shared prelude per package.
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
		out, err := backend.TranspilePackage(pkg)
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
	perFile, err := check.AnalyzePackageInDir(srcs, pkg.Dir)
	if err != nil {
		return nil, err
	}

	depth, derr := runDepthChecks(pkg)
	if derr != nil {
		fmt.Fprintf(errOut, "goal check: depth stage unavailable for %s: %v\n", pkg.Dir, briefDepthErr(derr))
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

// briefDepthErr renders a depth-stage failure for the non-fatal `check` note: it drops the
// "--- generated ---" Go dump that the transpile error carries for `goal build`'s benefit,
// keeping only the concise reason. The full dump still surfaces on `goal build`, the gate
// that hard-fails on non-transpiling source.
func briefDepthErr(err error) string {
	brief, _, _ := strings.Cut(err.Error(), "\n--- generated ---\n")
	return brief
}

// runDepthChecks loads the package's lowered Go into go/types and runs every depth check.
// It returns an error only when the package fails to transpile or parse (a goal-compiler
// problem); user type errors are tolerated inside Load.
func runDepthChecks(pkg *project.Package) ([]typecheck.Diagnostic, error) {
	// Resolve depth diagnostics through the TypeChecker seam so the go/types crutch can be
	// swapped for a native goal checker later without changing this caller (US-028).
	var tc typecheck.TypeChecker = typecheck.GoTypesChecker{}
	return tc.Check(pkg)
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

// Command goal is the umbrella CLI for the goal language: it discovers the .goal
// packages under a path, transpiles them with the AST front-end, and drives the Go
// toolchain over the result.
//
//	goal build [--emit[=dir]] [path]        transpile and `go build` (default ./.)
//	goal run   [--engine=ast|interp] [path] transpile and `go run` the main package (or interpret a file)
//	goal check [path]                       run the static checker over the package(s)
//	goal test  [path]                       transpile and `go test` the package's doctests
//	goal fix   [-inplace] [path]            rewrite plain-Go patterns into idiomatic goal
//	goal fmt   [-w] [path]                  format .goal source into the canonical layout
//	goal ai    [section]                    print the AI bootstrap guide to stdout
//	goal category [name]                    list language features, or print one in full
//	goal lsp                                run the language server over stdio
//
// By default build/run are ephemeral: the generated Go is mapped into the module with
// `go build -overlay`, so nothing is written to the source tree and module/stdlib
// imports still resolve. --emit instead writes the generated .go beside each .goal (or
// into dir) for tooling and inspection. goalc remains the single-file primitive.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"goal/internal/backend"
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
		Usage:   "goal run [--engine=ast|interp] [--emit[=dir]] [path] [args...]",
		Flags: []guide.Flag{
			{Name: "--engine=ast|interp", Summary: "ast (default) transpiles and `go run`s; interp runs a single .goal file under the goscript tree-walking interpreter"},
			{Name: "--emit[=dir]", Summary: "also write generated .go beside each .goal (or under dir)"},
			{Name: "[args...]", Summary: "arguments after the path are passed through to the running program (default engine), as with `go run <pkg> [args...]`"},
		},
	},
	{
		Name:    "check",
		Summary: "run the static checker over the package(s)",
		Usage:   "goal check [path]",
	},
	{
		Name:    "test",
		Summary: "transpile and `go test` the package's doctests (ephemeral, via -overlay)",
		Usage:   "goal test [path]",
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
		Name:    "category",
		Summary: "list the language features, or print one feature's full reference",
		Usage:   "goal category [name]",
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

// codedError tags an error with the process exit code goal should return for it.
// It classifies a failure into one of goal's exit-code tiers (see exitCode) while
// staying transparent to errors.Is/As and preserving the underlying message, so
// callers and tests that inspect the cause are unaffected.
type codedError struct {
	code int
	err  error
}

func (e *codedError) Error() string { return e.err.Error() }
func (e *codedError) Unwrap() error { return e.err }

// usageErr tags err as a caller/invocation mistake — an unknown subcommand, an
// unknown or malformed flag, or bad arguments — which goal reports with exit 2.
func usageErr(err error) error {
	if err == nil {
		return nil
	}
	return &codedError{code: 2, err: err}
}

// internalErr tags err as a goal-internal failure — a transpiler ICE (generated
// Go that does not parse) or a build-overlay/toolchain setup failure not
// attributable to the user's code — which goal reports with exit 3.
func internalErr(err error) error {
	if err == nil {
		return nil
	}
	return &codedError{code: 3, err: err}
}

// exitCode maps a run() error to goal's process exit code so an automated
// consumer can triage failures without parsing prose: 0 success, 2 usage, 3
// internal, and 1 (the default) for user-code diagnostics — checker findings,
// syntax errors, a failed `go build` of correct-shaped output, a program's own
// non-zero `goal run` exit, and interpreter runtime failures.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code
	}
	return 1
}

// Build metadata, injected at release time by goreleaser via -ldflags -X (see
// .goreleaser.yaml). The defaults keep plain `go build`/`go run` working.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "goal:", err)
		os.Exit(exitCode(err))
	}
}

func run(args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		return usageErr(fmt.Errorf("%s", topUsage()))
	}
	cmd, rest := args[0], args[1:]
	if cmd == "--ai" { // alias: `goal --ai [section]` == `goal ai [section]`
		cmd = "ai"
	}
	switch cmd {
	case "version", "--version":
		fmt.Fprintf(out, "goal %s (commit %s, built %s)\n", version, commit, date)
		return nil
	case "ai":
		return cmdAI(rest, out)
	case "category":
		return cmdCategory(rest, out)
	case "lsp":
		return lsp.NewServer(out).Run(os.Stdin)
	case "fix":
		path, inplace, err := parseFixFlags(rest)
		if err != nil {
			return usageErr(err)
		}
		return cmdFix(path, inplace, out, errOut)
	case "fmt":
		path, write, err := parseFmtFlags(rest)
		if err != nil {
			return usageErr(err)
		}
		return cmdFmt(path, write, out, errOut)
	case "run":
		engine, emit, emitDir, root, progArgs, err := parseRunFlags(rest)
		if err != nil {
			return usageErr(err)
		}
		if engine == engineInterp {
			return cmdRunInterp(root, progArgs, out, errOut)
		}
		return cmdRun(root, emit, emitDir, progArgs, out, errOut)
	case "test":
		emit, jsonOut, _, root, err := parseFlags(rest)
		if err != nil {
			return usageErr(err)
		}
		// `goal test` is deliberately ephemeral: --emit (write files to the tree)
		// and --json (a check-only diagnostics mode) both contradict it.
		if emit {
			return usageErr(fmt.Errorf("unknown flag %q", "--emit"))
		}
		if jsonOut {
			return usageErr(fmt.Errorf("unknown flag %q", "--json"))
		}
		return cmdTest(root, out, errOut)
	case "build", "check":
		emit, jsonOut, emitDir, root, err := parseFlags(rest)
		if err != nil {
			return usageErr(err)
		}
		if cmd == "build" {
			if jsonOut {
				return usageErr(fmt.Errorf("unknown flag %q", "--json"))
			}
			return cmdBuild(root, emit, emitDir, out, errOut)
		}
		return cmdCheck(root, jsonOut, out, errOut)
	default:
		return usageErr(fmt.Errorf("unknown command %q (%s)", cmd, topUsage()))
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
		return usageErr(fmt.Errorf("usage: goal ai [section] (sections: %s)", strings.Join(guide.SectionKeys(), ", ")))
	}
	return guide.Render(out, section, guideCommands)
}

// cmdCategory prints the list of language-feature categories (no argument) or one
// category's full reference (a single name). More than one argument is a usage error; an
// unknown name is reported by guide.RenderCategory with the valid set named.
func cmdCategory(args []string, out io.Writer) error {
	switch len(args) {
	case 0:
		var b strings.Builder
		b.WriteString("Language features — run `goal category <name>` for one in full:\n\n")
		for _, c := range guide.Categories() {
			fmt.Fprintf(&b, "  %-14s %s\n", c.Name, c.Description)
		}
		_, err := io.WriteString(out, b.String())
		return err
	case 1:
		return guide.RenderCategory(out, args[0])
	default:
		return usageErr(fmt.Errorf("usage: goal category [name] (categories: %s)", strings.Join(guide.CategoryNames(), ", ")))
	}
}

// parseFlags pulls --emit[=dir], --json, and a single optional path argument out
// of args. The path defaults to "." and a trailing "/..." (or bare "...") is
// stripped, since discovery is already recursive. --json is meaningful only for
// `check` (machine-readable diagnostics); `build`'s dispatch rejects it.
func parseFlags(args []string) (emit, jsonOut bool, emitDir, root string, err error) {
	root = "."
	gotPath := false
	for _, a := range args {
		switch {
		case a == "--emit":
			emit = true
		case strings.HasPrefix(a, "--emit="):
			emit, emitDir = true, strings.TrimPrefix(a, "--emit=")
		case a == "--json":
			jsonOut = true
		case strings.HasPrefix(a, "-"):
			return false, false, "", "", fmt.Errorf("unknown flag %q", a)
		default:
			if gotPath {
				return false, false, "", "", fmt.Errorf("expected a single path, got extra %q", a)
			}
			root, gotPath = a, true
		}
	}
	root = strings.TrimSuffix(strings.TrimSuffix(root, "..."), "/")
	if root == "" {
		root = "."
	}
	return emit, jsonOut, emitDir, root, nil
}

// Engine names select which back-end `goal run` uses. ast (the default)
// transpiles to Go and drives the Go toolchain; interp runs a single .goal file
// directly under the goscript tree-walking interpreter (internal/interp).
const (
	engineAST    = "ast"
	engineInterp = "interp"
)

// parseRunFlags parses the `run` subcommand's flags: --engine=ast|interp
// (default ast), --emit[=dir], a single path, and any program arguments that
// follow it. Like `go run <pkg> [args...]`, goal's own flags must precede the
// path; the first positional is the path, and every token after it — flags
// included — is collected verbatim as a program argument and handed to the
// running program rather than interpreted by goal. An unknown engine value is a
// descriptive error so a typo never silently falls back to a different back-end.
func parseRunFlags(args []string) (engine string, emit bool, emitDir, root string, progArgs []string, err error) {
	engine, root = engineAST, "."
	gotPath := false
	for i, a := range args {
		// Once the path is set, the rest of the line belongs to the program.
		if gotPath {
			progArgs = args[i:]
			break
		}
		switch {
		case a == "--engine" || a == "-engine":
			return "", false, "", "", nil, fmt.Errorf("flag %q requires a value (--engine=ast|interp)", a)
		case strings.HasPrefix(a, "--engine="):
			engine = strings.TrimPrefix(a, "--engine=")
		case strings.HasPrefix(a, "-engine="):
			engine = strings.TrimPrefix(a, "-engine=")
		case a == "--emit":
			emit = true
		case strings.HasPrefix(a, "--emit="):
			emit, emitDir = true, strings.TrimPrefix(a, "--emit=")
		case strings.HasPrefix(a, "-"):
			return "", false, "", "", nil, fmt.Errorf("unknown flag %q", a)
		default:
			root, gotPath = a, true
		}
	}
	if engine != engineAST && engine != engineInterp {
		return "", false, "", "", nil, fmt.Errorf("unknown engine %q (want ast or interp)", engine)
	}
	root = strings.TrimSuffix(strings.TrimSuffix(root, "..."), "/")
	if root == "" {
		root = "."
	}
	return engine, emit, emitDir, root, progArgs, nil
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
func cmdRunInterp(path string, progArgs []string, out, errOut io.Writer) error {
	if len(progArgs) > 0 {
		// The interpreter has no os.Args bridge yet, so it cannot deliver program
		// arguments to the running program. Refuse loudly by name rather than
		// silently dropping them and running as if none were given.
		return fmt.Errorf("--engine=interp does not support program arguments yet (got %v); use the default engine to pass arguments", progArgs)
	}
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
		newSrc, _, reports, ferr := fix.File(fr.src)
		if ferr != nil {
			// fix's own rewrite failed to reparse: abort before writing so -inplace
			// can never overwrite the file with corrupt output. fix.File already
			// reverted newSrc to the pristine input; refuse the file loudly.
			return fmt.Errorf("%s: %w", fr.path, ferr)
		}
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

// transpileAll discovers and transpiles the package(s) at root plus the transitive
// closure of the module-local .goal packages they import, through the AST backend:
// backend.TranspilePackage performs the cross-file fact merge plus a single shared
// prelude per package. Building/emitting a sub-package therefore also transpiles the
// sibling packages it depends on, so the overlay build and the emitted tree both see
// Go source for the whole dependency graph rather than a .goal-only sibling the Go
// toolchain reports as missing.
func transpileAll(root string) ([]transpiled, error) {
	pkgs, err := discoverClosure(root)
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

// discoverClosure discovers the .goal packages under root, then expands the set with
// the transitive closure of their module-local imports. A build/emit that names a
// sub-package (e.g. ./cmd/app) thereby also transpiles the sibling .goal packages it
// depends on, so the overlay build and the emitted tree both see Go source for the
// whole dependency graph. Resolution is targeted: each module-local import is mapped
// straight to its directory and that one directory is read — no module-wide walk —
// so an import to a Go-only sibling (no .goal source) resolves to nothing, an
// import satisfied by a seed dedups away, and a whole-module `.`/`./...` build adds
// no packages and keeps its prior emit layout byte-for-byte. When root is not inside
// a Go module, the seed set is returned unchanged.
func discoverClosure(root string) ([]*project.Package, error) {
	seeds, err := project.Discover(root)
	if err != nil {
		return nil, err
	}
	modDir, modPath, ok := moduleInfo(root)
	if !ok {
		return seeds, nil
	}

	closure := map[string]*project.Package{} // import path -> package
	var order, queue []*project.Package
	enqueue := func(p *project.Package) {
		ip := pkgImportPath(modPath, modDir, p.Dir)
		if closure[ip] != nil {
			return
		}
		closure[ip] = p
		order = append(order, p)
		queue = append(queue, p)
	}
	for _, s := range seeds {
		enqueue(s)
	}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, imp := range packageImports(p) {
			if !isModuleLocal(modPath, imp) || closure[imp] != nil {
				continue
			}
			dep, err := resolveLocalImport(root, modDir, modPath, imp)
			if err != nil {
				return nil, err
			}
			if dep != nil {
				enqueue(dep)
			}
		}
	}
	sort.Slice(order, func(a, b int) bool { return order[a].Dir < order[b].Dir })
	return order, nil
}

// resolveLocalImport maps a module-local import path to the .goal package in its
// directory, or nil when that directory has no .goal source (a Go-only sibling the
// Go toolchain resolves on its own) or does not exist. The returned package's Dir is
// expressed in root's coordinate system (absolute for an absolute root, else
// relative to the working directory) so it resolves through filepath.Abs exactly as
// a plain project.Discover would.
func resolveLocalImport(root, modDir, modPath, imp string) (*project.Package, error) {
	sub := strings.TrimPrefix(strings.TrimPrefix(imp, modPath), "/")
	absDir := modDir
	if sub != "" {
		absDir = filepath.Join(modDir, filepath.FromSlash(sub))
	}
	dir := absDir
	if !filepath.IsAbs(root) {
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, absDir); err == nil {
				dir = rel
			}
		}
	}
	return readPackageDir(dir)
}

// readPackageDir reads exactly the .goal files directly in dir (non-recursively) as
// one package, mirroring what project.Discover produces for a single directory. It
// returns nil when dir has no .goal files or does not exist — the two cases a
// module-local import need not (or cannot) transpile. Reading only the named
// directory avoids walking sibling trees, so a directory of intentionally
// conflicting example packages elsewhere in the module never derails resolution.
func readPackageDir(dir string) (*project.Package, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	name := ""
	var files []project.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), project.Ext) {
			continue
		}
		p := filepath.Join(dir, e.Name())
		src, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		clause := project.PackageClause(string(src))
		if clause == "" {
			continue
		}
		if name == "" {
			name = clause
		}
		files = append(files, project.File{Path: p, Name: e.Name(), Src: string(src)})
	}
	if len(files) == 0 {
		return nil, nil
	}
	sort.Slice(files, func(a, b int) bool { return files[a].Path < files[b].Path })
	return &project.Package{Dir: dir, Name: name, Files: files}, nil
}

// moduleInfo locates the enclosing Go module of dir: the nearest ancestor holding a
// go.mod, as an absolute path, plus that file's declared module path. It reports
// ok=false when no go.mod is found, so callers fall back to path-scoped discovery.
func moduleInfo(dir string) (modDir, modPath string, ok bool) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", false
	}
	for {
		if data, err := os.ReadFile(filepath.Join(abs, "go.mod")); err == nil {
			if mp := modulePathFromGoMod(string(data)); mp != "" {
				return abs, mp, true
			}
			return "", "", false
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", "", false
		}
		abs = parent
	}
}

// modulePathFromGoMod returns the path from a go.mod's `module` directive, or "".
func modulePathFromGoMod(src string) string {
	for _, line := range strings.Split(src, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// pkgImportPath is the Go import path of a package: the module path joined with the
// package directory's path relative to the module root. It resolves both dirs to
// absolute form first, so it is independent of the coordinate system pkgDir is
// expressed in. The module-root package itself maps to the bare module path.
func pkgImportPath(modPath, absModDir, pkgDir string) string {
	abs, err := filepath.Abs(pkgDir)
	if err != nil {
		return modPath
	}
	rel, err := filepath.Rel(absModDir, abs)
	if err != nil || rel == "." {
		return modPath
	}
	return modPath + "/" + filepath.ToSlash(rel)
}

// isModuleLocal reports whether import path imp resolves inside the module modPath.
func isModuleLocal(modPath, imp string) bool {
	return imp == modPath || strings.HasPrefix(imp, modPath+"/")
}

// packageImports returns the distinct import paths declared across a package's files,
// parsed from source (a file that fails to parse contributes none — a genuine syntax
// error surfaces later in the shared syntax-diagnostic pass).
func packageImports(p *project.Package) []string {
	var out []string
	seen := map[string]bool{}
	for _, f := range p.Files {
		file, err := parser.ParseFile(f.Src)
		if err != nil || file == nil {
			continue
		}
		for _, is := range file.Imports {
			if is.Path == nil {
				continue
			}
			path, err := strconv.Unquote(is.Path.Value)
			if err != nil || seen[path] {
				continue
			}
			seen[path] = true
			out = append(out, path)
		}
	}
	return out
}

// reportWarnings prints each package's out-of-band front-end warnings (e.g. `?`
// arity-resolution fallbacks, US-022) to errOut in the shared `file:line:col:
// warning: [code] message` format, so they parse with the same regex as check and
// build diagnostics. Warnings never affect exit status or generated output.
func reportWarnings(ts []transpiled, errOut io.Writer) {
	for _, t := range ts {
		for _, w := range t.out.Warnings {
			fmt.Fprintf(errOut, "%s:%d:%d: warning: [%s] %s\n", w.File, w.Line, w.Col, w.Code, w.Message)
		}
	}
}

func cmdBuild(root string, emit bool, emitDir string, out, errOut io.Writer) error {
	// A syntax error is reported in the shared `file:line:col: error: [syntax] message`
	// format before the Go toolchain runs, so build parse errors parse with the same
	// regex as check diagnostics (rather than the backend's bare "parse:" wrapper).
	pkgs, derr := project.Discover(root)
	if derr != nil {
		return derr
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no .goal packages found under %s", root)
	}
	var syntax []checkDiag
	for _, pkg := range pkgs {
		syntax = append(syntax, packageSyntaxDiags(pkg)...)
	}
	if len(syntax) > 0 {
		sortDiags(syntax)
		for _, d := range syntax {
			fmt.Fprintln(errOut, d.render())
		}
		return fmt.Errorf("%d syntax error(s)", len(syntax))
	}

	// Every file parsed cleanly above, so a transpile failure here is a
	// goal-internal defect (a backend ICE), not the user's syntax — classify it
	// as an internal error (exit 3), distinct from user-code diagnostics.
	ts, err := transpileAll(root)
	if err != nil {
		return internalErr(err)
	}
	reportWarnings(ts, errOut)
	if emit {
		return emitFiles(ts, emitDir, out, errOut)
	}
	return goToolchain(root, ts, out, errOut, "build", "./...")
}

// cmdTest transpiles the package(s) under root and runs their doctests via the
// build overlay, so no generated .go/_test.go files touch the source tree. The
// doctest sidecars are mapped into the module by writeOverlay; `go test` picks
// them up. A doctest failure propagates as `go test`'s non-zero exit (exit 1);
// its message carries the .goal source position (package-mode doctest rendering,
// US-014). `go test` output is passed through unmodified — unlike `build`,
// goToolchain does not rewrite the test verb's stderr.
func cmdTest(root string, out, errOut io.Writer) error {
	ts, err := transpileAll(root)
	if err != nil {
		// A clean-parsing package that fails to transpile is a backend ICE, not
		// user syntax — classify it as internal (exit 3), as cmdBuild does.
		return internalErr(err)
	}
	reportWarnings(ts, errOut)
	return goToolchain(root, ts, out, errOut, "test", "./...", "-count=1")
}

func cmdRun(root string, emit bool, emitDir string, progArgs []string, out, errOut io.Writer) error {
	ts, err := transpileAll(root)
	if err != nil {
		return err
	}
	reportWarnings(ts, errOut)
	if emit {
		if err := emitFiles(ts, emitDir, out, errOut); err != nil {
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
		return runGo(root, nil, out, errOut, "run", target, progArgs...)
	}
	return goToolchain(root, ts, out, errOut, "run", target, progArgs...)
}

func cmdCheck(root string, jsonMode bool, out, errOut io.Writer) error {
	pkgs, err := project.Discover(root)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no .goal packages found under %s", root)
	}
	// A directory/package target is checked in full — every sibling file is present,
	// so a depth-stage transpile failure is a genuine compile error and hard-fails
	// (below). A single-file target is an incomplete unit (a referenced sibling may be
	// out of view), so its transpile failure stays the soft "depth stage unavailable"
	// note. os.Stat failure (e.g. a `./...` pattern already trimmed to a dir) defaults
	// to the strict, directory reading.
	strict := true
	if fi, err := os.Stat(root); err == nil && !fi.IsDir() {
		strict = false
	}
	// One source importer, shared across every package's depth stage: it type-checks each
	// dependency (stdlib and sibling goal packages) from source once and caches it, so an
	// N-package run does not re-type-check every shared sibling ~N times.
	shared := typecheck.NewImporters()

	// Accumulate every package's findings so JSON mode can emit a single array and
	// both modes render in one stable order.
	var all []checkDiag
	for _, pkg := range pkgs {
		diags, err := checkPackage(pkg, shared, strict, errOut)
		if err != nil {
			return fmt.Errorf("check %s: %w", pkg.Dir, err)
		}
		all = append(all, diags...)
	}
	sortDiags(all)

	total := 0
	for _, d := range all {
		if _, ok := d.severity.(sema.Severity_Error); ok {
			total++
		}
	}

	if jsonMode {
		// Machine-readable diagnostics go to stdout as one JSON array (empty ->
		// "[]"); nothing else may print to stdout. The "ok" line is suppressed and
		// any depth-stage notes stay on stderr (emitted inside checkPackage).
		payload := make([]jsonDiag, 0, len(all))
		for _, d := range all {
			payload = append(payload, d.toJSON())
		}
		enc, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(enc))
		if total > 0 {
			return fmt.Errorf("%d checker error(s)", total)
		}
		return nil
	}

	for _, d := range all {
		fmt.Fprintln(errOut, d.render())
	}
	if total > 0 {
		return fmt.Errorf("%d checker error(s)", total)
	}
	fmt.Fprintln(out, "ok")
	return nil
}

// checkPackage runs both checker stages over one package and returns their merged,
// deduplicated findings. The AST stage (internal/sema) runs on the parsed source,
// before lowering; the typed depth stage (internal/typecheck) runs on the lowered Go and
// answers what the AST stage had to defer. When both flag the same construct (same
// file, line, and feature), the type-backed finding wins — it is grounded in real type
// information, whereas the AST one may be a conservative deferral or, for an elided
// composite literal, an outright misfire. A depth-stage load failure (the program does
// not transpile) is reported as a note and the AST findings still stand; goal build is
// the gate that hard-fails on non-transpiling source.
func checkPackage(pkg *project.Package, shared *typecheck.Importers, strict bool, errOut io.Writer) ([]checkDiag, error) {
	// A file that fails to parse has no meaningful AST facts (and AnalyzePackageInDir
	// would bail on the whole package), so a syntax error short-circuits the sema and
	// depth stages: report it in the shared diagnostic format and stop here.
	if syntax := packageSyntaxDiags(pkg); len(syntax) > 0 {
		return syntax, nil
	}

	srcs := make([]string, len(pkg.Files))
	for i, f := range pkg.Files {
		srcs[i] = f.Src
	}
	perFile, err := sema.AnalyzePackageInDir(srcs, pkg.Dir)
	if err != nil {
		return nil, err
	}

	depth, derr := runDepthChecks(pkg, shared)
	var depthDiags []checkDiag
	if derr != nil {
		fmt.Fprintf(errOut, "goal check: depth stage unavailable for %s: %v\n", pkg.Dir, briefDepthErr(derr))
		// A full-package check has every sibling file, so a transpile failure is a real
		// compile error: surface it as a hard error so `check` never reports `ok` for
		// source that cannot build. A single-file check may fail only because a sibling
		// is out of view, so it keeps just the soft note above. The full `--- generated
		// ---` dump stays reserved for `goal build`.
		if strict {
			depthDiags = append(depthDiags, checkDiag{
				checkTargetFile(pkg), 1, 1, sema.Severity(sema.Severity_Error{}),
				"depth-transpile", briefDepthErr(derr), nil,
			})
		}
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
			// sema diagnostics carry Line/Col directly on Pos, so no offset-to-position
			// mapping is needed (unlike the legacy lexical checker's byte offsets).
			if suppress[dedupKey(path, d.Pos.Line, d.Feature)] {
				continue // type-backed finding owns this construct
			}
			diags = append(diags, checkDiag{path, d.Pos.Line, d.Pos.Col, d.Severity, d.Code, d.Message, d.Fix})
		}
	}
	for _, d := range depth {
		// The depth stage reuses the legacy check.Severity type; convert it to the
		// equivalent sema.Severity (same int ordering: Error=0, Warning=1) so both
		// stages render uniformly without cmd/goal importing internal/check.
		diags = append(diags, checkDiag{
			depthFilePath(pkg, d.Pos.Filename), d.Pos.Line, d.Pos.Column,
			sema.Severity(d.Severity), d.Code, d.Message, nil,
		})
	}
	return append(diags, depthDiags...), nil
}

// checkTargetFile is the file a package-level transpile failure is attributed to: the
// package's first source file (a transpile error carries no reliable position, and
// within-package basenames are unique). Falls back to the package directory.
func checkTargetFile(pkg *project.Package) string {
	if len(pkg.Files) > 0 {
		return pkg.Files[0].Path
	}
	return pkg.Dir
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
func runDepthChecks(pkg *project.Package, shared *typecheck.Importers) ([]typecheck.Diagnostic, error) {
	// Resolve depth diagnostics through the TypeChecker seam so the go/types crutch can be
	// swapped for a native goal checker later without changing this caller (US-028). The
	// shared importer caches each dependency's type-check across the whole run.
	var tc typecheck.TypeChecker = typecheck.GoTypesChecker{Shared: shared}
	return tc.Check(pkg)
}

// checkDiag is a stage-agnostic rendered finding, so the two stages' diagnostics order
// and print uniformly.
type checkDiag struct {
	file          string
	line, col     int
	severity      sema.Severity
	code, message string
	fix           *sema.SuggestedFix // machine-applyable repair, or nil (US-030)
}

// render formats the finding as `file:line:col: severity: [code] message`, matching both
// stages' native rendering.
func (d checkDiag) render() string {
	return fmt.Sprintf("%s:%d:%d: %s: [%s] %s", d.file, d.line, d.col, sema.SeverityLabel(d.severity), d.code, d.message)
}

// jsonDiag is the machine-readable shape of one diagnostic for `goal check --json`.
// The core fields are omitempty-free so every diagnostic serializes the full record;
// SuggestedFix (US-030) is optional and omitted for diagnostics with no known repair.
type jsonDiag struct {
	File         string            `json:"file"`
	Line         int               `json:"line"`
	Col          int               `json:"col"`
	Severity     string            `json:"severity"`
	Code         string            `json:"code"`
	Message      string            `json:"message"`
	SuggestedFix *jsonSuggestedFix `json:"suggestedFix,omitempty"`
}

// jsonSuggestedFix is the machine-applyable repair carried in --json: insert NewText at
// (Line, Col) — the front-end's 1-based source position of the character the text is
// inserted before. A consumer applies it as a pure insertion.
type jsonSuggestedFix struct {
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	NewText string `json:"newText"`
}

// toJSON projects a checkDiag into its machine-readable form, rendering the severity
// interface as its stable label ("error"/"warning").
func (d checkDiag) toJSON() jsonDiag {
	jd := jsonDiag{
		File:     d.file,
		Line:     d.line,
		Col:      d.col,
		Severity: sema.SeverityLabel(d.severity),
		Code:     d.code,
		Message:  d.message,
	}
	if d.fix != nil {
		jd.SuggestedFix = &jsonSuggestedFix{
			Line:    d.fix.Pos.Line,
			Col:     d.fix.Pos.Col,
			NewText: d.fix.NewText,
		}
	}
	return jd
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

// parseErrorDiags renders located parse errors as [syntax] findings at path, so a
// syntax error prints in the same `file:line:col: error: [code] message` format as
// every checker diagnostic.
func parseErrorDiags(path string, errs []*parser.Error) []checkDiag {
	diags := make([]checkDiag, 0, len(errs))
	for _, e := range errs {
		diags = append(diags, checkDiag{path, e.Pos.Line, e.Pos.Col, sema.Severity(sema.Severity_Error{}), "syntax", e.Msg, nil})
	}
	return diags
}

// packageSyntaxDiags parses each file in pkg and returns [syntax] diagnostics for
// any that fail, keyed to the file's full source path. It returns nil when every
// file parses cleanly. parser.CollectErrors deduplicates repeated trailing errors.
func packageSyntaxDiags(pkg *project.Package) []checkDiag {
	var diags []checkDiag
	for _, f := range pkg.Files {
		if _, perr := parser.ParseFile(f.Src); perr != nil {
			diags = append(diags, parseErrorDiags(f.Path, parser.CollectErrors(perr))...)
		}
	}
	return diags
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
// already mapped to .goal positions by the //line directives) is relayed verbatim —
// except for `build`, whose stderr is rewritten into the shared check diagnostic format
// (see rewriteGoBuildStderr). Only build is rewritten: `goal run` must relay the running
// program's own stderr untouched.
func goToolchain(root string, ts []transpiled, out, errOut io.Writer, verb, target string, progArgs ...string) error {
	overlayPath, cleanup, err := writeOverlay(ts)
	if err != nil {
		// Materializing the overlay is pure toolchain scaffolding (temp dir + JSON);
		// a failure here is internal, not a user-code diagnostic.
		return internalErr(err)
	}
	defer cleanup()
	flags := []string{"-overlay", overlayPath}
	if verb == "build" {
		// Capture the toolchain's stderr so Go-level build errors render as
		// `path/file.goal:line:col: error: [go-build] message`, parseable by the same
		// regex as check and syntax diagnostics.
		var buf bytes.Buffer
		runErr := runGo(root, flags, out, &buf, verb, target, progArgs...)
		if rendered := rewriteGoBuildStderr(ts, buf.String()); rendered != "" {
			fmt.Fprint(errOut, rendered)
		}
		return runErr
	}
	return runGo(root, flags, out, errOut, verb, target, progArgs...)
}

// goBuildLineRe matches a Go toolchain diagnostic line that carries a .goal source
// position. The Go tool emits line-only positions here (the backend's //line directives
// name the .goal basename with no column), so the column group is optional.
var goBuildLineRe = regexp.MustCompile(`^([^\s:]+\.goal):(\d+)(?::(\d+))?: (.*)$`)

// rewriteGoBuildStderr converts `go build`'s stderr into the shared check diagnostic
// format. Package-clause header lines (`# module/pkg`) are dropped; positioned lines
// become `[go-build]` diagnostics with the invocation-relative source path (mapped from
// the toolchain's basename back to the .goal path the user passed); tab/space-indented
// continuation lines (the have/want detail of a multi-line error) stay attached to their
// diagnostic; any other line is passed through after the structured lines so nothing is
// silently dropped. NOTE: within one build the .goal basenames are assumed unique across
// packages; a cross-package basename collision would map to the first-seen path.
func rewriteGoBuildStderr(ts []transpiled, raw string) string {
	pathByBase := map[string]string{}
	for _, t := range ts {
		for _, f := range t.pkg.Files {
			base := filepath.Base(f.Name)
			if _, seen := pathByBase[base]; !seen {
				pathByBase[base] = f.Path
			}
		}
	}

	var diags []checkDiag
	var passthrough []string
	last := -1 // index in diags of the most recent, for continuation lines
	for _, line := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			last = -1 // package header; the run that follows starts fresh
			continue
		}
		if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") {
			if last >= 0 {
				diags[last].message += " " + strings.TrimSpace(line)
			} else {
				passthrough = append(passthrough, line)
			}
			continue
		}
		if m := goBuildLineRe.FindStringSubmatch(line); m != nil {
			ln, _ := strconv.Atoi(m[2])
			col := 1
			if m[3] != "" {
				col, _ = strconv.Atoi(m[3])
			}
			path := m[1]
			if p, ok := pathByBase[filepath.Base(m[1])]; ok {
				path = p
			}
			diags = append(diags, checkDiag{path, ln, col, sema.Severity(sema.Severity_Error{}), "go-build", m[4], nil})
			last = len(diags) - 1
			continue
		}
		passthrough = append(passthrough, line)
		last = -1
	}

	var b strings.Builder
	for _, d := range diags {
		b.WriteString(d.render())
		b.WriteByte('\n')
	}
	for _, l := range passthrough {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.String()
}

// runGo invokes the go tool with the given verb, flags, and target from dir.
// Any progArgs are appended after the target, so the go tool grammar
// `go <verb> [build flags] <target> [program args...]` is preserved and the
// trailing tokens reach the running program rather than the go tool.
func runGo(dir string, flags []string, out, errOut io.Writer, verb, target string, progArgs ...string) error {
	args := append([]string{verb}, flags...)
	args = append(args, target)
	args = append(args, progArgs...)
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
		// Package Go and the doctest sidecars are both overlaid. The sidecars
		// (`<file>_test.go`) are harmless for build/run — the Go tool ignores
		// `_test.go` outside `go test` — and are what `goal test` runs.
		for _, gf := range append(append([]pipeline.GoFile{}, t.out.Files...), t.out.Tests...) {
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

// goalGeneratedMarker is the exact first line goal writes atop every emitted .go
// file. It is the single, version-independent ownership key for the emit
// lifecycle: both header emission (goalGeneratedHeader) and the ownership
// predicate (IsGoalGenerated) reference this one definition, so prune (US-005)
// and clean (US-006) can never disagree about what goal generated.
const goalGeneratedMarker = "// Code generated by goal; DO NOT EDIT."

// goalGeneratedHeader returns the goal-owned generated-code header for the given
// build version, terminated by a blank line, to be prepended to every emitted .go
// file. It is deterministic — version only, no commit or date — so repeated emits
// of identical source are byte-identical. The line-1 marker is what identifies a
// file as goal-generated for the emit lifecycle (prune/clean); it is intentionally
// stable and independent of the version on line 2.
func goalGeneratedHeader(version string) string {
	return goalGeneratedMarker + "\n" +
		"// generator: goal v" + version + "\n\n"
}

// IsGoalGenerated reports whether the given Go file content was generated by
// goal. It keys solely on the goal line-1 marker (goalGeneratedMarker) and is
// independent of the version string on line 2, so a file emitted by any goal
// version — "dev" or any release — still matches and can be reclaimed by prune
// or clean. Only the file's first line is considered: a goal-looking marker
// anywhere below line 1 does not qualify. Total function: any input, including
// empty content, yields a boolean and never panics.
func IsGoalGenerated(content []byte) bool {
	firstLine := content
	if i := bytes.IndexByte(content, '\n'); i >= 0 {
		firstLine = content[:i]
	}
	// Tolerate a Windows CRLF ending on the marker line.
	firstLine = bytes.TrimSuffix(firstLine, []byte("\r"))
	return string(firstLine) == goalGeneratedMarker
}

// goalManifestName is the fixed basename of the emit manifest goal writes at the
// emit root after a successful `goal build --emit`. It records which .go files
// goal produced so a later build can detect orphaned generated files even when
// their source package is gone (US-004 warn / US-005 prune consume it). It is
// not itself a .go file, so it never appears in its own file list.
const goalManifestName = ".goal-manifest.json"

// emitManifest is the on-disk record of one `goal build --emit` run: the
// generator name, the build version, and the module-relative paths of every .go
// file emitted in that run (sorted and deduped, so the file is deterministic).
type emitManifest struct {
	Generator string   `json:"generator"`
	Version   string   `json:"version"`
	Files     []string `json:"files"`
}

// emitManifestRoot is the directory the manifest is written under, and the base
// the manifest's paths are relative to: the emit target dir when `--emit=dir`
// was given, else the enclosing Go module root (so paths are module-relative),
// falling back to the first package's dir when no module is found.
func emitManifestRoot(ts []transpiled, emitDir string) string {
	if emitDir != "" {
		return emitDir
	}
	if len(ts) > 0 {
		if modDir, _, ok := moduleInfo(ts[0].pkg.Dir); ok {
			return modDir
		}
		return ts[0].pkg.Dir
	}
	return "."
}

// manifestRelPaths turns the absolute-ish emitted paths into root-relative,
// slash-separated paths, deduped and sorted for deterministic manifest output.
func manifestRelPaths(root string, emitted []string) []string {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	seen := map[string]bool{}
	rels := make([]string, 0, len(emitted))
	for _, p := range emitted {
		absP, err := filepath.Abs(p)
		if err != nil {
			absP = p
		}
		rel, err := filepath.Rel(absRoot, absP)
		if err != nil {
			rel = p
		}
		rel = filepath.ToSlash(rel)
		if !seen[rel] {
			seen[rel] = true
			rels = append(rels, rel)
		}
	}
	sort.Strings(rels)
	return rels
}

// writeEmitManifest writes the emit manifest (goalManifestName) at root, recording
// the goal generator, the build version, and the given module-relative file set.
func writeEmitManifest(root string, files []string) error {
	m := emitManifest{Generator: "goal", Version: version, Files: files}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, goalManifestName), data, 0o644)
}

// loadEmitManifest reads and parses an emit manifest from path, so a later build
// (or a test) can round-trip the recorded file set.
func loadEmitManifest(path string) (*emitManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m emitManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// emitFiles writes the generated Go to disk: beside each .goal by default, or mirrored
// under emitDir when given. Test sidecars are written too so doctests can run. Every
// file is prefixed with the goal-generated header (see goalGeneratedHeader). After a
// successful run it warns (to errOut) about any orphaned generated files (see
// warnOrphanedFiles) and writes an emit manifest (goalManifestName) at the emit root
// recording every .go path produced, so a later build can detect orphaned output.
func emitFiles(ts []transpiled, emitDir string, out, errOut io.Writer) error {
	header := goalGeneratedHeader(version)
	var emitted []string
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
			if err := os.WriteFile(p, []byte(header+gf.Go), 0o644); err != nil {
				return err
			}
			fmt.Fprintln(out, "emitted", p)
			emitted = append(emitted, p)
		}
	}
	root := emitManifestRoot(ts, emitDir)
	current := manifestRelPaths(root, emitted)
	// Warn about orphans BEFORE overwriting the prior manifest — the diff needs
	// the previous run's file set, which writeEmitManifest is about to replace.
	warnOrphanedFiles(root, current, errOut)
	return writeEmitManifest(root, current)
}

// warnOrphanedFiles reports, to errOut, any goal-generated .go file that a prior
// emit produced but this run did not (an orphan): its source is gone, so the
// stale .go would silently break the later Go build. An orphan is a path in the
// prior manifest that (a) is absent from the current emit set, (b) still exists
// on disk, and (c) still passes the goal ownership matcher (IsGoalGenerated) — a
// file whose header a user removed by hand is no longer goal-owned and is left
// alone. This is detect-and-surface only: nothing is ever deleted, and the
// absence of a prior manifest (first emit) yields no warnings and no error, so
// the build's exit status is unaffected either way. Deletion is the opt-in job
// of `--prune` (US-005) and `goal clean` (US-006).
func warnOrphanedFiles(root string, current []string, errOut io.Writer) {
	prior, err := loadEmitManifest(filepath.Join(root, goalManifestName))
	if err != nil {
		return // no prior manifest (or unreadable) — nothing to compare against
	}
	inCurrent := make(map[string]bool, len(current))
	for _, rel := range current {
		inCurrent[rel] = true
	}
	for _, rel := range prior.Files {
		if inCurrent[rel] {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(rel))
		content, err := os.ReadFile(p)
		if err != nil {
			continue // orphan candidate no longer on disk
		}
		if !IsGoalGenerated(content) {
			continue // header removed by hand — not goal-owned anymore
		}
		fmt.Fprintln(errOut, "warning: orphaned generated file "+p)
	}
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

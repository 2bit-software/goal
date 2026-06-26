package analyze

// Foreign-type enrichment: read the field sets of struct types declared in imported
// Go packages so feature 12 (derive-convert) can resolve, prove total, and lower a
// `derive func` whose source or target is an out-of-package type (e.g. a generated
// protobuf message `*hobv1.EnvironmentSpec`).
//
// The rest of analyze is purely lexical and IO-free — Build(src) turns one source
// string into name-keyed tables and never touches the disk. Enrichment is the one
// exception: it must resolve an import path to a directory and parse that package's
// Go source. It is therefore a SEPARATE, explicitly-IO entry point (EnrichForeign),
// called only by the package driver, so Build and every check/pass test stay offline.
// A single-file Transpile/Analyze has no package directory and so stays foreign-blind,
// exactly as before — completeness for an out-of-package type is still deferred there.
//
// Reading the foreign field set, not type-checking it: the user chose the lexical
// route (parse imported Go with stdlib go/parser) over loading go/types. So we read
// each imported package's `type X struct {…}` declarations, keep the EXPORTED fields,
// and key them by the qualifier the goal source uses (`alias.Type`). A field's type is
// re-rendered qualified by that same alias — a package-local `*Workspace` in the
// foreign source becomes `*alias.Workspace`, matching how the goal source names it —
// so the registry and recursion lookups in the derive pass align by string, exactly
// as they do for in-file structs.
//
// Foundation only (per the agreed scope): proto getters, enum→sum and oneof→sum
// bridging are NOT modeled here. A field whose type does not resolve nominally (an
// enum, a oneof wrapper, a cross-package selector) is left to the author's `from func`
// registration or explicit override, and derive defers/errors on it as it already
// does for any unbridged field.

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"goal/internal/scan"
)

// ImportSpec is one entry of a `.goal` file's import block: the qualifier the source
// uses (Alias, empty when the import has no explicit alias) and the import Path.
type ImportSpec struct {
	Alias string
	Path  string
}

// DirResolver maps an import path to the directory holding that package's Go source,
// resolved relative to fromDir. It is the one IO dependency of enrichment, injected so
// tests can resolve against a fixture directory without the go toolchain. DefaultResolver
// is the production implementation.
type DirResolver func(importPath, fromDir string) (string, error)

// EnrichForeign augments t with the struct field sets of imported Go packages that a
// `derive func` or `from func` in srcs references by qualifier, so feature 12 can
// resolve out-of-package source/target types. fromDir is the goal package's directory
// (import paths resolve relative to it). resolve may be nil, in which case
// DefaultResolver is used. It mutates t in place and returns any per-import errors
// (resolution or parse failures), which are non-fatal: an unresolved import simply
// leaves its types unknown, and derive defers as before.
func EnrichForeign(t *Tables, srcs []string, fromDir string, resolve DirResolver) []error {
	if resolve == nil {
		resolve = DefaultResolver
	}
	needed := neededAliases(srcs)
	for alias := range questionCalleeAliases(srcs) {
		needed[alias] = true
	}
	for _, src := range srcs {
		for alias := range QuestionMethodReceiverPkgs(src) {
			needed[alias] = true // load packages whose types host a `recv.Method()?` receiver
		}
	}
	if len(needed) == 0 {
		return nil // nothing references a foreign type (derive/from) or `?` callee — nothing to load
	}
	var errs []error
	loaded := map[string]bool{} // import paths already merged, for dedupe across files
	for _, src := range srcs {
		for _, imp := range ParseImports(src) {
			alias := imp.Alias
			if alias == "" {
				alias = lastSegment(imp.Path)
			}
			if !needed[alias] || loaded[imp.Path] {
				continue
			}
			loaded[imp.Path] = true
			dir, err := resolve(imp.Path, fromDir)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			_, structs, funcs, methods, err := foreignDecls(dir, imp.Alias)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for name, fields := range structs {
				t.Structs[name] = fields
				t.TypeDecls[name] = "struct"
			}
			for name, sig := range funcs {
				// Mode is left at its zero value (ModeNone): a foreign entry carries only the
				// `?`-relevant facts (arity and whether it ends in error), so existing
				// iterations over FuncSignatures (e.g. NeedsResultPrelude) ignore it.
				t.FuncSignatures[name] = sig
			}
			for name, sig := range methods {
				t.ForeignMethods[name] = sig
			}
		}
	}
	return errs
}

// questionCalleeAliases returns the package qualifiers that head a `?` callee
// (`alias.Func(...)?`) anywhere in srcs, so an import referenced only through `?` propagation
// — with no derive/from use — is still loaded for its function arities.
func questionCalleeAliases(srcs []string) map[string]bool {
	out := map[string]bool{}
	for _, src := range srcs {
		toks := scan.Lex(src)
		for q := range toks {
			if toks[q].Text != "?" {
				continue
			}
			lineStart := strings.LastIndexByte(src[:toks[q].Start], '\n') + 1
			_, rhs, _ := scan.SplitAssign(src[lineStart:toks[q].Start])
			if alias, _, ok := strings.Cut(scan.CalleeKey(rhs), "."); ok {
				out[alias] = true
			}
		}
	}
	return out
}

// ParseImports returns the entries of a `.goal` file's import block(s). It lexes rather
// than regexps so an `import` keyword inside a string or comment is never mistaken for
// the clause. Both the parenthesized block form and the single-import form are handled.
func ParseImports(src string) []ImportSpec {
	toks := scan.Lex(src)
	var specs []ImportSpec
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "import" {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			closeIdx := scan.MatchParen(toks, i+1)
			if closeIdx < 0 {
				continue
			}
			for j := i + 2; j < closeIdx; {
				j = parseOneImport(toks, j, closeIdx, &specs)
			}
			i = closeIdx
			continue
		}
		parseOneImport(toks, i+1, len(toks), &specs)
	}
	return specs
}

// parseOneImport reads a single `[alias] "path"` entry starting at j (bounded by limit)
// and appends it to specs, returning the index just past the entry. A leading identifier
// (or `_`/`.`) before the string literal is the explicit alias.
func parseOneImport(toks []scan.Token, j, limit int, specs *[]ImportSpec) int {
	if j >= limit {
		return limit
	}
	if path, ok := importPath(toks[j].Text); ok {
		*specs = append(*specs, ImportSpec{Path: path})
		return j + 1
	}
	alias := toks[j].Text
	if !(scan.IsIdent(alias) || alias == "_" || alias == ".") {
		return j + 1 // not the start of an import entry; step over it
	}
	if j+1 < limit {
		if path, ok := importPath(toks[j+1].Text); ok {
			*specs = append(*specs, ImportSpec{Alias: alias, Path: path})
			return j + 2
		}
	}
	return j + 1
}

// importPath returns the unquoted path of a string-literal token and whether tok was a
// string literal (an import path is always a double-quoted or raw string).
func importPath(tok string) (string, bool) {
	if len(tok) < 2 || (tok[0] != '"' && tok[0] != '`') {
		return "", false
	}
	if p, err := strconv.Unquote(tok); err == nil {
		return p, true
	}
	return "", false
}

// neededAliases returns the set of package qualifiers used in a TYPE position of any
// `derive func` or `from func` across srcs (the source param type and the return type).
// Enrichment loads only the imports these name, so a package with no derive/from use
// pays nothing and a huge unrelated dependency is never parsed. Inside a signature a
// `name.Member` only ever names a qualified type (parameter names are not dereferenced
// there), so collecting every `ident .` qualifier in the signature span is precise.
func neededAliases(srcs []string) map[string]bool {
	out := map[string]bool{}
	for _, src := range srcs {
		toks := scan.Lex(src)
		for i := 0; i+1 < len(toks); i++ {
			if toks[i+1].Text != "func" || (toks[i].Text != "derive" && toks[i].Text != "from") {
				continue
			}
			open := indexOf(toks, i+2, "(")
			if open < 0 {
				continue
			}
			closeP := scan.MatchParen(toks, open)
			if closeP < 0 {
				continue
			}
			end := scan.FirstBodyBrace(toks, i)
			if end < 0 {
				end = len(toks)
			}
			for k := open + 1; k < end && k+1 < len(toks); k++ {
				if scan.IsIdent(toks[k].Text) && toks[k+1].Text == "." {
					out[toks[k].Text] = true
				}
			}
		}
	}
	return out
}

// foreignDecls parses the Go source files in dir and returns its exported struct types
// (keyed `alias.Type`, each field's type rendered qualified by alias) and its exported,
// receiver-less function return arities (keyed `alias.Func`). requestedAlias is the qualifier
// the goal source uses; when empty (an unaliased import) the package's own declared name is
// used. The effective alias is also returned.
func foreignDecls(dir, requestedAlias string) (alias string, structs map[string][]Field, funcs, methods map[string]FuncSig, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, nil, nil, err
	}
	fset := token.NewFileSet()
	var files []*ast.File
	pkgName := ""
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.SkipObjectResolution)
		if perr != nil {
			continue // tolerate an unparseable sibling; read what we can
		}
		if pkgName == "" {
			pkgName = f.Name.Name
		}
		files = append(files, f)
	}
	alias = requestedAlias
	if alias == "" {
		alias = pkgName
	}
	structs = map[string][]Field{}
	funcs = map[string]FuncSig{}
	methods = map[string]FuncSig{}
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					structs[alias+"."+ts.Name.Name] = exportedFields(st, alias)
				}
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				sig := FuncSig{Arity: resultArity(d.Type), EndsInError: endsInErrorAST(d.Type)}
				if d.Recv == nil {
					funcs[alias+"."+d.Name.Name] = sig
				} else if base := foreignRecvBase(d.Recv); base != "" {
					// `os.File.Close` etc. — keyed the way goal source spells the receiver type,
					// so a `recv.Method()?` whose receiver is `*os.File` resolves to it.
					methods[alias+"."+base+"."+d.Name.Name] = sig
				}
			}
		}
	}
	return alias, structs, funcs, methods, nil
}

// foreignRecvBase returns the bare receiver type name of a foreign method (`*File` -> "File",
// `Tree[T]` -> "Tree"), or "" when it can't be read.
func foreignRecvBase(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	expr := recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	switch e := expr.(type) {
	case *ast.IndexExpr: // generic receiver Type[T]
		expr = e.X
	case *ast.IndexListExpr: // generic receiver Type[T, U]
		expr = e.X
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// endsInErrorAST reports whether a foreign function's last result is the type `error` — the
// failure a `?` propagates. A named group's last field (`… err error`) counts by its type.
func endsInErrorAST(ft *ast.FuncType) bool {
	if ft.Results == nil || len(ft.Results.List) == 0 {
		return false
	}
	last := ft.Results.List[len(ft.Results.List)-1]
	id, ok := last.Type.(*ast.Ident)
	return ok && id.Name == "error"
}

// resultArity reports how many values a foreign function returns: an unnamed result counts
// once and a named group (`(a, b int)`) counts by name.
func resultArity(ft *ast.FuncType) int {
	if ft.Results == nil {
		return 0
	}
	n := 0
	for _, field := range ft.Results.List {
		if len(field.Names) == 0 {
			n++
			continue
		}
		n += len(field.Names)
	}
	return n
}

// exportedFields returns the exported, named fields of a struct (embedded and unexported
// fields are skipped), each typed via goTypeString so package-local type references are
// qualified by alias.
func exportedFields(st *ast.StructType, alias string) []Field {
	var fields []Field
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue // embedded field — unsupported by the nominal derive model
		}
		typ := goTypeString(f.Type, alias)
		for _, n := range f.Names {
			if n.IsExported() {
				fields = append(fields, Field{Name: n.Name, Type: typ})
			}
		}
	}
	return fields
}

// goTypeString renders an AST type expression as goal-source text, qualifying every
// package-local named type with alias (so the foreign package's `*Workspace` reads as
// the goal source's `*alias.Workspace`). Predeclared types are left bare; an
// already-qualified `pkg.T` keeps its own qualifier; an anonymous/unhandled shape falls
// back to its printed form so an unbridgeable field is surfaced rather than dropped.
func goTypeString(expr ast.Expr, alias string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if isGoBuiltin(e.Name) {
			return e.Name
		}
		return alias + "." + e.Name
	case *ast.StarExpr:
		return "*" + goTypeString(e.X, alias)
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + goTypeString(e.Elt, alias)
		}
		return "[" + exprText(e.Len) + "]" + goTypeString(e.Elt, alias)
	case *ast.MapType:
		return "map[" + goTypeString(e.Key, alias) + "]" + goTypeString(e.Value, alias)
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return exprText(expr)
	case *ast.InterfaceType:
		return "any"
	case *ast.Ellipsis:
		return "..." + goTypeString(e.Elt, alias)
	default:
		return exprText(expr)
	}
}

// exprText prints an AST expression to source text via go/format, a syntactic fallback
// for type shapes goTypeString does not special-case.
func exprText(expr ast.Expr) string {
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), expr); err != nil {
		return ""
	}
	return b.String()
}

// isGoBuiltin reports whether name is a predeclared Go type, which must not be qualified.
func isGoBuiltin(name string) bool {
	switch name {
	case "bool", "string", "error", "any", "byte", "rune", "uintptr",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128":
		return true
	}
	return false
}

// DefaultResolver resolves an import path to its package directory. It first tries a
// same-module resolution (walk up from fromDir to go.mod and map a path under the module
// onto the tree) — offline, deterministic, and the common case for a project's own
// generated code. It falls back to `go list` for an external module (resolved through
// the module cache), matching how the rest of the toolchain shells out to `go`.
func DefaultResolver(importPath, fromDir string) (string, error) {
	if dir, ok := moduleResolve(importPath, fromDir); ok {
		return dir, nil
	}
	return goListResolve(importPath, fromDir)
}

// moduleResolve maps importPath onto the local module tree by finding the nearest go.mod
// at or above fromDir, reading its module path, and joining the path's tail. It reports
// ok=false when there is no enclosing module, the path is outside it, or the computed
// directory does not exist (so the caller falls back to `go list`).
func moduleResolve(importPath, fromDir string) (string, bool) {
	dir, err := filepath.Abs(fromDir)
	if err != nil {
		return "", false
	}
	for {
		modPath, ok := readModulePath(filepath.Join(dir, "go.mod"))
		if ok {
			if importPath == modPath {
				return dir, isDir(dir)
			}
			if rest, under := strings.CutPrefix(importPath, modPath+"/"); under {
				cand := filepath.Join(dir, filepath.FromSlash(rest))
				return cand, isDir(cand)
			}
			return "", false // inside a module, but path belongs to another module
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// readModulePath returns the module path declared by a go.mod file and whether the file
// was read and had a `module` directive.
func readModulePath(goMod string) (string, bool) {
	data, err := os.ReadFile(goMod)
	if err != nil {
		return "", false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module"); ok {
			return strings.TrimSpace(rest), true
		}
	}
	return "", false
}

// goListResolve asks the go tool for an import path's directory, run from fromDir so the
// module graph (requires, replaces, the cache) is the project's own.
func goListResolve(importPath, fromDir string) (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", "--", importPath)
	cmd.Dir = fromDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// isDir reports whether path exists and is a directory.
func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// lastSegment returns the final path element of a slash-separated import path, the
// conventional package name used as the qualifier for an unaliased import.
func lastSegment(importPath string) string {
	if i := strings.LastIndexByte(importPath, '/'); i >= 0 {
		return importPath[i+1:]
	}
	return importPath
}

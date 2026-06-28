package pipeline

import (
	"fmt"
	"strings"

	"goal/internal/ast"
	"goal/internal/parser"
	"goal/internal/token"
)

// AddLineDirectives inserts a Go `//line` directive before every top-level declaration
// in the generated Go, so the Go toolchain reports errors at goal positions (SPIKE-1
// proved the compiler honors these and gofmt preserves them).
//
// Granularity is per-declaration (BUILD-MODEL-TODO U5): a generated decl whose name
// matches a user declaration in the goal source is anchored to that source line — names
// survive lowering, so the match is by name; an ordinary Go type error in a passed-
// through function body then reports against the right `.goal` line. A generated decl
// with no goal counterpart (the enum encoding, a `var _` assertion, an injected import)
// re-anchors to the generated file at its own line, so its numbering stays truthful
// rather than inheriting the previous mapped decl's goal line.
//
// goalFile is the path reported for mapped decls; genFile is the generated file's name
// (its path in the build/emit directory) reported for synthesized decls.
//
// It is name-based and therefore engine-agnostic: both the splice front-end and the
// AST backend (backend.TranspilePackage) apply it to their formatted Go output.
func AddLineDirectives(goalSrc, genGo, goalFile, genFile string) string {
	goalLine := declLines(goalSrc)
	sites := declSites(genGo)
	if len(sites) == 0 {
		return genGo
	}

	var b strings.Builder
	prev := 0
	for _, s := range sites {
		b.WriteString(genGo[prev:s.off]) // text up to the decl (ends at a line start)
		physical := strings.Count(b.String(), "\n") + 1 // line the directive occupies
		if ln, ok := goalLine[s.name]; ok {
			fmt.Fprintf(&b, "//line %s:%d\n", goalFile, ln)
		} else {
			// Identity: the decl lands on the line after this directive (physical+1).
			fmt.Fprintf(&b, "//line %s:%d\n", genFile, physical+1)
		}
		prev = s.off
	}
	b.WriteString(genGo[prev:])
	return b.String()
}

// declSite is one top-level declaration: the byte offset where its keyword begins and
// the declared name ("" or "_" when there is nothing to map, e.g. an import or a
// `var _ I = T{}` assertion).
type declSite struct {
	off  int
	name string
}

// declSites returns every top-level declaration in src, in source order. It parses
// src with the goal parser (a superset of Go, so it reads both the goal source and
// the generated Go output) and reads each declaration's keyword offset from the AST.
// In gofmt'd Go every top-level declaration starts its own line, so the keyword
// offset is a line start — the position a `//line` directive must precede.
func declSites(src string) []declSite {
	file, _ := parser.ParseFile(src)
	if file == nil {
		return nil
	}
	sites := make([]declSite, 0, len(file.Decls))
	for _, d := range file.Decls {
		sites = append(sites, declSite{off: d.Pos().Offset, name: declName(d)})
	}
	return sites
}

// declLines maps each named top-level declaration in src to its 1-based line. On a
// duplicate name the first wins (deterministic); unnamed decls are skipped.
func declLines(src string) map[string]int {
	lines := map[string]int{}
	for _, s := range declSites(src) {
		if s.name == "" || s.name == "_" {
			continue
		}
		if _, dup := lines[s.name]; !dup {
			lines[s.name] = strings.Count(src[:s.off], "\n") + 1
		}
	}
	return lines
}

// declName returns the name a top-level declaration introduces, or "" when there is
// none to map (an import, or a grouped/multi-name const/var). A method's receiver is
// not its name, so a FuncDecl maps to its function/method name directly.
func declName(d ast.Decl) string {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Name != nil {
			return d.Name.Name
		}
	case *ast.EnumDecl:
		if d.Name != nil {
			return d.Name.Name
		}
	case *ast.SealedInterfaceDecl:
		if d.Name != nil {
			return d.Name.Name
		}
	case *ast.GenDecl:
		// Imports introduce no mappable name; a const/var/type maps only when it
		// declares a single name (a grouped block has no single anchor).
		if d.Tok == token.IMPORT || len(d.Specs) != 1 {
			return ""
		}
		switch s := d.Specs[0].(type) {
		case *ast.TypeSpec:
			if s.Name != nil {
				return s.Name.Name
			}
		case *ast.ValueSpec:
			if len(s.Names) == 1 && s.Names[0] != nil {
				return s.Names[0].Name
			}
		}
	}
	return ""
}

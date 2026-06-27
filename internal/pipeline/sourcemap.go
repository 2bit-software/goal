package pipeline

import (
	"fmt"
	"strings"

	"goal/internal/scan"
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

// declSites returns every top-level declaration in src, in source order. Top-level
// means a declaration keyword at brace/paren/bracket depth 0 that begins its line.
func declSites(src string) []declSite {
	toks := scan.Lex(src)
	var sites []declSite
	depth := 0
	for i := range toks {
		switch toks[i].Text {
		case "{", "(", "[":
			depth++
			continue
		case "}", ")", "]":
			depth--
			continue
		}
		if depth != 0 || !isDeclKeyword(toks[i].Text) || !scan.IsLineStart(src, toks[i].Start) {
			continue
		}
		sites = append(sites, declSite{off: toks[i].Start, name: declName(toks, i)})
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

// isDeclKeyword reports whether t begins a top-level declaration. `enum` is goal-only
// (it never appears in generated Go); including it lets one scanner serve both sides.
func isDeclKeyword(t string) bool {
	switch t {
	case "func", "type", "var", "const", "import", "enum":
		return true
	}
	return false
}

// declName returns the name introduced by the declaration whose keyword is toks[i], or
// "" when there is none. A method's receiver is skipped so the name is the method name.
func declName(toks []scan.Token, i int) string {
	switch toks[i].Text {
	case "func":
		j := i + 1
		if j < len(toks) && toks[j].Text == "(" {
			j = scan.MatchParen(toks, j) + 1 // skip receiver
		}
		if j < len(toks) && scan.IsIdent(toks[j].Text) {
			return toks[j].Text
		}
	case "type", "var", "const", "enum":
		if i+1 < len(toks) && scan.IsIdent(toks[i+1].Text) {
			return toks[i+1].Text
		}
	}
	return ""
}

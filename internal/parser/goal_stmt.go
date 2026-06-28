package parser

// This file implements the parser half of goal's remaining constructs (US-023):
// the `assert` statement, the `from`/`derive` function modifier, and the `///`
// doc-comment / doctest surface. The corresponding AST nodes live in
// internal/ast/goal_stmt.go (AssertStmt, DocComment, Doctest) and the from/derive
// modifier is recorded on ast.FuncDecl (Mod/ModPos).

import (
	"strings"

	"goal/internal/ast"
	"goal/internal/token"
)

// parseAssertStmt parses a goal `assert` statement in both forms:
//
//	assert <cond>                              // bare
//	assert <cond>, <format>, <args...>         // printf-message
//
// The condition is parsed as a full expression, which stops at the first
// top-level comma (a comma is not a binary operator), so commas nested inside
// the condition — e.g. a call's arguments `clamp(lo, hi, n)` — stay part of
// Cond. Only that first top-level comma separates the format string; any further
// top-level commas separate the format arguments.
func (p *parser) parseAssertStmt() *ast.AssertStmt {
	kw := p.expect(token.ASSERT)
	s := &ast.AssertStmt{Assert: kw.Pos}
	s.Cond = p.parseExpr()
	if p.at(token.COMMA) {
		comma := p.advance()
		s.Comma = comma.Pos
		s.Msg = p.parseExpr()
		for p.at(token.COMMA) {
			p.advance()
			s.Args = append(s.Args, p.parseExpr())
		}
	}
	return s
}

// parseModFuncDecl parses a `from func` / `derive func` declaration. The cursor
// is on the contextual modifier IDENT (`from` or `derive`), with the FUNC
// keyword next. The modifier is consumed and recorded on the FuncDecl; the rest
// of the declaration (bodyless or bodied) parses via the ordinary parseFuncDecl,
// which leaves Body nil when no `{` follows the signature.
func (p *parser) parseModFuncDecl(mod ast.FuncMod) *ast.FuncDecl {
	modTok := p.advance() // consume `from` / `derive`
	fd := p.parseFuncDecl()
	fd.Mod = mod
	fd.ModPos = modTok.Pos
	return fd
}

// collectDoc consumes a run of consecutive `///` DOC_COMMENT tokens at the
// cursor and returns them as an *ast.DocComment, with the `///` prefix (and at
// most one following space) stripped from each line and any `>>>` doctests
// extracted. It returns nil when the cursor is not on a doc comment.
func (p *parser) collectDoc() *ast.DocComment {
	if !p.at(token.DOC_COMMENT) {
		return nil
	}
	doc := &ast.DocComment{Slash: p.cur().Pos}
	for p.at(token.DOC_COMMENT) {
		t := p.advance()
		doc.Lines = append(doc.Lines, stripDocPrefix(t.Lit))
	}
	doc.Doctests = extractDoctests(doc.Lines)
	return doc
}

// stripDocPrefix removes the leading `///` and at most one following space from
// a raw doc-comment line literal.
func stripDocPrefix(lit string) string {
	s := strings.TrimPrefix(lit, "///")
	if strings.HasPrefix(s, " ") {
		s = s[1:]
	}
	return s
}

// extractDoctests scans prefix-stripped doc lines for `>>>` examples. A line
// whose trimmed text starts with `>>>` opens a new doctest whose Input is the
// text after the `>>>`; the following non-`>>>` lines are its expected output,
// up to the next `>>>` or the end of the run. Lines before the first `>>>` are
// prose and contribute no doctest.
func extractDoctests(lines []string) []*ast.Doctest {
	var out []*ast.Doctest
	var cur *ast.Doctest
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if rest, ok := strings.CutPrefix(t, ">>>"); ok {
			cur = &ast.Doctest{Input: strings.TrimSpace(rest)}
			out = append(out, cur)
			continue
		}
		if cur != nil {
			cur.Expected = append(cur.Expected, t)
		}
	}
	return out
}

// This file implements goal's `match` parsing (REWRITE-ARCHITECTURE §1.4): the
// match expression, its arms, and the arm patterns. `match` is parsed by a
// single parseMatchExpr that returns an *ast.MatchExpr (an Expr), so it works in
// both positions:
//
//   - statement position — parseStmt dispatches token.MATCH to a match
//     expression wrapped in an *ast.ExprStmt (the AST has no separate MatchStmt);
//   - value/expression position — parseOperand dispatches token.MATCH so
//     `var x = match s { … }` and `return match s { … }` parse. Value-position
//     match is refused by the legacy splice approach; here it is representable.
//
// The subject is parsed with composite-literal braces suppressed (exprLev < 0),
// exactly as if/for/switch headers are, so the `{` after the subject is taken as
// the arms brace rather than a composite-literal body.
package parser

import (
	"goal/internal/ast"
	"goal/internal/token"
)

// parseMatchExpr parses `match Subject { Arm... }` into an *ast.MatchExpr. The
// subject is parsed with composite-literal braces suppressed so the trailing
// `{` opens the arms block, not a composite literal.
func (p *parser) parseMatchExpr() *ast.MatchExpr {
	matchTok := p.expect(token.MATCH)
	m := &ast.MatchExpr{Match: matchTok.Pos}

	prev := p.exprLev
	p.exprLev = -1
	m.Subject = p.parseExpr()
	p.exprLev = prev

	lb := p.expect(token.LBRACE)
	m.Lbrace = lb.Pos
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		m.Arms = append(m.Arms, p.parseMatchArm())
	}
	rb := p.expect(token.RBRACE)
	m.Rbrace = rb.Pos
	return m
}

// parseMatchArm parses one `Pattern => Body` arm. The body is a block when it
// opens with `{`, otherwise an expression (value-position arm).
func (p *parser) parseMatchArm() *ast.MatchArm {
	arm := &ast.MatchArm{Pattern: p.parsePattern()}
	arrow := p.expect(token.FAT_ARROW)
	arm.Arrow = arrow.Pos
	if p.at(token.LBRACE) {
		arm.Body = p.parseBlock()
	} else {
		arm.Body = p.parseExpr()
	}
	return arm
}

// parsePattern parses a match-arm pattern: the catch-all `_` rest pattern or a
// variant pattern.
func (p *parser) parsePattern() ast.Expr {
	if p.at(token.IDENT) && p.cur().Lit == "_" {
		t := p.advance()
		return &ast.RestPattern{Underscore: t.Pos}
	}
	return p.parseVariantPattern()
}

// parseVariantPattern parses `Enum.Variant`, `Enum.Variant(binding)`, or a bare
// `Variant`. The last dotted segment is the variant tag; the prefix (if any) is
// the enum reference (an *ast.Ident or, when further qualified, a
// *ast.SelectorExpr). An optional parenthesized identifier binds the payload.
func (p *parser) parseVariantPattern() ast.Expr {
	vp := &ast.VariantPattern{Variant: p.ident()}
	for p.at(token.PERIOD) {
		p.advance()
		next := p.ident()
		// The previously-held tag is part of the enum path, not the variant.
		if vp.Enum == nil {
			vp.Enum = vp.Variant
		} else {
			vp.Enum = &ast.SelectorExpr{X: vp.Enum, Sel: vp.Variant}
		}
		vp.Variant = next
	}
	if p.at(token.LPAREN) {
		lp := p.advance()
		vp.Lparen = lp.Pos
		if p.at(token.IDENT) {
			vp.Binding = p.ident()
		}
		rp := p.expect(token.RPAREN)
		vp.Rparen = rp.Pos
	}
	return vp
}

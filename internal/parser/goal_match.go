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
// opens with `{`; a statement when it begins with a statement keyword
// (`=> return true`, `=> for { … }`, etc.); otherwise an expression
// (value-position arm such as `=> 1`). MatchArm.Body is a generic ast.Node, so
// all three forms fit.
func (p *parser) parseMatchArm() *ast.MatchArm {
	arm := &ast.MatchArm{Pattern: p.parsePattern()}
	arrow := p.expect(token.FAT_ARROW)
	arm.Arrow = arrow.Pos
	switch {
	case p.at(token.LBRACE):
		arm.Body = p.parseBlock()
	case startsArmStmt(p.kind()):
		arm.Body = p.parseStmt()
	default:
		// A value-position arm body stops at a newline-leading binary operator so a
		// following `*T` type-pattern arm is not absorbed as a multiplication.
		prev := p.armBody
		p.armBody = true
		arm.Body = p.parseExpr()
		p.armBody = prev
	}
	return arm
}

// startsArmStmt reports whether k begins a non-expression statement that may be a
// brace-less match-arm body (e.g. `=> return x`). Expression-valued tokens —
// including `match` (a value-position match arm) — are excluded so they parse as
// expression arm bodies via parseExpr.
func startsArmStmt(k token.Kind) bool {
	switch k {
	case token.RETURN, token.IF, token.FOR, token.SWITCH, token.DEFER, token.GO,
		token.BREAK, token.CONTINUE, token.GOTO, token.FALLTHROUGH,
		token.CONST, token.VAR, token.TYPE, token.ASSERT:
		return true
	}
	return false
}

// parsePattern parses a match-arm pattern: the catch-all `_` rest pattern, a
// type pattern over a sealed-interface scrutinee (`*Ident`, `*Ident(x)`), or an
// enum variant pattern. A pattern that opens with `*` is a pointer type pattern
// (the §8.1 sealed-interface implementors are pointer types); anything else stays
// an enum variant pattern, so existing enum/Result/Option matches are unaffected.
func (p *parser) parsePattern() ast.Expr {
	if p.at(token.IDENT) && p.cur().Lit == "_" {
		t := p.advance()
		return &ast.RestPattern{Underscore: t.Pos}
	}
	if p.at(token.MUL) {
		return p.parseTypePattern()
	}
	return p.parseVariantPattern()
}

// parseTypePattern parses a type-pattern arm `*T` or `*T(binding)` over a
// sealed-interface scrutinee. The type is a full type expression (a `*T`
// StarExpr); an optional parenthesized identifier binds the narrowed value.
func (p *parser) parseTypePattern() ast.Expr {
	tp := &ast.TypePattern{Type: p.parseType()}
	if p.at(token.LPAREN) {
		lp := p.advance()
		tp.Lparen = lp.Pos
		if p.at(token.IDENT) {
			tp.Binding = p.ident()
		}
		rp := p.expect(token.RPAREN)
		tp.Rparen = rp.Pos
	}
	return tp
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

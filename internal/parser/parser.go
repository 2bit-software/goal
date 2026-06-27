// Package parser turns the lexer's token stream into a goal AST. This file
// implements the declaration tier (REWRITE-ARCHITECTURE §1.4): a hand-written,
// Go-grammar-shaped recursive-descent parser that reads the package clause, the
// import block(s), and the func/type/var/const declarations of the Go subset into
// an *ast.File.
//
// It is deliberately scoped to top-level structure. Function bodies are captured
// as a balanced-brace *ast.BlockStmt with no statement list (statement parsing is
// a later story); declaration initializer values use a minimal operand+postfix
// expression parser (precedence-climbing, unary, and postfix `?` arrive later);
// and the goal-specific declarations (enum/sealed/implements/from/derive) are not
// handled here. Type expressions, however, are parsed fully for the Go subset
// because they are part of a declaration's shape.
//
// The lexer emits no semicolon/newline terminators, so declaration boundaries are
// structural: top-level declarations dispatch on their leading keyword, and
// grouped specs, field lists, and parameter lists run until their closing
// delimiter. COMMENT/DOC_COMMENT trivia is skipped for now; comment attachment is
// the formatter's concern in a later story.
package parser

import (
	"errors"
	"fmt"

	"goal/internal/ast"
	"goal/internal/lexer"
	"goal/internal/token"
)

// parser holds the token stream and cursor for one source file.
type parser struct {
	toks []token.Token // lexed tokens, trivia removed, ending in EOF
	pos  int           // index of the current token
	errs []error       // accumulated parse errors
}

// ParseFile tokenizes src and parses it into an *ast.File. It returns the parse
// errors encountered (joined) or nil for well-formed Go-subset input. A non-nil
// *ast.File is always returned, even on error, so callers can inspect partial
// structure.
func ParseFile(src string) (*ast.File, error) {
	p := newParser(src)
	file := p.parseFile()
	if len(p.errs) > 0 {
		return file, errors.Join(p.errs...)
	}
	return file, nil
}

// newParser lexes src and strips comment/doc-comment trivia, leaving a token
// slice that always ends in EOF.
func newParser(src string) *parser {
	all := lexer.Tokens(src)
	toks := make([]token.Token, 0, len(all))
	for _, t := range all {
		if t.Kind == token.COMMENT || t.Kind == token.DOC_COMMENT {
			continue
		}
		toks = append(toks, t)
	}
	if len(toks) == 0 {
		toks = append(toks, token.Token{Kind: token.EOF})
	}
	return &parser{toks: toks}
}

// ----------------------------------------------------------------------------
// Token cursor

// cur returns the current token.
func (p *parser) cur() token.Token { return p.toks[p.pos] }

// kind returns the current token's kind.
func (p *parser) kind() token.Kind { return p.toks[p.pos].Kind }

// at reports whether the current token has kind k.
func (p *parser) at(k token.Kind) bool { return p.toks[p.pos].Kind == k }

// peekKind returns the kind of the token after the current one (EOF at the end).
func (p *parser) peekKind() token.Kind {
	if p.pos+1 < len(p.toks) {
		return p.toks[p.pos+1].Kind
	}
	return token.EOF
}

// advance consumes the current token and returns it. The cursor never moves past
// the trailing EOF, guaranteeing progress without overrun.
func (p *parser) advance() token.Token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

// expect consumes the current token, recording an error if it is not of kind k.
// It always advances so the parser makes progress on malformed input.
func (p *parser) expect(k token.Kind) token.Token {
	t := p.cur()
	if t.Kind != k {
		p.errorf(t.Pos, "expected %s, found %s", k.String(), describe(t))
	}
	return p.advance()
}

// errorf records a parse error at pos.
func (p *parser) errorf(pos token.Pos, format string, args ...any) {
	p.errs = append(p.errs, fmt.Errorf("%s: %s", pos.String(), fmt.Sprintf(format, args...)))
}

// describe renders a token for error messages.
func describe(t token.Token) string {
	switch {
	case t.Kind == token.EOF:
		return "EOF"
	case t.Lit != "":
		return fmt.Sprintf("%q", t.Lit)
	default:
		return t.Kind.String()
	}
}

// ident consumes an identifier and returns it as an *ast.Ident.
func (p *parser) ident() *ast.Ident {
	t := p.expect(token.IDENT)
	return &ast.Ident{NamePos: t.Pos, Name: t.Lit}
}

// ----------------------------------------------------------------------------
// File and declarations

// parseFile parses a whole source file: the package clause followed by the
// top-level declarations.
func (p *parser) parseFile() *ast.File {
	file := &ast.File{}

	pkg := p.expect(token.PACKAGE)
	file.Package = pkg.Pos
	name := p.expect(token.IDENT)
	file.Name = &ast.Ident{NamePos: name.Pos, Name: name.Lit}

	for !p.at(token.EOF) {
		d := p.parseDecl()
		if d == nil {
			// Unknown construct at top level: report and skip one token so the
			// loop always makes progress.
			p.errorf(p.cur().Pos, "expected declaration, found %s", describe(p.cur()))
			p.advance()
			continue
		}
		file.Decls = append(file.Decls, d)
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			for _, s := range gd.Specs {
				if is, ok := s.(*ast.ImportSpec); ok {
					file.Imports = append(file.Imports, is)
				}
			}
		}
	}
	return file
}

// parseDecl dispatches a top-level declaration by its leading keyword. It returns
// nil when the current token starts no recognized declaration.
func (p *parser) parseDecl() ast.Decl {
	switch p.kind() {
	case token.IMPORT, token.CONST, token.VAR, token.TYPE:
		return p.parseGenDecl(p.kind())
	case token.FUNC:
		return p.parseFuncDecl()
	default:
		return nil
	}
}

// parseGenDecl parses an import/const/var/type declaration in both its single and
// parenthesized-group forms.
func (p *parser) parseGenDecl(tok token.Kind) *ast.GenDecl {
	keyword := p.expect(tok)
	d := &ast.GenDecl{TokPos: keyword.Pos, Tok: tok}
	if p.at(token.LPAREN) {
		p.advance()
		for !p.at(token.RPAREN) && !p.at(token.EOF) {
			d.Specs = append(d.Specs, p.parseSpec(tok))
		}
		p.expect(token.RPAREN)
	} else {
		d.Specs = append(d.Specs, p.parseSpec(tok))
	}
	return d
}

// parseSpec parses a single spec within a GenDecl, chosen by the declaration's
// keyword.
func (p *parser) parseSpec(tok token.Kind) ast.Spec {
	switch tok {
	case token.IMPORT:
		return p.parseImportSpec()
	case token.TYPE:
		return p.parseTypeSpec()
	default: // CONST, VAR
		return p.parseValueSpec()
	}
}

// parseImportSpec parses one import: an optional local name (an identifier, "_",
// or "." for a dot import) followed by the import path string.
func (p *parser) parseImportSpec() *ast.ImportSpec {
	spec := &ast.ImportSpec{}
	switch p.kind() {
	case token.PERIOD:
		t := p.advance()
		spec.Name = &ast.Ident{NamePos: t.Pos, Name: "."}
	case token.IDENT:
		t := p.advance()
		spec.Name = &ast.Ident{NamePos: t.Pos, Name: t.Lit}
	}
	if p.at(token.STRING) {
		t := p.advance()
		spec.Path = &ast.BasicLit{ValuePos: t.Pos, Kind: token.STRING, Value: t.Lit}
	} else {
		p.errorf(p.cur().Pos, "expected import path, found %s", describe(p.cur()))
		p.advance()
	}
	return spec
}

// parseTypeSpec parses a single type declaration: a name, an optional "=" (type
// alias), and the underlying type expression.
func (p *parser) parseTypeSpec() *ast.TypeSpec {
	spec := &ast.TypeSpec{Name: p.ident()}
	if p.at(token.ASSIGN) {
		p.advance() // alias form: type T = U
	}
	spec.Type = p.parseType()
	return spec
}

// parseValueSpec parses a single const/var spec: a name list, an optional type,
// and an optional "= values" list.
func (p *parser) parseValueSpec() *ast.ValueSpec {
	spec := &ast.ValueSpec{Names: p.parseIdentList()}
	if !p.at(token.ASSIGN) && p.startsType() {
		spec.Type = p.parseType()
	}
	if p.at(token.ASSIGN) {
		p.advance()
		spec.Values = p.parseExprList()
	}
	return spec
}

// parseFuncDecl parses a function or method declaration. A leading "(" after the
// func keyword introduces a receiver. The body, when present, is captured as a
// balanced-brace block without parsing its statements.
func (p *parser) parseFuncDecl() *ast.FuncDecl {
	keyword := p.expect(token.FUNC)
	fd := &ast.FuncDecl{}
	ft := &ast.FuncType{Func: keyword.Pos}
	if p.at(token.LPAREN) {
		fd.Recv = p.parseParamList()
	}
	fd.Name = p.ident()
	ft.Params = p.parseParamList()
	ft.Results = p.parseResults()
	fd.Type = ft
	if p.at(token.LBRACE) {
		fd.Body = p.parseBlockSkip()
	}
	return fd
}

// ----------------------------------------------------------------------------
// Identifier and expression lists

// parseIdentList parses one or more comma-separated identifiers.
func (p *parser) parseIdentList() []*ast.Ident {
	list := []*ast.Ident{p.ident()}
	for p.at(token.COMMA) {
		p.advance()
		list = append(list, p.ident())
	}
	return list
}

// parseExprList parses one or more comma-separated expressions.
func (p *parser) parseExprList() []ast.Expr {
	list := []ast.Expr{p.parseExpr()}
	for p.at(token.COMMA) {
		p.advance()
		list = append(list, p.parseExpr())
	}
	return list
}

// ----------------------------------------------------------------------------
// Type expressions

// startsType reports whether the current token can begin a type expression.
func (p *parser) startsType() bool { return startsTypeKind(p.kind()) }

func startsTypeKind(k token.Kind) bool {
	switch k {
	case token.IDENT, token.MUL, token.LBRACK, token.MAP, token.STRUCT,
		token.INTERFACE, token.FUNC, token.CHAN, token.ARROW, token.LPAREN,
		token.ELLIPSIS:
		return true
	}
	return false
}

// parseType parses a Go-subset type expression.
func (p *parser) parseType() ast.Expr {
	switch p.kind() {
	case token.IDENT:
		return p.parseTypeName()
	case token.MUL:
		star := p.advance()
		return &ast.StarExpr{Star: star.Pos, X: p.parseType()}
	case token.LBRACK:
		return p.parseArrayOrSliceType()
	case token.MAP:
		return p.parseMapType()
	case token.STRUCT:
		return p.parseStructType()
	case token.INTERFACE:
		return p.parseInterfaceType()
	case token.FUNC:
		kw := p.advance()
		ft := p.parseSignature()
		ft.Func = kw.Pos
		return ft
	case token.CHAN, token.ARROW:
		return p.parseChanType()
	case token.LPAREN:
		p.advance()
		t := p.parseType()
		p.expect(token.RPAREN)
		return t
	case token.ELLIPSIS:
		e := p.advance()
		return &ast.Ellipsis{Ellipsis: e.Pos, Elt: p.parseType()}
	default:
		p.errorf(p.cur().Pos, "expected type, found %s", describe(p.cur()))
		t := p.advance()
		return &ast.Ident{NamePos: t.Pos, Name: t.Lit}
	}
}

// parseTypeName parses an identifier type, possibly qualified (pkg.Name) and/or
// instantiated with a single index (Name[T]).
func (p *parser) parseTypeName() ast.Expr {
	return p.typeNameFrom(p.ident())
}

// typeNameFrom extends an already-consumed identifier into a qualified and/or
// indexed type name.
func (p *parser) typeNameFrom(id *ast.Ident) ast.Expr {
	var x ast.Expr = id
	if p.at(token.PERIOD) {
		p.advance()
		x = &ast.SelectorExpr{X: id, Sel: p.ident()}
	}
	if p.at(token.LBRACK) {
		x = p.parseIndexSuffix(x)
	}
	return x
}

// parseArrayOrSliceType parses [N]T, []T, or [...]T.
func (p *parser) parseArrayOrSliceType() ast.Expr {
	lb := p.expect(token.LBRACK)
	arr := &ast.ArrayType{Lbrack: lb.Pos}
	if p.at(token.ELLIPSIS) {
		e := p.advance()
		arr.Len = &ast.Ellipsis{Ellipsis: e.Pos}
	} else if !p.at(token.RBRACK) {
		arr.Len = p.parseExpr()
	}
	p.expect(token.RBRACK)
	arr.Elt = p.parseType()
	return arr
}

// parseMapType parses map[K]V.
func (p *parser) parseMapType() ast.Expr {
	m := p.expect(token.MAP)
	p.expect(token.LBRACK)
	key := p.parseType()
	p.expect(token.RBRACK)
	return &ast.MapType{Map: m.Pos, Key: key, Value: p.parseType()}
}

// parseChanType parses chan T, chan<- T, and <-chan T.
func (p *parser) parseChanType() ast.Expr {
	c := &ast.ChanType{Begin: p.cur().Pos, Dir: ast.SendRecv}
	if p.at(token.ARROW) {
		p.advance()
		p.expect(token.CHAN)
		c.Dir = ast.RecvOnly
	} else {
		p.expect(token.CHAN)
		if p.at(token.ARROW) {
			p.advance()
			c.Dir = ast.SendOnly
		}
	}
	c.Value = p.parseType()
	return c
}

// parseStructType parses struct{ ... } with its field list. (The goal
// `struct implements I` clause is a later story.)
func (p *parser) parseStructType() ast.Expr {
	kw := p.expect(token.STRUCT)
	st := &ast.StructType{Struct: kw.Pos}
	fl := &ast.FieldList{}
	lb := p.expect(token.LBRACE)
	fl.Opening = lb.Pos
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		fl.List = append(fl.List, p.parseField())
	}
	rb := p.expect(token.RBRACE)
	fl.Closing = rb.Pos
	st.Fields = fl
	return st
}

// parseField parses one struct field: a name list followed by a type, or an
// embedded type. An optional trailing string literal is the field tag.
func (p *parser) parseField() *ast.Field {
	f := &ast.Field{}
	if p.at(token.IDENT) {
		names := []*ast.Ident{p.ident()}
		for p.at(token.COMMA) {
			p.advance()
			names = append(names, p.ident())
		}
		if p.startsType() {
			f.Names = names
			f.Type = p.parseType()
		} else if len(names) == 1 {
			// Embedded field: the lone identifier is (the start of) a type name.
			f.Type = p.typeNameFrom(names[0])
		} else {
			f.Names = names
			p.errorf(p.cur().Pos, "expected field type, found %s", describe(p.cur()))
		}
	} else {
		f.Type = p.parseType()
	}
	if p.at(token.STRING) {
		t := p.advance()
		f.Tag = &ast.BasicLit{ValuePos: t.Pos, Kind: token.STRING, Value: t.Lit}
	}
	return f
}

// parseInterfaceType parses interface{ ... } with its method/embedding list.
func (p *parser) parseInterfaceType() ast.Expr {
	kw := p.expect(token.INTERFACE)
	it := &ast.InterfaceType{Interface: kw.Pos}
	ml := &ast.FieldList{}
	lb := p.expect(token.LBRACE)
	ml.Opening = lb.Pos
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		ml.List = append(ml.List, p.parseMethodSpec())
	}
	rb := p.expect(token.RBRACE)
	ml.Closing = rb.Pos
	it.Methods = ml
	return it
}

// parseMethodSpec parses one interface element: a method (name + signature) or an
// embedded interface name.
func (p *parser) parseMethodSpec() *ast.Field {
	f := &ast.Field{}
	name := p.ident()
	if p.at(token.LPAREN) {
		f.Names = []*ast.Ident{name}
		f.Type = p.parseSignature()
	} else {
		f.Type = p.typeNameFrom(name)
	}
	return f
}

// parseSignature parses a parameter list and optional results into a FuncType
// (without the leading func keyword position).
func (p *parser) parseSignature() *ast.FuncType {
	ft := &ast.FuncType{}
	ft.Params = p.parseParamList()
	ft.Results = p.parseResults()
	return ft
}

// parseParamList parses a parenthesized, comma-separated parameter (or receiver)
// list.
func (p *parser) parseParamList() *ast.FieldList {
	fl := &ast.FieldList{}
	lp := p.expect(token.LPAREN)
	fl.Opening = lp.Pos
	for !p.at(token.RPAREN) && !p.at(token.EOF) {
		fl.List = append(fl.List, p.parseParam())
		if p.at(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	rp := p.expect(token.RPAREN)
	fl.Closing = rp.Pos
	return fl
}

// parseParam parses one parameter: "name type", a bare type, or a variadic
// "...type". Whether a leading identifier is a name or an unnamed type is decided
// by one-token lookahead.
func (p *parser) parseParam() *ast.Field {
	f := &ast.Field{}
	if p.nameThenType() {
		f.Names = []*ast.Ident{p.ident()}
		f.Type = p.parseTypeOrVariadic()
	} else {
		f.Type = p.parseTypeOrVariadic()
	}
	return f
}

// nameThenType reports whether the current identifier is a parameter name
// followed by a type (rather than itself being an unnamed parameter type).
func (p *parser) nameThenType() bool {
	if !p.at(token.IDENT) {
		return false
	}
	switch p.peekKind() {
	case token.COMMA, token.RPAREN, token.PERIOD:
		return false // unnamed type, qualified type, or end of list
	}
	return startsTypeKind(p.peekKind())
}

// parseTypeOrVariadic parses a type or a variadic "...type" element.
func (p *parser) parseTypeOrVariadic() ast.Expr {
	if p.at(token.ELLIPSIS) {
		e := p.advance()
		return &ast.Ellipsis{Ellipsis: e.Pos, Elt: p.parseType()}
	}
	return p.parseType()
}

// parseResults parses a function's results: a parenthesized list, a single
// unnamed type, or nothing.
func (p *parser) parseResults() *ast.FieldList {
	if p.at(token.LPAREN) {
		return p.parseParamList()
	}
	if p.startsType() {
		return &ast.FieldList{List: []*ast.Field{{Type: p.parseType()}}}
	}
	return nil
}

// parseBlockSkip consumes a balanced-brace block, recording its brace positions
// but not parsing its statements (statement parsing is a later story).
func (p *parser) parseBlockSkip() *ast.BlockStmt {
	lb := p.expect(token.LBRACE)
	b := &ast.BlockStmt{Lbrace: lb.Pos}
	depth := 1
	for !p.at(token.EOF) {
		switch p.kind() {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			depth--
			if depth == 0 {
				b.Rbrace = p.cur().Pos
				p.advance()
				return b
			}
		}
		p.advance()
	}
	p.errorf(p.cur().Pos, "unterminated block")
	b.Rbrace = p.cur().Pos
	return b
}

// ----------------------------------------------------------------------------
// Minimal expressions (operand + postfix only)
//
// Declaration initializer values need just enough expression grammar to capture
// literals, names, calls, indexing, selectors, and composite literals. Full
// precedence-climbing, unary operators, and postfix `?` are a later story; this
// parser intentionally stops at the first token it cannot extend.

// parseExpr parses an operand followed by any postfix chain.
func (p *parser) parseExpr() ast.Expr {
	return p.parsePostfix(p.parseOperand())
}

// parseOperand parses a primary operand: an identifier, a basic literal, or a
// parenthesized expression.
func (p *parser) parseOperand() ast.Expr {
	t := p.cur()
	switch t.Kind {
	case token.IDENT:
		p.advance()
		return &ast.Ident{NamePos: t.Pos, Name: t.Lit}
	case token.INT, token.FLOAT, token.IMAG, token.CHAR, token.STRING:
		p.advance()
		return &ast.BasicLit{ValuePos: t.Pos, Kind: t.Kind, Value: t.Lit}
	case token.LPAREN:
		p.advance()
		inner := p.parseExpr()
		rp := p.expect(token.RPAREN)
		return &ast.ParenExpr{Lparen: t.Pos, X: inner, Rparen: rp.Pos}
	default:
		p.errorf(t.Pos, "expected expression, found %s", describe(t))
		p.advance()
		return &ast.Ident{NamePos: t.Pos, Name: t.Lit}
	}
}

// parsePostfix applies any chain of selector, call, index, and composite-literal
// suffixes to x.
func (p *parser) parsePostfix(x ast.Expr) ast.Expr {
	for {
		switch p.kind() {
		case token.PERIOD:
			p.advance()
			x = &ast.SelectorExpr{X: x, Sel: p.ident()}
		case token.LPAREN:
			x = p.parseCallSuffix(x)
		case token.LBRACK:
			x = p.parseIndexSuffix(x)
		case token.LBRACE:
			if !compositeOK(x) {
				return x
			}
			x = p.parseCompositeLit(x)
		default:
			return x
		}
	}
}

// compositeOK reports whether x is a type-name form that may precede a composite
// literal body.
func compositeOK(x ast.Expr) bool {
	switch x.(type) {
	case *ast.Ident, *ast.SelectorExpr, *ast.IndexExpr:
		return true
	}
	return false
}

// parseCallSuffix parses a call argument list applied to fun.
func (p *parser) parseCallSuffix(fun ast.Expr) ast.Expr {
	lp := p.expect(token.LPAREN)
	call := &ast.CallExpr{Fun: fun, Lparen: lp.Pos}
	for !p.at(token.RPAREN) && !p.at(token.EOF) {
		call.Args = append(call.Args, p.parseExpr())
		if p.at(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	rp := p.expect(token.RPAREN)
	call.Rparen = rp.Pos
	return call
}

// parseIndexSuffix parses a single index applied to x.
func (p *parser) parseIndexSuffix(x ast.Expr) ast.Expr {
	lb := p.expect(token.LBRACK)
	idx := p.parseExpr()
	rb := p.expect(token.RBRACK)
	return &ast.IndexExpr{X: x, Lbrack: lb.Pos, Index: idx, Rbrack: rb.Pos}
}

// parseCompositeLit parses a composite-literal body { ... } for the given (or
// elided, when nil) type.
func (p *parser) parseCompositeLit(typ ast.Expr) ast.Expr {
	lb := p.expect(token.LBRACE)
	cl := &ast.CompositeLit{Type: typ, Lbrace: lb.Pos}
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		cl.Elts = append(cl.Elts, p.parseElement())
		if p.at(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	rb := p.expect(token.RBRACE)
	cl.Rbrace = rb.Pos
	return cl
}

// parseElement parses one composite-literal element: a value or a key:value pair.
func (p *parser) parseElement() ast.Expr {
	x := p.parseElementValue()
	if p.at(token.COLON) {
		colon := p.advance()
		return &ast.KeyValueExpr{Key: x, Colon: colon.Pos, Value: p.parseElementValue()}
	}
	return x
}

// parseElementValue parses a composite-literal key or value, allowing a nested
// brace-elided composite literal.
func (p *parser) parseElementValue() ast.Expr {
	if p.at(token.LBRACE) {
		return p.parseCompositeLit(nil)
	}
	return p.parseExpr()
}

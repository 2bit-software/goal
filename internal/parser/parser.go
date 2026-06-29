// Package parser turns the lexer's token stream into a goal AST. This file
// implements the declaration tier (REWRITE-ARCHITECTURE §1.4): a hand-written,
// Go-grammar-shaped recursive-descent parser that reads the package clause, the
// import block(s), and the func/type/var/const declarations of the Go subset into
// an *ast.File.
//
// Function bodies are parsed into a real statement list (US-018): assignment and
// short-var declarations, if/for/switch (incl. range and three-clause for),
// return/defer/go/break/continue, nested blocks, and const/var/type declared
// inside a body. Expressions parse with full Go operator precedence and left
// associativity (precedence climbing), prefix/unary operators, and goal's
// postfix `?` unwrap as ast.UnwrapExpr (US-019); the goal-specific declarations
// (enum/sealed/implements/from/derive) are not handled here. Type expressions,
// however, are parsed fully for the Go subset because they are part of a
// declaration's shape.
//
// Because the lexer inserts no semicolons, a statement list is delimited
// structurally: each statement parser consumes exactly its tokens and stops, and
// the block loop runs until the closing brace. An explicit ';' (as written in a
// three-clause for header) is still lexed and consumed where the grammar expects
// it. Control-clause headers parse their expressions with composite-literal
// braces suppressed (exprLev < 0) so a trailing body '{' is taken as the block,
// not as a composite literal.
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
	toks    []token.Token // lexed tokens, trivia removed, ending in EOF
	pos     int           // index of the current token
	errs    []error       // accumulated parse errors
	exprLev int           // <0 while parsing a control-clause header (suppresses composite-literal braces)
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

// newParser lexes src and strips ordinary `//` comment trivia, but KEEPS
// `///` DOC_COMMENT tokens in the stream so doctests can be collected and
// attached to the following declaration (US-023). The slice always ends in EOF.
func newParser(src string) *parser {
	all := lexer.Tokens(src)
	toks := make([]token.Token, 0, len(all))
	for _, t := range all {
		if t.Kind == token.COMMENT {
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

// kindAt returns the kind of the token n positions ahead of the current one
// (kindAt(1) == peekKind), or EOF past the end.
func (p *parser) kindAt(n int) token.Kind {
	if p.pos+n < len(p.toks) {
		return p.toks[p.pos+n].Kind
	}
	return token.EOF
}

// onNewLine reports whether the current token begins on a later source line than
// the previously consumed token. The lexer drops newlines/implicit semicolons,
// so this reconstructs the line boundary used to terminate a grouped const/var
// spec. False at the start of the stream (no previous token).
func (p *parser) onNewLine() bool {
	if p.pos == 0 {
		return false
	}
	return p.cur().Pos.Line > p.toks[p.pos-1].Pos.Line
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
		// A `///` doc-comment run preceding a declaration is collected and
		// attached to the following function (US-023).
		doc := p.collectDoc()
		if p.at(token.EOF) {
			break
		}
		d := p.parseDecl()
		if d == nil {
			// Unknown construct at top level: report and skip one token so the
			// loop always makes progress.
			p.errorf(p.cur().Pos, "expected declaration, found %s", describe(p.cur()))
			p.advance()
			continue
		}
		if fd, ok := d.(*ast.FuncDecl); ok && doc != nil {
			fd.Doc = doc
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
	case token.ENUM:
		return p.parseEnumDecl()
	default:
		// `sealed` is a contextual keyword (lexed as IDENT) introducing a sealed
		// interface declaration.
		if p.isContextual("sealed") {
			return p.parseSealedInterfaceDecl()
		}
		// `from func` / `derive func`: the contextual modifier (lexed as IDENT)
		// precedes an ordinary func declaration (US-023). Recorded as a modifier
		// on the FuncDecl, not as a token kind.
		if p.isContextual("from") && p.peekKind() == token.FUNC {
			return p.parseModFuncDecl(ast.FuncFrom)
		}
		if p.isContextual("derive") && p.peekKind() == token.FUNC {
			return p.parseModFuncDecl(ast.FuncDerive)
		}
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

// parseTypeSpec parses a single type declaration: a name, an optional generic
// type-parameter list, an optional "=" (type alias), and the underlying type
// expression.
func (p *parser) parseTypeSpec() *ast.TypeSpec {
	spec := &ast.TypeSpec{Name: p.ident()}
	if p.atTypeParams() {
		spec.TypeParams = p.parseTypeParams()
	}
	if p.at(token.ASSIGN) {
		spec.Assign = p.advance().Pos // alias form: type T = U
	}
	spec.Type = p.parseType()
	return spec
}

// atTypeParams reports whether the current "[" begins a generic type-parameter
// list (e.g. `[T any]`) rather than an array/slice type (`[]T`, `[N]T`, `[3]T`).
// A type-parameter list always opens with `ident` followed by a constraint type
// or a comma; an array length is either absent (`[]`), a non-identifier, or an
// identifier immediately followed by `]`.
func (p *parser) atTypeParams() bool {
	if !p.at(token.LBRACK) || p.peekKind() != token.IDENT {
		return false
	}
	k2 := p.kindAt(2) // token after the first identifier inside "["
	return k2 == token.COMMA || startsTypeKind(k2)
}

// parseTypeParams parses a generic type-parameter list "[ name... constraint,
// ... ]". Names sharing a constraint (`[T, U any]`) collect onto one Field; a
// new constraint after a comma (`[T any, U int]`) starts a new Field.
func (p *parser) parseTypeParams() *ast.FieldList {
	fl := &ast.FieldList{}
	lb := p.expect(token.LBRACK)
	fl.Opening = lb.Pos
	for !p.at(token.RBRACK) && !p.at(token.EOF) {
		f := &ast.Field{Names: []*ast.Ident{p.ident()}}
		for p.at(token.COMMA) && p.peekKind() == token.IDENT {
			p.advance() // consume the comma separating shared names
			f.Names = append(f.Names, p.ident())
		}
		f.Type = p.parseType() // the shared constraint
		fl.List = append(fl.List, f)
		if p.at(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	rb := p.expect(token.RBRACK)
	fl.Closing = rb.Pos
	return fl
}

// parseValueSpec parses a single const/var spec: a name list, an optional type,
// and an optional "= values" list.
func (p *parser) parseValueSpec() *ast.ValueSpec {
	spec := &ast.ValueSpec{Names: p.parseIdentList()}
	// The lexer drops newlines/implicit semicolons, so within a grouped
	// const/var a spec boundary is reconstructed from source lines: an optional
	// type belongs to this spec only when it begins on the same line as the
	// names. A following identifier on the next line is a new bare spec (e.g. an
	// iota-continuation const), not this spec's type.
	if !p.at(token.ASSIGN) && p.startsType() && !p.onNewLine() {
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
	ft.Results = p.parseResults(ft.Params.Closing)
	fd.Type = ft
	if p.at(token.LBRACE) {
		fd.Body = p.parseBlock()
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

// parseStructType parses struct{ ... } with its field list, including the goal
// `struct implements I { ... }` clause (consumed before the field-list brace;
// absent for an ordinary inline struct type).
func (p *parser) parseStructType() ast.Expr {
	kw := p.expect(token.STRUCT)
	st := &ast.StructType{Struct: kw.Pos}
	st.Implements = p.parseImplementsClause()
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
	return &ast.InterfaceType{Interface: kw.Pos, Methods: p.parseInterfaceBody()}
}

// parseInterfaceBody parses a braced interface method/embedding list { ... }
// into a FieldList. It is shared by an ordinary interface type and a sealed
// interface declaration.
func (p *parser) parseInterfaceBody() *ast.FieldList {
	ml := &ast.FieldList{}
	lb := p.expect(token.LBRACE)
	ml.Opening = lb.Pos
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		ml.List = append(ml.List, p.parseMethodSpec())
	}
	rb := p.expect(token.RBRACE)
	ml.Closing = rb.Pos
	return ml
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
	ft.Results = p.parseResults(ft.Params.Closing)
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
// unnamed type, or nothing. A result must begin on the same source line as
// paramsEnd (the closing ')' of the parameter list): Go inserts a semicolon
// after that ')' when a newline follows, ending the signature, so a type on a
// later line is the next construct — e.g. the next interface method — not a
// result. goal's lexer strips newlines, so we reconstruct that boundary from the
// line directly.
func (p *parser) parseResults(paramsEnd token.Pos) *ast.FieldList {
	if p.cur().Pos.Line != paramsEnd.Line {
		return nil
	}
	if p.at(token.LPAREN) {
		return p.parseParamList()
	}
	if p.startsType() {
		return &ast.FieldList{List: []*ast.Field{{Type: p.parseType()}}}
	}
	return nil
}

// ----------------------------------------------------------------------------
// Statements
//
// The lexer emits no semicolon/newline terminators, so a statement list is
// delimited structurally: parseBlock loops until the closing brace, and each
// statement parser consumes exactly its own tokens and stops. An explicit ';'
// (written in a three-clause for header) is lexed as token.SEMICOLON and consumed
// where the grammar expects it; a stray one is an empty statement.

// parseBlock parses a braced statement list { ... }, filling BlockStmt.List.
func (p *parser) parseBlock() *ast.BlockStmt {
	lb := p.expect(token.LBRACE)
	b := &ast.BlockStmt{Lbrace: lb.Pos}
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		// A stray `///` doc comment inside a body is not attached to anything;
		// skip it so statement parsing is unaffected (US-023).
		if p.at(token.DOC_COMMENT) {
			p.advance()
			continue
		}
		b.List = append(b.List, p.parseStmt())
	}
	rb := p.expect(token.RBRACE)
	b.Rbrace = rb.Pos
	return b
}

// parseStmt parses a single statement, dispatching on the leading token.
func (p *parser) parseStmt() ast.Stmt {
	// A "label:" prefix at statement position is a labeled statement (the target
	// of goto/break/continue), distinct from an expression starting with an ident.
	if p.at(token.IDENT) && p.peekKind() == token.COLON {
		return p.parseLabeledStmt()
	}
	switch p.kind() {
	case token.LBRACE:
		return p.parseBlock()
	case token.IF:
		return p.parseIfStmt()
	case token.FOR:
		return p.parseForStmt()
	case token.SWITCH:
		return p.parseSwitchStmt()
	case token.SELECT:
		return p.parseSelectStmt()
	case token.RETURN:
		return p.parseReturnStmt()
	case token.DEFER:
		return p.parseCallStmt(token.DEFER)
	case token.GO:
		return p.parseCallStmt(token.GO)
	case token.BREAK, token.CONTINUE, token.GOTO, token.FALLTHROUGH:
		return p.parseBranchStmt()
	case token.CONST, token.VAR, token.TYPE:
		return &ast.DeclStmt{Decl: p.parseGenDecl(p.kind())}
	case token.SEMICOLON:
		t := p.advance()
		return &ast.EmptyStmt{Semicolon: t.Pos}
	case token.MATCH:
		// Statement-position match: the same MatchExpr node, wrapped as a
		// statement (the AST has no separate MatchStmt).
		return &ast.ExprStmt{X: p.parseMatchExpr()}
	case token.ASSERT:
		return p.parseAssertStmt()
	default:
		return p.parseSimpleStmt(false)
	}
}

// parseSimpleStmt parses a simple statement: an expression statement, an
// assignment/short-var declaration, or an increment/decrement. When allowRange is
// set (only in a for header), a `range x` operand after the assignment token
// yields a *ast.RangeStmt instead.
func (p *parser) parseSimpleStmt(allowRange bool) ast.Stmt {
	lhs := p.parseExprList()
	switch p.kind() {
	case token.ASSIGN, token.DEFINE,
		token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN,
		token.REM_ASSIGN, token.AND_ASSIGN, token.OR_ASSIGN, token.XOR_ASSIGN,
		token.SHL_ASSIGN, token.SHR_ASSIGN, token.AND_NOT_ASSIGN:
		tok := p.advance()
		if allowRange && p.at(token.RANGE) && (tok.Kind == token.ASSIGN || tok.Kind == token.DEFINE) {
			return p.parseRangeRest(lhs, tok)
		}
		rhs := p.parseExprList()
		return &ast.AssignStmt{Lhs: lhs, TokPos: tok.Pos, Tok: tok.Kind, Rhs: rhs}
	case token.ARROW:
		arrow := p.advance()
		return &ast.SendStmt{Chan: lhs[0], Arrow: arrow.Pos, Value: p.parseExpr()}
	case token.INC, token.DEC:
		tok := p.advance()
		return &ast.IncDecStmt{X: lhs[0], TokPos: tok.Pos, Tok: tok.Kind}
	default:
		if len(lhs) > 1 {
			p.errorf(p.cur().Pos, "expected := or = (multiple expressions in statement), found %s", describe(p.cur()))
		}
		return &ast.ExprStmt{X: lhs[0]}
	}
}

// parseRangeRest builds a RangeStmt from an already-parsed key/value list and the
// assignment token, with the cursor on the `range` keyword.
func (p *parser) parseRangeRest(lhs []ast.Expr, tok token.Token) *ast.RangeStmt {
	p.expect(token.RANGE)
	rs := &ast.RangeStmt{TokPos: tok.Pos, Tok: tok.Kind, X: p.parseExpr()}
	if len(lhs) > 0 {
		rs.Key = lhs[0]
	}
	if len(lhs) > 1 {
		rs.Value = lhs[1]
	}
	return rs
}

// parseIfStmt parses an if statement with an optional init statement and else
// branch.
func (p *parser) parseIfStmt() ast.Stmt {
	ifPos := p.expect(token.IF).Pos
	s := &ast.IfStmt{If: ifPos}

	prev := p.exprLev
	p.exprLev = -1
	s1 := p.parseSimpleStmt(false)
	if p.at(token.SEMICOLON) {
		p.advance()
		s.Init = s1
		s.Cond = p.parseExpr()
	} else {
		s.Cond = stmtExpr(s1)
	}
	p.exprLev = prev

	s.Body = p.parseBlock()
	if p.at(token.ELSE) {
		p.advance()
		if p.at(token.IF) {
			s.Else = p.parseIfStmt()
		} else {
			s.Else = p.parseBlock()
		}
	}
	return s
}

// parseForStmt parses a for statement in its infinite, condition-only,
// three-clause, and range forms.
func (p *parser) parseForStmt() ast.Stmt {
	forPos := p.expect(token.FOR).Pos

	// for { ... }
	if p.at(token.LBRACE) {
		return &ast.ForStmt{For: forPos, Body: p.parseBlock()}
	}

	prev := p.exprLev
	p.exprLev = -1

	// for range x { ... }
	if p.at(token.RANGE) {
		p.advance()
		x := p.parseExpr()
		p.exprLev = prev
		return &ast.RangeStmt{For: forPos, X: x, Body: p.parseBlock()}
	}

	var s1 ast.Stmt
	if !p.at(token.SEMICOLON) {
		s1 = p.parseSimpleStmt(true)
		if rs, ok := s1.(*ast.RangeStmt); ok {
			rs.For = forPos
			p.exprLev = prev
			rs.Body = p.parseBlock()
			return rs
		}
	}

	// Three-clause for: init ; cond ; post
	if p.at(token.SEMICOLON) {
		p.advance()
		s := &ast.ForStmt{For: forPos, Init: s1}
		if !p.at(token.SEMICOLON) {
			s.Cond = p.parseExpr()
		}
		p.expect(token.SEMICOLON)
		if !p.at(token.LBRACE) {
			s.Post = p.parseSimpleStmt(false)
		}
		p.exprLev = prev
		s.Body = p.parseBlock()
		return s
	}

	// Condition-only for: s1 carries the condition.
	s := &ast.ForStmt{For: forPos, Cond: stmtExpr(s1)}
	p.exprLev = prev
	s.Body = p.parseBlock()
	return s
}

// parseSwitchStmt parses an expression switch with an optional init and tag.
func (p *parser) parseSwitchStmt() ast.Stmt {
	swPos := p.expect(token.SWITCH).Pos

	// Parse an optional init statement and an optional tag/guard. With a
	// semicolon, the first simple statement is the init and the second is the
	// tag/guard; without one, the lone simple statement is the tag/guard.
	prev := p.exprLev
	p.exprLev = -1
	var init, guard ast.Stmt
	if !p.at(token.LBRACE) && !p.at(token.SEMICOLON) {
		guard = p.parseSimpleStmt(false)
	}
	if p.at(token.SEMICOLON) {
		p.advance()
		init = guard
		guard = nil
		if !p.at(token.LBRACE) {
			guard = p.parseSimpleStmt(false)
		}
	}
	p.exprLev = prev

	// A guard of the form `x.(type)` or `v := x.(type)` makes this a type switch.
	if isTypeSwitchGuard(guard) {
		ts := &ast.TypeSwitchStmt{Switch: swPos, Init: init, Assign: guard}
		ts.Body = p.parseCaseBody()
		return ts
	}
	return &ast.SwitchStmt{Switch: swPos, Init: init, Tag: stmtExpr(guard), Body: p.parseCaseBody()}
}

// isTypeSwitchGuard reports whether s is a type-switch guard: a bare `x.(type)`
// assertion or a `v := x.(type)` binding (a TypeAssertExpr with no asserted type).
func isTypeSwitchGuard(s ast.Stmt) bool {
	switch g := s.(type) {
	case *ast.ExprStmt:
		ta, ok := g.X.(*ast.TypeAssertExpr)
		return ok && ta.Type == nil
	case *ast.AssignStmt:
		if g.Tok != token.DEFINE || len(g.Rhs) != 1 {
			return false
		}
		ta, ok := g.Rhs[0].(*ast.TypeAssertExpr)
		return ok && ta.Type == nil
	}
	return false
}

// parseCaseBody parses the braced "{ case … }" body shared by ordinary and type
// switches, with the cursor on the opening brace.
func (p *parser) parseCaseBody() *ast.BlockStmt {
	lb := p.expect(token.LBRACE)
	body := &ast.BlockStmt{Lbrace: lb.Pos}
	for p.at(token.CASE) || p.at(token.DEFAULT) {
		body.List = append(body.List, p.parseCaseClause())
	}
	body.Rbrace = p.expect(token.RBRACE).Pos
	return body
}

// parseSelectStmt parses a select statement: "select { case <comm>: … }". Each
// comm clause guards on a channel send or receive, or is the default clause.
func (p *parser) parseSelectStmt() ast.Stmt {
	pos := p.expect(token.SELECT).Pos
	lb := p.expect(token.LBRACE)
	body := &ast.BlockStmt{Lbrace: lb.Pos}
	for p.at(token.CASE) || p.at(token.DEFAULT) {
		body.List = append(body.List, p.parseCommClause())
	}
	body.Rbrace = p.expect(token.RBRACE).Pos
	return &ast.SelectStmt{Select: pos, Body: body}
}

// parseCommClause parses one clause of a select. A "case" guard is a send or
// receive simple statement (`ch <- v`, `v := <-ch`, `<-ch`); "default" has none.
func (p *parser) parseCommClause() ast.Stmt {
	cc := &ast.CommClause{Case: p.cur().Pos}
	if p.at(token.CASE) {
		p.advance()
		cc.Comm = p.parseSimpleStmt(false)
	} else {
		p.expect(token.DEFAULT)
	}
	cc.Colon = p.expect(token.COLON).Pos
	for !p.at(token.CASE) && !p.at(token.DEFAULT) && !p.at(token.RBRACE) && !p.at(token.EOF) {
		if p.at(token.SEMICOLON) {
			p.advance()
			continue
		}
		cc.Body = append(cc.Body, p.parseStmt())
	}
	return cc
}

// parseLabeledStmt parses "label: stmt", with the cursor on the label ident. A
// label that is the last thing in a block (no following statement) gets an empty
// statement so a goto target can sit at a block's end.
func (p *parser) parseLabeledStmt() ast.Stmt {
	label := p.ident()
	colon := p.expect(token.COLON)
	ls := &ast.LabeledStmt{Label: label, Colon: colon.Pos}
	if p.at(token.RBRACE) || p.at(token.EOF) {
		ls.Stmt = &ast.EmptyStmt{Semicolon: colon.Pos}
	} else {
		ls.Stmt = p.parseStmt()
	}
	return ls
}

// parseCaseClause parses one case or default clause within a switch body.
func (p *parser) parseCaseClause() ast.Stmt {
	cc := &ast.CaseClause{Case: p.cur().Pos}
	if p.at(token.CASE) {
		p.advance()
		cc.List = p.parseExprList()
	} else {
		p.expect(token.DEFAULT)
	}
	cc.Colon = p.expect(token.COLON).Pos
	for !p.at(token.CASE) && !p.at(token.DEFAULT) && !p.at(token.RBRACE) && !p.at(token.EOF) {
		if p.at(token.SEMICOLON) {
			p.advance()
			continue
		}
		cc.Body = append(cc.Body, p.parseStmt())
	}
	return cc
}

// parseReturnStmt parses a return statement with an optional result list.
func (p *parser) parseReturnStmt() ast.Stmt {
	pos := p.expect(token.RETURN).Pos
	r := &ast.ReturnStmt{Return: pos}
	if startsExpr(p.kind()) {
		r.Results = p.parseExprList()
	}
	return r
}

// parseCallStmt parses a defer or go statement, whose operand must be a call.
func (p *parser) parseCallStmt(tok token.Kind) ast.Stmt {
	pos := p.expect(tok).Pos
	x := p.parseExpr()
	call, ok := x.(*ast.CallExpr)
	if !ok {
		p.errorf(pos, "expected function call after %s", tok.String())
	}
	if tok == token.DEFER {
		return &ast.DeferStmt{Defer: pos, Call: call}
	}
	return &ast.GoStmt{Go: pos, Call: call}
}

// parseBranchStmt parses break/continue/goto/fallthrough, with an optional label.
func (p *parser) parseBranchStmt() ast.Stmt {
	t := p.advance()
	b := &ast.BranchStmt{TokPos: t.Pos, Tok: t.Kind}
	if t.Kind != token.FALLTHROUGH && p.at(token.IDENT) {
		b.Label = p.ident()
	}
	return b
}

// stmtExpr extracts the expression from a simple statement that turned out to be a
// bare condition/tag (an *ast.ExprStmt); it returns nil otherwise.
func stmtExpr(s ast.Stmt) ast.Expr {
	if es, ok := s.(*ast.ExprStmt); ok {
		return es.X
	}
	return nil
}

// startsExpr reports whether k can begin an expression — an operand or a
// prefix/unary operator (+ - ! ^ & <- and the pointer-deref *).
func startsExpr(k token.Kind) bool {
	switch k {
	case token.IDENT, token.INT, token.FLOAT, token.IMAG, token.CHAR, token.STRING, token.LPAREN,
		token.ADD, token.SUB, token.NOT, token.XOR, token.AND, token.ARROW, token.MUL,
		token.MATCH,
		// Type-literal operand starts that parseOperand accepts as the head of a
		// composite literal, conversion, or func literal — e.g. `return []T{…}`,
		// `return map[K]V{…}`, `return struct{…}{…}`, `return func() {…}`.
		token.LBRACK, token.MAP, token.STRUCT, token.CHAN, token.INTERFACE, token.FUNC:
		return true
	}
	return false
}

// ----------------------------------------------------------------------------
// Expressions (precedence climbing + unary + postfix)
//
// Expressions parse with Go's operator precedence and left associativity via
// precedence climbing (parseBinary), prefix/unary operators (parseUnary), and a
// postfix chain (parsePostfix) of selector/call/index/composite-literal suffixes
// plus goal's postfix `?` unwrap. The postfix chain binds tightest, then unary,
// then binary by precedence level. `?` therefore wraps the fully-built postfix
// operand: `f(x)?` unwraps the call result and `a.b?` unwraps the selector.

// lowestBinaryPrec is the precedence of the loosest binary operator (||); a
// minPrec of lowestBinaryPrec admits every binary operator.
const lowestBinaryPrec = 1

// precedence returns the binary-operator precedence of k (1..5, Go semantics) or
// 0 when k is not a binary operator. `<-` (ARROW) is unary-only here, so it
// returns 0.
func precedence(k token.Kind) int {
	switch k {
	case token.LOR:
		return 1
	case token.LAND:
		return 2
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return 3
	case token.ADD, token.SUB, token.OR, token.XOR:
		return 4
	case token.MUL, token.QUO, token.REM, token.SHL, token.SHR, token.AND, token.AND_NOT:
		return 5
	}
	return 0
}

// parseExpr parses a full expression: a precedence-climbing tower of binary
// operators over unary operands, each operand carrying its postfix chain.
func (p *parser) parseExpr() ast.Expr {
	return p.parseBinary(lowestBinaryPrec)
}

// parseBinary parses a binary-operator expression whose operators have
// precedence >= minPrec, left associative. With minPrec one above the current
// operator's precedence on the recursive call, equal-precedence operators nest
// to the left.
func (p *parser) parseBinary(minPrec int) ast.Expr {
	x := p.parseUnary()
	for {
		opPrec := precedence(p.kind())
		if opPrec < minPrec {
			return x
		}
		op := p.advance()
		y := p.parseBinary(opPrec + 1)
		x = &ast.BinaryExpr{X: x, OpPos: op.Pos, Op: op.Kind, Y: y}
	}
}

// parseUnary parses a prefix/unary expression. `+ - ! ^ & <-` produce a
// UnaryExpr over the (recursively unary) operand; `*x` produces a StarExpr
// (pointer dereference); anything else is an operand plus its postfix chain.
func (p *parser) parseUnary() ast.Expr {
	switch p.kind() {
	case token.ADD, token.SUB, token.NOT, token.XOR, token.AND, token.ARROW:
		op := p.advance()
		return &ast.UnaryExpr{OpPos: op.Pos, Op: op.Kind, X: p.parseUnary()}
	case token.MUL:
		star := p.advance()
		return &ast.StarExpr{Star: star.Pos, X: p.parseUnary()}
	default:
		return p.parsePostfix(p.parseOperand())
	}
}

// parseOperand parses a primary operand: an identifier, a basic literal, or a
// parenthesized expression.
func (p *parser) parseOperand() ast.Expr {
	t := p.cur()
	switch t.Kind {
	case token.MATCH:
		// Value-position match: `var x = match s { … }`, `return match s { … }`.
		return p.parseMatchExpr()
	case token.IDENT:
		p.advance()
		return &ast.Ident{NamePos: t.Pos, Name: t.Lit}
	case token.INT, token.FLOAT, token.IMAG, token.CHAR, token.STRING:
		p.advance()
		return &ast.BasicLit{ValuePos: t.Pos, Kind: t.Kind, Value: t.Lit}
	case token.LPAREN:
		p.advance()
		prev := p.exprLev
		p.exprLev++
		inner := p.parseExpr()
		p.exprLev = prev
		rp := p.expect(token.RPAREN)
		return &ast.ParenExpr{Lparen: t.Pos, X: inner, Rparen: rp.Pos}
	case token.LBRACK, token.MAP, token.STRUCT, token.CHAN, token.INTERFACE:
		// A type literal in operand position: an array/slice, map, struct, channel,
		// or interface type used as a conversion target (`[]byte(p)`), a builtin
		// argument (`make(chan int, 1)`), or the type of a composite literal
		// (`map[string]string{}`, `[]T{...}`). parsePostfix then takes a
		// trailing `(` as the conversion call and a trailing `{` as the composite
		// body (compositeOK admits these type forms).
		return p.parseType()
	case token.FUNC:
		return p.parseFuncOperand()
	default:
		p.errorf(t.Pos, "expected expression, found %s", describe(t))
		p.advance()
		return &ast.Ident{NamePos: t.Pos, Name: t.Lit}
	}
}

// parseFuncOperand parses a `func` in expression position: a function literal
// `func(sig) { body }` (the common case), or — when no body follows — a bare
// function type used as a conversion or type operand. The body is parsed with
// the expression level reset to 0, so composite literals and statements inside
// the body are not suppressed when the literal itself appears in a control-clause
// header (exprLev < 0).
func (p *parser) parseFuncOperand() ast.Expr {
	kw := p.expect(token.FUNC)
	ft := p.parseSignature()
	ft.Func = kw.Pos
	if !p.at(token.LBRACE) {
		return ft
	}
	prev := p.exprLev
	p.exprLev = 0
	body := p.parseBlock()
	p.exprLev = prev
	return &ast.FuncLit{Type: ft, Body: body}
}

// parseTypeAssert parses a type assertion suffix "(T)" applied to x, with the
// cursor on "(" (the leading "." already consumed). The reserved form "x.(type)"
// — valid only as a type-switch guard — yields a TypeAssertExpr with a nil Type.
func (p *parser) parseTypeAssert(x ast.Expr) ast.Expr {
	lp := p.expect(token.LPAREN)
	ta := &ast.TypeAssertExpr{X: x, Lparen: lp.Pos}
	if p.at(token.TYPE) {
		p.advance() // "x.(type)" guard — Type stays nil
	} else {
		ta.Type = p.parseType()
	}
	ta.Rparen = p.expect(token.RPAREN).Pos
	return ta
}

// parsePostfix applies any chain of selector, call, index, and composite-literal
// suffixes to x.
func (p *parser) parsePostfix(x ast.Expr) ast.Expr {
	for {
		switch p.kind() {
		case token.PERIOD:
			p.advance()
			if p.at(token.LPAREN) {
				x = p.parseTypeAssert(x)
			} else {
				x = &ast.SelectorExpr{X: x, Sel: p.ident()}
			}
		case token.LPAREN:
			x = p.parseCallSuffix(x)
		case token.LBRACK:
			x = p.parseIndexSuffix(x)
		case token.QUESTION:
			q := p.advance()
			x = &ast.UnwrapExpr{X: x, Question: q.Pos}
		case token.LBRACE:
			// In a control-clause header (exprLev < 0), a "{" begins the body
			// block, not a composite literal.
			if p.exprLev < 0 || !compositeOK(x) {
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
	case *ast.Ident, *ast.SelectorExpr, *ast.IndexExpr, *ast.IndexListExpr,
		*ast.ArrayType, *ast.MapType, *ast.StructType:
		return true
	}
	return false
}

// parseCallSuffix parses a call argument list applied to fun. A list containing
// at least one labeled argument (`Label: Value`) is variant construction and
// yields an *ast.VariantLit; an all-positional list yields an ordinary
// *ast.CallExpr.
func (p *parser) parseCallSuffix(fun ast.Expr) ast.Expr {
	lp := p.expect(token.LPAREN)
	prev := p.exprLev
	p.exprLev++
	var args []ast.Expr
	var ellipsis token.Pos
	labeled := false
	for !p.at(token.RPAREN) && !p.at(token.EOF) {
		arg := p.parseCallArg()
		if _, ok := arg.(*ast.LabeledArg); ok {
			labeled = true
		}
		args = append(args, arg)
		if p.at(token.ELLIPSIS) { // variadic spread f(a, xs...) — must be the last arg
			ellipsis = p.advance().Pos
			if p.at(token.COMMA) { // tolerate a trailing comma after the spread
				p.advance()
			}
			break
		}
		if p.at(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.exprLev = prev
	rp := p.expect(token.RPAREN)
	if labeled {
		return p.makeVariantLit(fun, lp.Pos, args, rp.Pos)
	}
	return &ast.CallExpr{Fun: fun, Lparen: lp.Pos, Args: args, Ellipsis: ellipsis, Rparen: rp.Pos}
}

// parseIndexSuffix parses a "[...]" suffix applied to x. A ':' makes it a slice
// expression x[low?:high?] or x[low?:high:max] (any bound omittable), yielding an
// *ast.SliceExpr; otherwise a single index yields an *ast.IndexExpr and a
// comma-separated list (a multi-element generic type-argument list such as
// Result[int, error]) yields an *ast.IndexListExpr. This single entry point
// serves both type position (typeNameFrom) and expression position
// (parsePostfix). Elements parse via parseExpr, which — because parseOperand
// accepts type-literal starts — also covers type arguments like []byte.
func (p *parser) parseIndexSuffix(x ast.Expr) ast.Expr {
	lb := p.expect(token.LBRACK)
	prev := p.exprLev
	p.exprLev++

	// A leading ':' is a slice with an empty low bound (x[:high]).
	if p.at(token.COLON) {
		return p.finishSlice(x, lb.Pos, prev, nil)
	}
	first := p.parseExpr()
	if p.at(token.COLON) {
		return p.finishSlice(x, lb.Pos, prev, first)
	}

	indices := []ast.Expr{first}
	for p.at(token.COMMA) {
		p.advance()
		if p.at(token.RBRACK) { // tolerate a trailing comma
			break
		}
		indices = append(indices, p.parseExpr())
	}
	p.exprLev = prev
	rb := p.expect(token.RBRACK)
	if len(indices) == 1 {
		return &ast.IndexExpr{X: x, Lbrack: lb.Pos, Index: indices[0], Rbrack: rb.Pos}
	}
	return &ast.IndexListExpr{X: x, Lbrack: lb.Pos, Indices: indices, Rbrack: rb.Pos}
}

// finishSlice parses the colon-separated remainder of a slice expression after
// its optional low bound (low is nil for x[:high]). The cursor is on the first
// ':'. It accepts the two- and three-index forms x[low:high] and
// x[low:high:max], with any bound omitted. prevLev restores the caller's
// expression level once the bracket closes.
func (p *parser) finishSlice(x ast.Expr, lbrack token.Pos, prevLev int, low ast.Expr) ast.Expr {
	s := &ast.SliceExpr{X: x, Lbrack: lbrack, Low: low}
	p.expect(token.COLON)
	if !p.at(token.RBRACK) && !p.at(token.COLON) {
		s.High = p.parseExpr()
	}
	if p.at(token.COLON) { // full slice: x[low:high:max]
		p.advance()
		if !p.at(token.RBRACK) {
			s.Max = p.parseExpr()
		}
	}
	p.exprLev = prevLev
	rb := p.expect(token.RBRACK)
	s.Rbrack = rb.Pos
	return s
}

// parseCompositeLit parses a composite-literal body { ... } for the given (or
// elided, when nil) type.
func (p *parser) parseCompositeLit(typ ast.Expr) ast.Expr {
	lb := p.expect(token.LBRACE)
	cl := &ast.CompositeLit{Type: typ, Lbrace: lb.Pos}
	prev := p.exprLev
	p.exprLev++
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		cl.Elts = append(cl.Elts, p.parseElement())
		if p.at(token.COMMA) {
			p.advance()
		} else {
			break
		}
	}
	p.exprLev = prev
	rb := p.expect(token.RBRACE)
	cl.Rbrace = rb.Pos
	return cl
}

// parseElement parses one composite-literal element: a spread (`...X`), a value,
// or a key:value pair.
func (p *parser) parseElement() ast.Expr {
	if p.at(token.ELLIPSIS) {
		return p.parseSpreadElement()
	}
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

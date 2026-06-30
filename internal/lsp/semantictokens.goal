package lsp

import (
	"encoding/json"

	"goal/internal/ast"
	"goal/internal/lexer"
	"goal/internal/parser"
	"goal/internal/token"
)

// semanticTokens answers a textDocument/semanticTokens/full request with the open document's
// tokens, classified by semantic role from the parsed AST. It is best-effort — an unknown URI
// or source that does not parse yields an empty (non-nil) token set rather than an error,
// matching the document-symbol contract.
func (s *Server) semanticTokens(raw json.RawMessage) SemanticTokens {
	empty := SemanticTokens{Data: []uint{}}
	var p SemanticTokensParams
	if !s.decode(raw, &p, "semanticTokens") {
		return empty
	}
	text, _, ok := s.buffer(p.TextDocument.URI)
	if !ok {
		return empty
	}
	return SemanticTokens{Data: computeSemanticTokens(text)}
}

// computeSemanticTokens classifies every confidently-typed lexeme in src and returns the LSP
// delta-encoded token stream: a flat array of 5-tuples [deltaLine, deltaStartChar, length,
// tokenType, tokenModifiers] in document order. Positions come from the lexer (which carries
// first-class token.Pos); the semantic ROLE of each identifier comes from the AST role map,
// so the classification is AST-driven rather than a flat token-kind colouring. Identifiers
// the AST cannot place (e.g. a builtin like int) are left unclassified rather than guessed.
func computeSemanticTokens(src string) []uint {
	roles := astRoles(src)
	data := []uint{}
	prevLine, prevChar := 0, 0
	for _, t := range lexer.Tokens(src) {
		if t.Kind == token.EOF {
			break
		}
		sem, ok := classifyToken(t, roles)
		if !ok {
			continue
		}
		pos := token.OffsetToPosition(src, t.Pos.Offset)
		line0 := pos.Line - 1
		char0 := pos.Col - 1
		dl := line0 - prevLine
		dc := char0
		if dl == 0 {
			dc = char0 - prevChar
		}
		data = append(data, uint(dl), uint(dc), uint(tokenLen(t)), uint(sem), 0)
		prevLine, prevChar = line0, char0
	}
	return data
}

// classifyToken maps one lexer token to its semantic token-type index, reporting ok=false for
// tokens that are not surfaced as semantic tokens (delimiters, ordinary arithmetic operators,
// and identifiers the AST could not classify — the editor's grammar colours those). Keywords,
// literals, comments, the goal-specific operators, and AST-placed identifiers are surfaced.
func classifyToken(t token.Token, roles map[int]int) (int, bool) {
	switch {
	case t.Kind == token.COMMENT || t.Kind == token.DOC_COMMENT:
		return semComment, true
	case t.Kind == token.STRING || t.Kind == token.CHAR:
		return semString, true
	case t.Kind == token.INT || t.Kind == token.FLOAT || t.Kind == token.IMAG:
		return semNumber, true
	case t.Kind.IsKeyword():
		return semKeyword, true // includes match, enum, assert
	case t.Kind == token.QUESTION || t.Kind == token.FAT_ARROW || t.Kind == token.ELLIPSIS:
		return semOperator, true
	case t.Kind == token.IDENT:
		if role, ok := roles[t.Pos.Offset]; ok {
			return role, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// tokenLen is the character length of a token. Identifiers, literals, and comments carry their
// source text in Lit; operators and delimiters carry an empty Lit, so their length comes from
// the canonical spelling (token.Kind.String()). Source is ASCII, so bytes == characters.
func tokenLen(t token.Token) int {
	if t.Lit != "" {
		return len(t.Lit)
	}
	return len(t.Kind.String())
}

// astRoles parses src and walks its AST to build a map from an identifier's byte offset to the
// semantic token-type it should be coloured with. Only structurally unambiguous roles are
// recorded (enum/variant/type/func/method names, parameters, struct fields, and the enum and
// variant references inside construction/pattern nodes). Source that does not parse yields an
// empty map, so classification degrades to keyword/literal/operator colouring without failing.
func astRoles(src string) map[int]int {
	roles := map[int]int{}
	file, err := parser.ParseFile(src)
	if err != nil || file == nil {
		return roles
	}
	ast.Walk(roleVisitor{roles}, file)
	return roles
}

// roleVisitor records the semantic role of an identifier the moment its enclosing declaration
// or expression is visited, so a name's role is fixed by its STRUCTURAL parent rather than by
// a second guess at the bare Ident. It always returns itself so Walk descends into nested
// bodies (where match patterns and constructions live).
type roleVisitor struct {
	roles map[int]int
}

func (v roleVisitor) Visit(n ast.Node) ast.Visitor {
	switch d := n.(type) {
	case *ast.EnumDecl:
		v.set(d.Name, semEnum)
		for _, vr := range d.Variants {
			v.set(vr.Name, semEnumMember)
			for _, f := range vr.Payload {
				v.set(f.Name, semProperty)
				v.setType(f.Type)
			}
		}
	case *ast.SealedInterfaceDecl:
		v.set(d.Name, semInterface)
	case *ast.TypeSpec:
		v.set(d.Name, typeSpecRole(d))
		if st, ok := d.Type.(*ast.StructType); ok && st.Fields != nil {
			v.fields(st.Fields, semProperty)
		}
		if it, ok := d.Type.(*ast.InterfaceType); ok && it.Methods != nil {
			v.fields(it.Methods, semMethod)
		}
	case *ast.FuncDecl:
		role := semFunction
		if d.Recv != nil {
			role = semMethod
			v.fields(d.Recv, semParameter)
		}
		v.set(d.Name, role)
		if d.Type != nil {
			v.fields(d.Type.Params, semParameter)
			v.fields(d.Type.Results, semParameter)
		}
	case *ast.VariantLit:
		v.set(d.Variant, semEnumMember)
		v.setEnum(d.Enum)
	case *ast.VariantPattern:
		v.set(d.Variant, semEnumMember)
		v.setEnum(d.Enum)
		v.set(d.Binding, semVariable)
	case *ast.CallExpr:
		switch fun := d.Fun.(type) {
		case *ast.Ident:
			v.set(fun, semFunction)
		case *ast.SelectorExpr:
			v.set(fun.Sel, semMethod)
		}
	}
	return v
}

// set records role for ident at its byte offset, if the identifier is present and named.
func (v roleVisitor) set(ident *ast.Ident, role int) {
	if ident == nil || ident.Name == "" {
		return
	}
	v.roles[ident.Pos().Offset] = role
}

// setEnum records an enum/sealed type reference (the `Status` in `Status.Active`). It fires
// only for a bare *Ident; a qualified reference is left to grammar colouring.
func (v roleVisitor) setEnum(e ast.Expr) {
	if id, ok := e.(*ast.Ident); ok {
		v.set(id, semEnum)
	}
}

// setType records a type-position *Ident as a type. Composite type expressions are left
// unclassified rather than guessed.
func (v roleVisitor) setType(e ast.Expr) {
	if id, ok := e.(*ast.Ident); ok {
		v.set(id, semType)
	}
}

// fields records each field name in fl with nameRole, and each field's *Ident type as a type.
func (v roleVisitor) fields(fl *ast.FieldList, nameRole int) {
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		for _, nm := range f.Names {
			v.set(nm, nameRole)
		}
		v.setType(f.Type)
	}
}

// typeSpecRole classifies a type declaration's name: a struct, an interface, or — for an alias
// or any other underlying type — a plain type.
func typeSpecRole(ts *ast.TypeSpec) int {
	if ts.Assign != (token.Pos{}) {
		return semType // type X = … alias
	}
	switch ts.Type.(type) {
	case *ast.StructType:
		return semStruct
	case *ast.InterfaceType:
		return semInterface
	default:
		return semType
	}
}

package backend

import (
	"fmt"
	"strings"

	"goal/internal/ast"
)

// emitter renders the plain-Go subset of the goal AST to Go source text. It is
// backend/go: US-026 seeded it with the ordinary-Go nodes a no-goal-specific
// fixture exercises, and US-032 completed the ordinary-Go statement set (notably
// expression switch / case clauses) so the emitter covers the whole Go subset
// the parser produces (package/import/func/var/const/type plus every statement,
// expression, and type form). Any node it does not handle — every goal-specific
// node (enum/match/`?`/construct/spread/assert, from/derive modifiers) — yields a
// descriptive error, since those are lowered by later stories (US-033+).
//
// The emitter does not format: it emits syntactically valid, token-correct Go
// (balanced braces, spaces between tokens), and the Formatter normalizes layout
// afterward. So readability here is irrelevant; only parseability matters.
type emitter struct {
	b   strings.Builder
	err error
}

// emitFile renders a whole *ast.File to Go source text, or returns the first
// unsupported-node error encountered.
func emitFile(f *ast.File) (string, error) {
	var e emitter
	e.file(f)
	if e.err != nil {
		return "", e.err
	}
	return e.b.String(), nil
}

// fail records the first error; subsequent emit calls short-circuit on it.
func (e *emitter) fail(format string, args ...any) {
	if e.err == nil {
		e.err = fmt.Errorf("backend: "+format, args...)
	}
}

func (e *emitter) p(s string) {
	if e.err == nil {
		e.b.WriteString(s)
	}
}

func (e *emitter) file(f *ast.File) {
	if f == nil || f.Name == nil {
		e.fail("file has no package name")
		return
	}
	e.p("package ")
	e.p(f.Name.Name)
	e.p("\n\n")
	for _, d := range f.Decls {
		e.decl(d)
		e.p("\n\n")
	}
}

func (e *emitter) decl(d ast.Decl) {
	switch d := d.(type) {
	case *ast.GenDecl:
		e.genDecl(d)
	case *ast.FuncDecl:
		e.funcDecl(d)
	default:
		e.fail("unsupported declaration %T", d)
	}
}

func (e *emitter) genDecl(d *ast.GenDecl) {
	e.p(d.Tok.String())
	e.p(" ")
	multi := len(d.Specs) > 1
	if multi {
		e.p("(\n")
	}
	for _, s := range d.Specs {
		e.spec(s)
		e.p("\n")
	}
	if multi {
		e.p(")")
	}
}

func (e *emitter) spec(s ast.Spec) {
	switch s := s.(type) {
	case *ast.ImportSpec:
		if s.Name != nil {
			e.p(s.Name.Name)
			e.p(" ")
		}
		if s.Path != nil {
			e.p(s.Path.Value)
		}
	case *ast.ValueSpec:
		e.identList(s.Names)
		if s.Type != nil {
			e.p(" ")
			e.expr(s.Type)
		}
		if len(s.Values) > 0 {
			e.p(" = ")
			e.exprList(s.Values)
		}
	case *ast.TypeSpec:
		if s.Name != nil {
			e.p(s.Name.Name)
		}
		e.p(" ")
		e.expr(s.Type)
	default:
		e.fail("unsupported spec %T", s)
	}
}

func (e *emitter) funcDecl(d *ast.FuncDecl) {
	if d.Mod != ast.FuncPlain {
		e.fail("unsupported func modifier %v (goal from/derive is a later story)", d.Mod)
		return
	}
	e.p("func ")
	if d.Recv != nil {
		e.fieldList(d.Recv, "(", ")")
		e.p(" ")
	}
	if d.Name != nil {
		e.p(d.Name.Name)
	}
	e.funcSig(d.Type)
	if d.Body != nil {
		e.p(" ")
		e.block(d.Body)
	}
}

// funcSig emits the parameter and result lists of a signature.
func (e *emitter) funcSig(t *ast.FuncType) {
	if t == nil {
		e.fail("function has no signature")
		return
	}
	e.fieldList(t.Params, "(", ")")
	if t.Results != nil && len(t.Results.List) > 0 {
		e.p(" ")
		// Multiple results, or a single named result, need parentheses; a single
		// unnamed result does not. gofmt will drop redundant parens, so we always
		// parenthesize when there is more than one field or any field is named.
		if len(t.Results.List) > 1 || len(t.Results.List[0].Names) > 0 {
			e.fieldList(t.Results, "(", ")")
		} else {
			e.expr(t.Results.List[0].Type)
		}
	}
}

// fieldList emits a comma-separated, parenthesized field list — the form used by
// parameter, result, and receiver lists. Struct fields and interface methods are
// NOT comma-separated (commas there are a Go syntax error gofmt cannot repair);
// they go through structType / interfaceType, which newline-separate instead.
func (e *emitter) fieldList(fl *ast.FieldList, open, close string) {
	e.p(open)
	if fl != nil {
		for i, f := range fl.List {
			if i > 0 {
				e.p(", ")
			}
			e.field(f)
		}
	}
	e.p(close)
}

// structType emits a struct type. Fields are newline-separated (a comma between
// struct fields is a Go syntax error); gofmt then aligns them. A goal
// `implements` clause is goal-specific and lowered by a later story (US-033).
func (e *emitter) structType(x *ast.StructType) {
	if x.Implements != nil {
		e.fail("unsupported struct implements clause (goal implements is a later story)")
		return
	}
	e.p("struct {\n")
	if x.Fields != nil {
		for _, f := range x.Fields.List {
			e.field(f)
			e.p("\n")
		}
	}
	e.p("}")
}

// interfaceType emits an interface type. Each element is on its own line: a named
// method renders as `Name(params) results` (no `func` keyword), an embedded
// interface as its type name.
func (e *emitter) interfaceType(x *ast.InterfaceType) {
	e.p("interface {\n")
	if x.Methods != nil {
		for _, m := range x.Methods.List {
			if len(m.Names) > 0 {
				e.identList(m.Names)
				if ft, ok := m.Type.(*ast.FuncType); ok {
					e.funcSig(ft)
				} else if m.Type != nil {
					e.p(" ")
					e.expr(m.Type)
				}
			} else if m.Type != nil {
				e.expr(m.Type)
			}
			e.p("\n")
		}
	}
	e.p("}")
}

func (e *emitter) field(f *ast.Field) {
	if len(f.Names) > 0 {
		e.identList(f.Names)
		e.p(" ")
	}
	if f.Type != nil {
		e.expr(f.Type)
	}
	if f.Tag != nil {
		e.p(" ")
		e.p(f.Tag.Value)
	}
}

func (e *emitter) block(b *ast.BlockStmt) {
	e.p("{\n")
	for _, s := range b.List {
		e.stmt(s)
		e.p("\n")
	}
	e.p("}")
}

func (e *emitter) stmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.BlockStmt:
		e.block(s)
	case *ast.ExprStmt:
		e.expr(s.X)
	case *ast.AssignStmt:
		e.exprList(s.Lhs)
		e.p(" ")
		e.p(s.Tok.String())
		e.p(" ")
		e.exprList(s.Rhs)
	case *ast.IncDecStmt:
		e.expr(s.X)
		e.p(s.Tok.String())
	case *ast.ReturnStmt:
		e.p("return")
		if len(s.Results) > 0 {
			e.p(" ")
			e.exprList(s.Results)
		}
	case *ast.IfStmt:
		e.ifStmt(s)
	case *ast.ForStmt:
		e.forStmt(s)
	case *ast.RangeStmt:
		e.rangeStmt(s)
	case *ast.SwitchStmt:
		e.switchStmt(s)
	case *ast.DeclStmt:
		e.decl(s.Decl)
	case *ast.DeferStmt:
		e.p("defer ")
		e.expr(s.Call)
	case *ast.GoStmt:
		e.p("go ")
		e.expr(s.Call)
	case *ast.BranchStmt:
		e.p(s.Tok.String())
		if s.Label != nil {
			e.p(" ")
			e.p(s.Label.Name)
		}
	case *ast.EmptyStmt:
		// nothing
	default:
		e.fail("unsupported statement %T", s)
	}
}

func (e *emitter) ifStmt(s *ast.IfStmt) {
	e.p("if ")
	if s.Init != nil {
		e.stmt(s.Init)
		e.p("; ")
	}
	e.expr(s.Cond)
	e.p(" ")
	e.block(s.Body)
	if s.Else != nil {
		e.p(" else ")
		e.stmt(s.Else)
	}
}

func (e *emitter) forStmt(s *ast.ForStmt) {
	e.p("for ")
	if s.Init != nil || s.Post != nil {
		if s.Init != nil {
			e.stmt(s.Init)
		}
		e.p("; ")
		if s.Cond != nil {
			e.expr(s.Cond)
		}
		e.p("; ")
		if s.Post != nil {
			e.stmt(s.Post)
		}
		e.p(" ")
	} else if s.Cond != nil {
		e.expr(s.Cond)
		e.p(" ")
	}
	e.block(s.Body)
}

// switchStmt emits an expression switch: an optional init statement, an optional
// tag expression, and a brace block of case/default clauses. Like the if/for
// headers, the tag is just an expression and the clauses carry their own
// statement lists; gofmt normalizes the layout afterward.
func (e *emitter) switchStmt(s *ast.SwitchStmt) {
	e.p("switch ")
	if s.Init != nil {
		e.stmt(s.Init)
		e.p("; ")
	}
	if s.Tag != nil {
		e.expr(s.Tag)
		e.p(" ")
	}
	e.p("{\n")
	if s.Body != nil {
		for _, c := range s.Body.List {
			cc, ok := c.(*ast.CaseClause)
			if !ok {
				e.fail("unsupported switch body element %T (expected case clause)", c)
				return
			}
			e.caseClause(cc)
		}
	}
	e.p("}")
}

// caseClause emits one clause of a switch: "case e1, e2:" (a non-empty
// expression list) or "default:" (an empty list), followed by the clause's
// statement list.
func (e *emitter) caseClause(c *ast.CaseClause) {
	if len(c.List) > 0 {
		e.p("case ")
		e.exprList(c.List)
		e.p(":\n")
	} else {
		e.p("default:\n")
	}
	for _, s := range c.Body {
		e.stmt(s)
		e.p("\n")
	}
}

func (e *emitter) rangeStmt(s *ast.RangeStmt) {
	e.p("for ")
	if s.Key != nil {
		e.expr(s.Key)
		if s.Value != nil {
			e.p(", ")
			e.expr(s.Value)
		}
		e.p(" ")
		e.p(s.Tok.String())
		e.p(" ")
	}
	e.p("range ")
	e.expr(s.X)
	e.p(" ")
	e.block(s.Body)
}

func (e *emitter) expr(x ast.Expr) {
	switch x := x.(type) {
	case *ast.Ident:
		e.p(x.Name)
	case *ast.BasicLit:
		e.p(x.Value)
	case *ast.ParenExpr:
		e.p("(")
		e.expr(x.X)
		e.p(")")
	case *ast.UnaryExpr:
		e.p(x.Op.String())
		e.expr(x.X)
	case *ast.BinaryExpr:
		e.expr(x.X)
		e.p(" ")
		e.p(x.Op.String())
		e.p(" ")
		e.expr(x.Y)
	case *ast.SelectorExpr:
		e.expr(x.X)
		e.p(".")
		e.p(x.Sel.Name)
	case *ast.StarExpr:
		e.p("*")
		e.expr(x.X)
	case *ast.IndexExpr:
		e.expr(x.X)
		e.p("[")
		e.expr(x.Index)
		e.p("]")
	case *ast.IndexListExpr:
		e.expr(x.X)
		e.p("[")
		e.exprList(x.Indices)
		e.p("]")
	case *ast.SliceExpr:
		e.sliceExpr(x)
	case *ast.CallExpr:
		e.expr(x.Fun)
		e.p("(")
		e.exprList(x.Args)
		e.p(")")
	case *ast.KeyValueExpr:
		e.expr(x.Key)
		e.p(": ")
		e.expr(x.Value)
	case *ast.CompositeLit:
		if x.Type != nil {
			e.expr(x.Type)
		}
		e.p("{")
		e.exprList(x.Elts)
		e.p("}")
	case *ast.FuncLit:
		e.p("func")
		e.funcSig(x.Type)
		e.p(" ")
		e.block(x.Body)
	// Type expressions.
	case *ast.ArrayType:
		e.p("[")
		if x.Len != nil {
			e.expr(x.Len)
		}
		e.p("]")
		e.expr(x.Elt)
	case *ast.MapType:
		e.p("map[")
		e.expr(x.Key)
		e.p("]")
		e.expr(x.Value)
	case *ast.StructType:
		e.structType(x)
	case *ast.InterfaceType:
		e.interfaceType(x)
	case *ast.FuncType:
		e.p("func")
		e.funcSig(x)
	case *ast.ChanType:
		e.chanType(x)
	case *ast.Ellipsis:
		e.p("...")
		if x.Elt != nil {
			e.expr(x.Elt)
		}
	default:
		e.fail("unsupported expression %T", x)
	}
}

func (e *emitter) sliceExpr(x *ast.SliceExpr) {
	e.expr(x.X)
	e.p("[")
	if x.Low != nil {
		e.expr(x.Low)
	}
	e.p(":")
	if x.High != nil {
		e.expr(x.High)
	}
	if x.Max != nil {
		e.p(":")
		e.expr(x.Max)
	}
	e.p("]")
}

func (e *emitter) chanType(x *ast.ChanType) {
	switch x.Dir {
	case ast.RecvOnly:
		e.p("<-chan ")
	case ast.SendOnly:
		e.p("chan<- ")
	default:
		e.p("chan ")
	}
	e.expr(x.Value)
}

func (e *emitter) identList(ids []*ast.Ident) {
	for i, id := range ids {
		if i > 0 {
			e.p(", ")
		}
		e.p(id.Name)
	}
}

func (e *emitter) exprList(xs []ast.Expr) {
	for i, x := range xs {
		if i > 0 {
			e.p(", ")
		}
		e.expr(x)
	}
}

package backend

import (
	"fmt"
	"strconv"
	"strings"

	"goal/internal/ast"
	"goal/internal/sema"
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
	// info carries the resolved semantic facts the goal-construct lowering reads
	// (enums, sealed interfaces); nil-safe — the plain-Go subset ignores it.
	info *sema.Info
	// pointerRecv is the set of type names with a pointer-receiver method, used to
	// address an `implements` assertion as `(*T)(nil)` rather than `T{}`.
	pointerRecv map[string]bool
	// fnKind is the enclosing function's Result/Option kind, so a `return
	// Result.Ok/Err` / `return Option.Some/None` constructor lowers to the native
	// (T, error) pair / pointer form. roNone outside a Result/Option function.
	fnKind roKind
	// renames maps an identifier to its replacement within the currently-emitting
	// match arm body (e.g. an Ok binding `cfg` -> a gensym `v`). Empty outside a
	// match arm; consulted by the Ident emission.
	renames map[string]string
	// fileIdents is the set of identifiers used anywhere in the file; it seeds each
	// function's gensym collision set so a generated name never clashes with source.
	fileIdents map[string]bool
	// taken is the in-scope identifier set for the current function — fileIdents
	// plus every name already minted by gensym in this function. Reset per funcDecl.
	taken map[string]bool
	// okName / errName are the current open-E Result function's generated success
	// and error return names (scope-aware, no `__goal_` prefix). A Result `?` and a
	// `return Result.Ok/Err` propagate through them; empty outside such a function.
	okName, errName string
}

// emitFile renders a whole *ast.File to Go source text, lowering goal-specific
// constructs through info, or returns the first unsupported-node error
// encountered.
func emitFile(f *ast.File, info *sema.Info) (string, error) {
	e := emitter{info: info, pointerRecv: pointerReceiverSet(f), fileIdents: fileIdentSet(f)}
	e.file(f)
	if e.err != nil {
		return "", e.err
	}
	return e.b.String(), nil
}

// gensym returns a fresh identifier built from want that collides with no name in
// scope for the current function, reserving it for the rest of the function. It is
// the scope-aware replacement for the magic `__goal_` prefix (US-035): the `?`
// propagation and match lowerings name their temporaries through it, so a
// generated name can never shadow a source identifier (e.g. a user's own `err`).
func (e *emitter) gensym(want string) string {
	if e.taken == nil {
		e.taken = map[string]bool{}
	}
	name := want
	for i := 1; e.taken[name]; i++ {
		name = want + strconv.Itoa(i)
	}
	e.taken[name] = true
	return name
}

// newScope returns a fresh copy of the file's identifier set — the collision base
// for one function's gensyms, so names minted in one function do not perturb the
// next.
func (e *emitter) newScope() map[string]bool {
	s := make(map[string]bool, len(e.fileIdents))
	for k := range e.fileIdents {
		s[k] = true
	}
	return s
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
		// A struct `implements` clause lowers to a separate marker/assertion decl
		// emitted right after the type declaration (the clause itself is dropped
		// from the struct by structType).
		e.implementsMarkers(d)
	case *ast.FuncDecl:
		e.funcDecl(d)
	case *ast.EnumDecl:
		e.enumDecl(d)
	case *ast.SealedInterfaceDecl:
		e.sealedInterfaceDecl(d)
	default:
		e.fail("unsupported declaration %T", d)
	}
}

// enumDecl lowers a goal `enum` to the §8.1 closed-sum encoding: a marker
// interface, one struct per variant, and a marker method per variant. The
// variants come from the resolved sema.Enum so a field type carrying an embedded
// comma is rendered correctly.
func (e *emitter) enumDecl(d *ast.EnumDecl) {
	if d.Name == nil {
		e.fail("enum declaration has no name")
		return
	}
	en := enumOf(e.info, d.Name.Name)
	if en == nil {
		e.fail("enum %s not resolved", d.Name.Name)
		return
	}
	e.p(genEnum(en))
}

// sealedInterfaceDecl lowers `sealed interface Name {}` to its marker interface
// `type Name interface{ isName() }`.
func (e *emitter) sealedInterfaceDecl(d *ast.SealedInterfaceDecl) {
	if d.Name == nil {
		e.fail("sealed interface declaration has no name")
		return
	}
	e.p(genSealedInterface(d.Name.Name))
}

// implementsMarkers emits the §8.5 marker/assertion for every struct in d that
// carries an `implements` clause: a sealed interface yields the marker method
// `func (T) isI() {}`; an ordinary interface yields the compile-time assertion
// `var _ I = T{}` (or `var _ I = (*T)(nil)` when T has a pointer-receiver method).
func (e *emitter) implementsMarkers(d *ast.GenDecl) {
	for _, s := range d.Specs {
		ts, ok := s.(*ast.TypeSpec)
		if !ok || ts.Name == nil {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Implements == nil {
			continue
		}
		e.p("\n\n")
		e.implementsMarker(ts.Name.Name, st.Implements)
	}
}

func (e *emitter) implementsMarker(typeName string, clause *ast.ImplementsClause) {
	iface := typeExprName(clause.Type)
	if iface == "" {
		e.fail("implements clause on %s has an unsupported interface type %T", typeName, clause.Type)
		return
	}
	switch {
	case isSealed(e.info, iface):
		e.p(genMarkerMethod(typeName, iface))
	case e.pointerRecv[typeName]:
		e.p(fmt.Sprintf("var _ %s = (*%s)(nil)", iface, typeName))
	default:
		e.p(fmt.Sprintf("var _ %s = %s{}", iface, typeName))
	}
}

// typeExprName renders a (possibly qualified) type name expression — an *Ident
// (`Shape`), a *SelectorExpr (`io.Writer`), or a pointer to one — to its text, or
// "" if the shape is unsupported.
func typeExprName(x ast.Expr) string {
	switch x := x.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		base := typeExprName(x.X)
		if base == "" || x.Sel == nil {
			return ""
		}
		return base + "." + x.Sel.Name
	case *ast.StarExpr:
		inner := typeExprName(x.X)
		if inner == "" {
			return ""
		}
		return "*" + inner
	default:
		return ""
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
	// Enter the function's gensym scope: seed the collision set with the source's
	// identifiers and, for an open-E Result function, mint the success/error return
	// names up front so the signature and the body's `?`/match lowering reference
	// them identically. All saved/restored so a sibling function starts clean (and
	// a nested func literal cannot leak its kind or names outward).
	kind, _ := resultOptionKind(d.Type)
	prevKind, prevOk, prevErr, prevTaken := e.fnKind, e.okName, e.errName, e.taken
	e.fnKind, e.taken, e.okName, e.errName = kind, e.newScope(), "", ""
	if kind == roResultOpen {
		e.okName = e.gensym("ok")
		e.errName = e.gensym("err")
	}
	defer func() {
		e.fnKind, e.okName, e.errName, e.taken = prevKind, prevOk, prevErr, prevTaken
	}()

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
		// An open-E Result[T, error] return lowers to native named Go returns
		// (ok T, err error, scope-aware gensyms): the named success return makes the
		// Err-path zero value available without synthesizing a type-specific zero
		// literal (§8.3). An Option[T] return needs no special case here — it falls
		// through to the IndexExpr lowering, which renders *T.
		if kind, success := resultOptionKind(t); kind == roResultOpen {
			// In a function declaration these names are minted by funcDecl; in a
			// bodyless context (an interface method, a func-type expression) there is
			// no body to agree with, so fall back to plain unused named returns.
			ok, errn := e.okName, e.errName
			if ok == "" {
				ok = "ok"
			}
			if errn == "" {
				errn = "err"
			}
			e.p("(" + ok + " ")
			e.expr(success)
			e.p(", " + errn + " error)")
			return
		}
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
// `implements` clause is dropped here — the satisfaction marker/assertion it
// implies is emitted as a separate decl by implementsMarkers — so the struct
// itself renders as plain Go.
func (e *emitter) structType(x *ast.StructType) {
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
		// A statement-position `match` over a Result/Option lowers to an if/else
		// split (§8.3/§8.4); a bare `expr?` discards the unwrapped value and
		// propagates only the failure; any other expression statement emits verbatim.
		switch x := s.X.(type) {
		case *ast.MatchExpr:
			e.matchStmt(x)
		case *ast.UnwrapExpr:
			e.unwrap("_", x, true)
		default:
			e.expr(s.X)
		}
	case *ast.AssignStmt:
		// `name := expr?` / `_ := expr?` lowers the `?` propagation (§3.7, §8.3): the
		// unwrapped value binds to name (or is discarded), the failure early-returns.
		if len(s.Rhs) == 1 {
			if u, ok := s.Rhs[0].(*ast.UnwrapExpr); ok {
				name := "_"
				if len(s.Lhs) == 1 {
					if id, ok := s.Lhs[0].(*ast.Ident); ok {
						name = id.Name
					}
				}
				e.unwrap(name, u, name == "_")
				return
			}
		}
		e.exprList(s.Lhs)
		e.p(" ")
		e.p(s.Tok.String())
		e.p(" ")
		e.exprList(s.Rhs)
	case *ast.IncDecStmt:
		e.expr(s.X)
		e.p(s.Tok.String())
	case *ast.ReturnStmt:
		e.returnStmt(s)
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
		// A match-arm binding rename (e.g. the Ok payload `cfg` -> a gensym `v`)
		// applies here; outside a renaming arm, renames is empty.
		if r, ok := e.renames[x.Name]; ok {
			e.p(r)
		} else {
			e.p(x.Name)
		}
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
		e.selectorExpr(x)
	case *ast.VariantLit:
		e.variantLit(x)
	case *ast.StarExpr:
		e.p("*")
		e.expr(x.X)
	case *ast.IndexExpr:
		e.indexExpr(x)
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

// selectorExpr emits a selector, lowering a data-less enum variant reference
// (`Status.Pending`, which parses to a SelectorExpr) to its construction encoding
// `Status(Status_Pending{})`. The guard requires the base to be an *Ident naming a
// resolved enum whose variant set contains the selector, so ordinary and
// package-qualified selectors (`io.Writer`, `c.n`) are emitted unchanged.
func (e *emitter) selectorExpr(x *ast.SelectorExpr) {
	if id, ok := x.X.(*ast.Ident); ok && x.Sel != nil {
		if en := enumOf(e.info, id.Name); en != nil && en.VSet[x.Sel.Name] {
			e.p(fmt.Sprintf("%s(%s_%s{})", id.Name, id.Name, x.Sel.Name))
			return
		}
	}
	e.expr(x.X)
	e.p(".")
	if x.Sel != nil {
		e.p(x.Sel.Name)
	}
}

// variantLit emits a payload variant construction `Enum.V(label: x)` as its
// encoding `Enum(Enum_V{Label: x})`: labels are exported and argument values are
// emitted recursively, so a nested construction in a payload lowers for free.
func (e *emitter) variantLit(x *ast.VariantLit) {
	enum, ok := x.Enum.(*ast.Ident)
	if !ok || enumOf(e.info, enum.Name) == nil {
		e.fail("unsupported variant construction (enum not resolved): %T", x.Enum)
		return
	}
	if x.Variant == nil {
		e.fail("variant construction has no variant tag")
		return
	}
	e.p(fmt.Sprintf("%s(%s_%s{", enum.Name, enum.Name, x.Variant.Name))
	for i, a := range x.Args {
		if i > 0 {
			e.p(", ")
		}
		la, ok := a.(*ast.LabeledArg)
		if !ok {
			e.fail("unsupported non-labeled variant argument %T", a)
			return
		}
		if la.Label != nil {
			e.p(exported(la.Label.Name))
			e.p(": ")
		}
		e.expr(la.Value)
	}
	e.p("})")
}

// indexExpr emits a single-index expression, lowering an `Option[T]` type to its
// pointer encoding `*T` (§8.4). The guard requires the base to be the `Option`
// type name, so ordinary indexing (`xs[0]`) and other generic instantiations are
// emitted unchanged.
func (e *emitter) indexExpr(x *ast.IndexExpr) {
	if id, ok := x.X.(*ast.Ident); ok && id.Name == "Option" {
		e.p("*")
		e.expr(x.Index)
		return
	}
	e.expr(x.X)
	e.p("[")
	e.expr(x.Index)
	e.p("]")
}

// returnStmt emits a return, lowering a Result/Option constructor in the
// enclosing function (§8.3/§8.4) to the native (T, error) pair / pointer form.
func (e *emitter) returnStmt(s *ast.ReturnStmt) {
	if len(s.Results) == 1 {
		switch e.fnKind {
		case roResultOpen:
			if e.emitResultReturn(s.Results[0]) {
				return
			}
		case roOption:
			if e.emitOptionReturn(s.Results[0]) {
				return
			}
		}
	}
	e.p("return")
	if len(s.Results) > 0 {
		e.p(" ")
		e.exprList(s.Results)
	}
}

// emitResultReturn lowers `return Result.Ok(X)` -> `return X, nil` and
// `return Result.Err(X)` -> `return ok, X` (the function's named zero success
// return). It reports whether it handled the expression.
func (e *emitter) emitResultReturn(x ast.Expr) bool {
	call, ok := x.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}
	if base, ok := sel.X.(*ast.Ident); !ok || base.Name != "Result" {
		return false
	}
	switch sel.Sel.Name {
	case "Ok":
		e.p("return ")
		e.exprList(call.Args)
		e.p(", nil")
		return true
	case "Err":
		e.p("return " + e.okName + ", ")
		e.exprList(call.Args)
		return true
	}
	return false
}

// emitOptionReturn lowers `return Option.None` -> `return nil` and
// `return Option.Some(x)` -> `return &x` (addressable identifier) or a boxed
// `some := x; return &some` (a gensym; §8.4). It reports whether it handled
// the expression.
func (e *emitter) emitOptionReturn(x ast.Expr) bool {
	switch v := x.(type) {
	case *ast.SelectorExpr:
		if base, ok := v.X.(*ast.Ident); ok && base.Name == "Option" && v.Sel != nil && v.Sel.Name == "None" {
			e.p("return nil")
			return true
		}
	case *ast.CallExpr:
		sel, ok := v.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Some" {
			return false
		}
		if base, ok := sel.X.(*ast.Ident); !ok || base.Name != "Option" {
			return false
		}
		if len(v.Args) != 1 {
			return false
		}
		if _, ok := v.Args[0].(*ast.Ident); ok {
			e.p("return &")
			e.expr(v.Args[0])
			return true
		}
		some := e.gensym("some")
		e.p(some + " := ")
		e.expr(v.Args[0])
		e.p("\nreturn &" + some)
		return true
	}
	return false
}

// unwrap lowers a postfix `?` (ast.UnwrapExpr) at statement position: `name :=
// expr?` binds the unwrapped value, a bare `expr?` or `_ := expr?` discards it,
// and either way the enclosing function's failure (the Err / None) is
// early-returned (§3.7, §8.3). All temporaries are scope-aware gensyms — there is
// no `__goal_` prefix (US-035).
func (e *emitter) unwrap(name string, u *ast.UnwrapExpr, discard bool) {
	switch e.fnKind {
	case roResultOpen:
		e.unwrapResult(name, u, discard)
	case roOption:
		e.unwrapOption(name, u, discard)
	default:
		e.fail("`?` outside a Result- or Option-returning function (open-E only; closed-E `?` is a later story)")
	}
}

// unwrapResult lowers `?` in an open-E Result function: it destructures the
// callee's trailing error and, on non-nil, returns the function's own generated
// (ok, err) pair. The number of values destructured follows the callee's lowered
// arity — a plain `error`-returning callee yields one value, a `Result` callee two
// — so an error-only `?` does not over-destructure. An unresolved callee keeps the
// two-value form.
func (e *emitter) unwrapResult(name string, u *ast.UnwrapExpr, discard bool) {
	n := 2
	if sig, ok := e.calleeSig(u.X); ok && sig.EndsInError && sig.Arity >= 1 {
		n = sig.Arity
	}
	if discard {
		e.p("if " + strings.Repeat("_, ", n-1) + e.errName + " := ")
		e.expr(u.X)
		e.p("; " + e.errName + " != nil {\nreturn " + e.okName + ", " + e.errName + "\n}")
		return
	}
	if sig, ok := e.calleeSig(u.X); ok && sig.Arity != 2 {
		e.fail("`?` binds a value but the callee returns %d value(s); write a bare `…?` to propagate only the error", sig.Arity)
		return
	}
	e.p(name + ", " + e.errName + " := ")
	e.expr(u.X)
	e.p("\nif " + e.errName + " != nil {\nreturn " + e.okName + ", " + e.errName + "\n}")
}

// unwrapOption lowers `?` in an Option function: it stores the *T result in a fresh
// pointer temp and, when nil, returns nil; otherwise it dereferences into the
// bound name. Each `?` site mints its own temp, so chained `?`s never collide.
func (e *emitter) unwrapOption(name string, u *ast.UnwrapExpr, discard bool) {
	o := e.gensym("o")
	if discard {
		e.p("if " + o + " := ")
		e.expr(u.X)
		e.p("; " + o + " == nil {\nreturn nil\n}")
		return
	}
	e.p(o + " := ")
	e.expr(u.X)
	e.p("\nif " + o + " == nil {\nreturn nil\n}\n" + name + " := *" + o)
}

// calleeSig returns the resolved signature of the function a `?` scrutinee directly
// calls (by name), so the destructure arity matches the callee's lowered shape. It
// reports false for a non-call, a non-identifier callee, or an unresolved name.
func (e *emitter) calleeSig(x ast.Expr) (sema.FuncSig, bool) {
	call, ok := x.(*ast.CallExpr)
	if !ok {
		return sema.FuncSig{}, false
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok || e.info == nil || e.info.FuncSignatures == nil {
		return sema.FuncSig{}, false
	}
	sig, ok := e.info.FuncSignatures[id.Name]
	return sig, ok
}

// matchStmt lowers a statement-position match over a Result or Option to an
// if/else split (§8.3/§8.4). Enum and value-position match are later stories
// (US-036) and yield a descriptive error here.
func (e *emitter) matchStmt(m *ast.MatchExpr) {
	switch q := matchQualifier(m); q {
	case "Result":
		e.resultMatch(m)
	case "Option":
		e.optionMatch(m)
	default:
		e.fail("unsupported statement-position match on %q (only Result/Option match is lowered; enum/value-position match is a later story)", q)
	}
}

// resultMatch lowers `match scrut { Result.Ok(v) => …; Result.Err(e) => … }` to
// `lhs, err := scrut; if err != nil { errBody } else { okBody }`. The destructure
// value/error names are fresh local gensyms (a statement-position match may sit in
// a function that is not itself Result-returning, e.g. a plain `handle`, so these
// are NOT the enclosing function's returns). The Ok binding is renamed to the
// value gensym (discarded with `_` when unused) and the Err binding to the error
// gensym, so an arm body that constructs another Result composes through the
// rename in emitResultReturn.
func (e *emitter) resultMatch(m *ast.MatchExpr) {
	if e.calleeMode(m.Subject) == sema.ModeResultClosed {
		e.fail("closed-E Result match is a later story (US-037)")
		return
	}
	okArm, errArm := armByVariant(m, "Ok"), armByVariant(m, "Err")
	if okArm == nil || errArm == nil {
		e.fail("Result match must have both Result.Ok and Result.Err arms")
		return
	}
	val, errVar := e.gensym("v"), e.gensym("err")
	okBinding := bindingName(okArm.Pattern)
	okLHS := "_"
	if okBinding != "" && usesIdent(okArm.Body, okBinding) {
		okLHS = val
	}
	e.p(okLHS + ", " + errVar + " := ")
	e.expr(m.Subject)
	e.p("\nif " + errVar + " != nil {\n")
	e.armBodyRenamed(errArm.Body, bindingName(errArm.Pattern), errVar)
	e.p("\n} else {\n")
	e.armBodyRenamed(okArm.Body, okBinding, val)
	e.p("\n}")
}

// optionMatch lowers `match opt { Option.Some(b) => …; Option.None => … }` to
// `if o := opt; o != nil { b := *o; someBody } else { noneBody }`, where `o` is a
// fresh local gensym. The Some binding keeps its name (declared only when used).
func (e *emitter) optionMatch(m *ast.MatchExpr) {
	someArm, noneArm := armByVariant(m, "Some"), armByVariant(m, "None")
	if someArm == nil || noneArm == nil {
		e.fail("Option match must have both Option.Some and Option.None arms")
		return
	}
	o := e.gensym("o")
	e.p("if " + o + " := ")
	e.expr(m.Subject)
	e.p("; " + o + " != nil {\n")
	if b := bindingName(someArm.Pattern); b != "" && usesIdent(someArm.Body, b) {
		e.p(b + " := *" + o + "\n")
	}
	e.armBody(someArm.Body)
	e.p("\n} else {\n")
	e.armBody(noneArm.Body)
	e.p("\n}")
}

// armByVariant returns the arm whose variant-pattern tag is variant, or nil.
func armByVariant(m *ast.MatchExpr, variant string) *ast.MatchArm {
	for _, arm := range m.Arms {
		if vp, ok := arm.Pattern.(*ast.VariantPattern); ok && vp.Variant != nil && vp.Variant.Name == variant {
			return arm
		}
	}
	return nil
}

// bindingName returns a variant pattern's payload binding name, or "".
func bindingName(p ast.Expr) string {
	if vp, ok := p.(*ast.VariantPattern); ok && vp.Binding != nil {
		return vp.Binding.Name
	}
	return ""
}

// calleeMode returns the Result/Option mode of the function a match scrutinee
// directly calls, so a closed-E Result match is not mis-lowered by the open-E
// path. It is ModeNone for a non-call scrutinee or an unresolved callee.
func (e *emitter) calleeMode(x ast.Expr) sema.Mode {
	call, ok := x.(*ast.CallExpr)
	if !ok {
		return sema.ModeNone
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok || e.info == nil || e.info.FuncSignatures == nil {
		return sema.ModeNone
	}
	return e.info.FuncSignatures[id.Name].Mode
}

// armBodyRenamed emits a match arm body with binding renamed to target for the
// duration of the body (the rename is scoped to this body alone).
func (e *emitter) armBodyRenamed(body ast.Node, binding, target string) {
	if binding != "" {
		if e.renames == nil {
			e.renames = map[string]string{}
		}
		e.renames[binding] = target
		defer delete(e.renames, binding)
	}
	e.armBody(body)
}

// armBody emits a match arm body: a statement/block as a statement, or an
// expression as an expression statement.
func (e *emitter) armBody(n ast.Node) {
	switch b := n.(type) {
	case nil:
		// empty arm body
	case ast.Stmt:
		e.stmt(b)
	case ast.Expr:
		e.expr(b)
	default:
		e.fail("unsupported match arm body %T", n)
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

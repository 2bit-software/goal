// Package ast defines the goal abstract syntax tree: the single structural model
// of goal source that every back-end and tool (parser, checker, Go backend, LSP,
// fmt) traverses. This file declares the core Go-subset nodes — the declarations,
// statements, expressions, and type expressions that goal shares with Go — plus
// the position-carrying Node interface and the category markers Decl/Stmt/Expr.
//
// It is modeled on go/ast, trimmed to the subset goal actually uses, and carries
// our own token.Pos so positions survive onto the tree (unlike the splice passes,
// which discarded offsets every pass). Goal-specific declaration and expression
// nodes (enum/sealed/implements/from-derive, match/patterns/construction) are
// added by later stories on top of this skeleton; Walk's type switch grows with
// them.
package ast

import "goal/internal/token"

// Node is the interface implemented by every AST node. Pos returns the position
// of the node's first token; End returns the position just past its last token.
type Node interface {
	Pos() token.Pos
	End() token.Pos
}

// Decl is the interface for declaration nodes (the children of a File). The
// unexported marker keeps the category closed to this package.
type Decl interface {
	Node
	declNode()
}

// Stmt is the interface for statement nodes.
type Stmt interface {
	Node
	stmtNode()
}

// Expr is the interface for expression and type-expression nodes.
type Expr interface {
	Node
	exprNode()
}

// Spec is the interface for the individual specs inside a GenDecl (import, value,
// type).
type Spec interface {
	Node
	specNode()
}

// ----------------------------------------------------------------------------
// Support nodes

// Field is one field in a FieldList: a (possibly empty) list of names sharing a
// type, with an optional struct tag. Used for struct fields, interface methods,
// and function parameters/results.
type Field struct {
	Names []*Ident   // field/parameter names; may be nil for embedded or unnamed
	Type  Expr       // field/parameter type
	Tag   *BasicLit  // struct tag; or nil
}

func (f *Field) Pos() token.Pos {
	if len(f.Names) > 0 {
		return f.Names[0].Pos()
	}
	if f.Type != nil {
		return f.Type.Pos()
	}
	return token.Pos{}
}

func (f *Field) End() token.Pos {
	if f.Tag != nil {
		return f.Tag.End()
	}
	if f.Type != nil {
		return f.Type.End()
	}
	if len(f.Names) > 0 {
		return f.Names[len(f.Names)-1].End()
	}
	return token.Pos{}
}

// FieldList is a parenthesized or braced list of Fields (parameters, results,
// struct fields, interface methods).
type FieldList struct {
	Opening token.Pos // position of opening "(", "[" or "{"; may be zero
	List    []*Field  // field entries
	Closing token.Pos // position of closing ")", "]" or "}"; may be zero
}

func (fl *FieldList) Pos() token.Pos {
	if fl.Opening != (token.Pos{}) {
		return fl.Opening
	}
	if len(fl.List) > 0 {
		return fl.List[0].Pos()
	}
	return token.Pos{}
}

func (fl *FieldList) End() token.Pos {
	if fl.Closing != (token.Pos{}) {
		return fl.Closing
	}
	if n := len(fl.List); n > 0 {
		return fl.List[n-1].End()
	}
	return token.Pos{}
}

// File is the root of one parsed source file.
type File struct {
	Package token.Pos      // position of the "package" keyword
	Name    *Ident         // package name
	Imports []*ImportSpec  // the file's import specs (also reachable via Decls)
	Decls   []Decl         // top-level declarations
}

func (f *File) Pos() token.Pos { return f.Package }

func (f *File) End() token.Pos {
	if n := len(f.Decls); n > 0 {
		return f.Decls[n-1].End()
	}
	if f.Name != nil {
		return f.Name.End()
	}
	return f.Package
}

// ----------------------------------------------------------------------------
// Declarations

// GenDecl is a general declaration: an import, const, var, or type declaration.
// Tok is one of token.IMPORT, token.CONST, token.VAR, or token.TYPE.
type GenDecl struct {
	TokPos token.Pos   // position of Tok
	Tok    token.Kind  // IMPORT, CONST, VAR, or TYPE
	Specs  []Spec      // the declared specs
}

func (d *GenDecl) Pos() token.Pos { return d.TokPos }

func (d *GenDecl) End() token.Pos {
	if n := len(d.Specs); n > 0 {
		return d.Specs[n-1].End()
	}
	return d.TokPos
}

func (*GenDecl) declNode() {}

// FuncDecl is a function or method declaration. Recv is nil for a plain function.
type FuncDecl struct {
	Recv *FieldList // receiver (methods); or nil
	Name *Ident     // function/method name
	Type *FuncType  // signature (params and results)
	Body *BlockStmt // body; or nil for a bodyless declaration
}

func (d *FuncDecl) Pos() token.Pos {
	if d.Type != nil {
		return d.Type.Pos()
	}
	if d.Name != nil {
		return d.Name.Pos()
	}
	return token.Pos{}
}

func (d *FuncDecl) End() token.Pos {
	if d.Body != nil {
		return d.Body.End()
	}
	if d.Type != nil {
		return d.Type.End()
	}
	if d.Name != nil {
		return d.Name.End()
	}
	return token.Pos{}
}

func (*FuncDecl) declNode() {}

// ----------------------------------------------------------------------------
// Specs

// ImportSpec is a single import within an import GenDecl.
type ImportSpec struct {
	Name *Ident    // local package name (incl. "." and "_"); or nil
	Path *BasicLit // import path string literal
}

func (s *ImportSpec) Pos() token.Pos {
	if s.Name != nil {
		return s.Name.Pos()
	}
	if s.Path != nil {
		return s.Path.Pos()
	}
	return token.Pos{}
}

func (s *ImportSpec) End() token.Pos {
	if s.Path != nil {
		return s.Path.End()
	}
	if s.Name != nil {
		return s.Name.End()
	}
	return token.Pos{}
}

func (*ImportSpec) specNode() {}

// ValueSpec is a const or var spec: a list of names with an optional type and
// optional values.
type ValueSpec struct {
	Names  []*Ident // value names
	Type   Expr     // value type; or nil
	Values []Expr   // initial values; or nil
}

func (s *ValueSpec) Pos() token.Pos {
	if len(s.Names) > 0 {
		return s.Names[0].Pos()
	}
	return token.Pos{}
}

func (s *ValueSpec) End() token.Pos {
	if n := len(s.Values); n > 0 {
		return s.Values[n-1].End()
	}
	if s.Type != nil {
		return s.Type.End()
	}
	if n := len(s.Names); n > 0 {
		return s.Names[n-1].End()
	}
	return token.Pos{}
}

func (*ValueSpec) specNode() {}

// TypeSpec is a single type declaration: a name bound to a type expression.
type TypeSpec struct {
	Name *Ident // type name
	Type Expr   // the underlying type expression
}

func (s *TypeSpec) Pos() token.Pos {
	if s.Name != nil {
		return s.Name.Pos()
	}
	return token.Pos{}
}

func (s *TypeSpec) End() token.Pos {
	if s.Type != nil {
		return s.Type.End()
	}
	if s.Name != nil {
		return s.Name.End()
	}
	return token.Pos{}
}

func (*TypeSpec) specNode() {}

// ----------------------------------------------------------------------------
// Expressions

// Ident is an identifier.
type Ident struct {
	NamePos token.Pos // identifier position
	Name    string    // identifier text
}

func (e *Ident) Pos() token.Pos { return e.NamePos }
func (e *Ident) End() token.Pos {
	return token.Pos{Offset: e.NamePos.Offset + len(e.Name), Line: e.NamePos.Line, Col: e.NamePos.Col + len(e.Name)}
}
func (*Ident) exprNode() {}

// BasicLit is a literal of basic type (int, float, imag, char, string).
type BasicLit struct {
	ValuePos token.Pos  // literal position
	Kind     token.Kind // INT, FLOAT, IMAG, CHAR, or STRING
	Value    string     // literal text, e.g. "42", "\"abc\""
}

func (e *BasicLit) Pos() token.Pos { return e.ValuePos }
func (e *BasicLit) End() token.Pos {
	return token.Pos{Offset: e.ValuePos.Offset + len(e.Value), Line: e.ValuePos.Line, Col: e.ValuePos.Col + len(e.Value)}
}
func (*BasicLit) exprNode() {}

// ParenExpr is a parenthesized expression.
type ParenExpr struct {
	Lparen token.Pos // position of "("
	X      Expr      // parenthesized expression
	Rparen token.Pos // position of ")"
}

func (e *ParenExpr) Pos() token.Pos { return e.Lparen }
func (e *ParenExpr) End() token.Pos {
	return token.Pos{Offset: e.Rparen.Offset + 1, Line: e.Rparen.Line, Col: e.Rparen.Col + 1}
}
func (*ParenExpr) exprNode() {}

// UnaryExpr is a unary expression. Op is the operator kind; for postfix unwrap
// (token.QUESTION) it is still represented here in the Go subset, though goal's
// dedicated UnwrapExpr arrives in a later story.
type UnaryExpr struct {
	OpPos token.Pos  // position of Op
	Op    token.Kind // operator kind
	X     Expr       // operand
}

func (e *UnaryExpr) Pos() token.Pos { return e.OpPos }
func (e *UnaryExpr) End() token.Pos {
	if e.X != nil {
		return e.X.End()
	}
	return e.OpPos
}
func (*UnaryExpr) exprNode() {}

// BinaryExpr is a binary expression.
type BinaryExpr struct {
	X     Expr       // left operand
	OpPos token.Pos  // position of Op
	Op    token.Kind // operator kind
	Y     Expr       // right operand
}

func (e *BinaryExpr) Pos() token.Pos {
	if e.X != nil {
		return e.X.Pos()
	}
	return e.OpPos
}
func (e *BinaryExpr) End() token.Pos {
	if e.Y != nil {
		return e.Y.End()
	}
	return e.OpPos
}
func (*BinaryExpr) exprNode() {}

// SelectorExpr is a selector x.Sel.
type SelectorExpr struct {
	X   Expr   // expression
	Sel *Ident // field or method selector
}

func (e *SelectorExpr) Pos() token.Pos {
	if e.X != nil {
		return e.X.Pos()
	}
	return e.Sel.Pos()
}
func (e *SelectorExpr) End() token.Pos {
	if e.Sel != nil {
		return e.Sel.End()
	}
	return e.X.End()
}
func (*SelectorExpr) exprNode() {}

// IndexExpr is an index expression x[Index].
type IndexExpr struct {
	X      Expr      // expression
	Lbrack token.Pos // position of "["
	Index  Expr      // index expression
	Rbrack token.Pos // position of "]"
}

func (e *IndexExpr) Pos() token.Pos {
	if e.X != nil {
		return e.X.Pos()
	}
	return e.Lbrack
}
func (e *IndexExpr) End() token.Pos {
	return token.Pos{Offset: e.Rbrack.Offset + 1, Line: e.Rbrack.Line, Col: e.Rbrack.Col + 1}
}
func (*IndexExpr) exprNode() {}

// SliceExpr is a slice expression x[Low:High:Max].
type SliceExpr struct {
	X      Expr      // expression
	Lbrack token.Pos // position of "["
	Low    Expr      // begin of slice range; or nil
	High   Expr      // end of slice range; or nil
	Max    Expr      // maximum capacity of slice; or nil
	Rbrack token.Pos // position of "]"
}

func (e *SliceExpr) Pos() token.Pos {
	if e.X != nil {
		return e.X.Pos()
	}
	return e.Lbrack
}
func (e *SliceExpr) End() token.Pos {
	return token.Pos{Offset: e.Rbrack.Offset + 1, Line: e.Rbrack.Line, Col: e.Rbrack.Col + 1}
}
func (*SliceExpr) exprNode() {}

// CallExpr is a function or method call (or a conversion).
type CallExpr struct {
	Fun    Expr      // function expression
	Lparen token.Pos // position of "("
	Args   []Expr    // function arguments
	Rparen token.Pos // position of ")"
}

func (e *CallExpr) Pos() token.Pos {
	if e.Fun != nil {
		return e.Fun.Pos()
	}
	return e.Lparen
}
func (e *CallExpr) End() token.Pos {
	return token.Pos{Offset: e.Rparen.Offset + 1, Line: e.Rparen.Line, Col: e.Rparen.Col + 1}
}
func (*CallExpr) exprNode() {}

// StarExpr is a pointer dereference *X or a pointer type *T.
type StarExpr struct {
	Star token.Pos // position of "*"
	X    Expr      // operand
}

func (e *StarExpr) Pos() token.Pos { return e.Star }
func (e *StarExpr) End() token.Pos {
	if e.X != nil {
		return e.X.End()
	}
	return e.Star
}
func (*StarExpr) exprNode() {}

// KeyValueExpr is a key: value pair in a composite literal.
type KeyValueExpr struct {
	Key   Expr      // key
	Colon token.Pos // position of ":"
	Value Expr      // value
}

func (e *KeyValueExpr) Pos() token.Pos {
	if e.Key != nil {
		return e.Key.Pos()
	}
	return e.Colon
}
func (e *KeyValueExpr) End() token.Pos {
	if e.Value != nil {
		return e.Value.End()
	}
	return e.Colon
}
func (*KeyValueExpr) exprNode() {}

// CompositeLit is a composite literal Type{Elts}.
type CompositeLit struct {
	Type   Expr      // literal type; or nil for an elided type
	Lbrace token.Pos // position of "{"
	Elts   []Expr    // list of composite elements
	Rbrace token.Pos // position of "}"
}

func (e *CompositeLit) Pos() token.Pos {
	if e.Type != nil {
		return e.Type.Pos()
	}
	return e.Lbrace
}
func (e *CompositeLit) End() token.Pos {
	return token.Pos{Offset: e.Rbrace.Offset + 1, Line: e.Rbrace.Line, Col: e.Rbrace.Col + 1}
}
func (*CompositeLit) exprNode() {}

// FuncLit is a function literal: a signature plus a body.
type FuncLit struct {
	Type *FuncType  // function signature
	Body *BlockStmt // function body
}

func (e *FuncLit) Pos() token.Pos {
	if e.Type != nil {
		return e.Type.Pos()
	}
	return token.Pos{}
}
func (e *FuncLit) End() token.Pos {
	if e.Body != nil {
		return e.Body.End()
	}
	if e.Type != nil {
		return e.Type.End()
	}
	return token.Pos{}
}
func (*FuncLit) exprNode() {}

// ----------------------------------------------------------------------------
// Type expressions

// ArrayType is an array or slice type [Len]Elt (Len nil for a slice).
type ArrayType struct {
	Lbrack token.Pos // position of "["
	Len    Expr      // array length; nil for a slice type
	Elt    Expr      // element type
}

func (e *ArrayType) Pos() token.Pos { return e.Lbrack }
func (e *ArrayType) End() token.Pos {
	if e.Elt != nil {
		return e.Elt.End()
	}
	return e.Lbrack
}
func (*ArrayType) exprNode() {}

// MapType is a map type map[Key]Value.
type MapType struct {
	Map   token.Pos // position of "map"
	Key   Expr      // key type
	Value Expr      // value type
}

func (e *MapType) Pos() token.Pos { return e.Map }
func (e *MapType) End() token.Pos {
	if e.Value != nil {
		return e.Value.End()
	}
	if e.Key != nil {
		return e.Key.End()
	}
	return e.Map
}
func (*MapType) exprNode() {}

// StructType is a struct type struct{ Fields }.
type StructType struct {
	Struct token.Pos  // position of "struct"
	Fields *FieldList // list of field declarations
}

func (e *StructType) Pos() token.Pos { return e.Struct }
func (e *StructType) End() token.Pos {
	if e.Fields != nil {
		return e.Fields.End()
	}
	return e.Struct
}
func (*StructType) exprNode() {}

// InterfaceType is an interface type interface{ Methods }.
type InterfaceType struct {
	Interface token.Pos  // position of "interface"
	Methods   *FieldList // list of embedded interfaces and methods
}

func (e *InterfaceType) Pos() token.Pos { return e.Interface }
func (e *InterfaceType) End() token.Pos {
	if e.Methods != nil {
		return e.Methods.End()
	}
	return e.Interface
}
func (*InterfaceType) exprNode() {}

// FuncType is a function signature: parameters and results.
type FuncType struct {
	Func    token.Pos  // position of "func" keyword (may be zero for methods)
	Params  *FieldList // parameters
	Results *FieldList // results; or nil
}

func (e *FuncType) Pos() token.Pos {
	if e.Func != (token.Pos{}) {
		return e.Func
	}
	if e.Params != nil {
		return e.Params.Pos()
	}
	return token.Pos{}
}
func (e *FuncType) End() token.Pos {
	if e.Results != nil {
		return e.Results.End()
	}
	if e.Params != nil {
		return e.Params.End()
	}
	return e.Func
}
func (*FuncType) exprNode() {}

// ChanType is a channel type chan Value (with an optional direction).
type ChanType struct {
	Begin token.Pos  // position of "chan" keyword or "<-" (whichever comes first)
	Dir   ChanDir    // channel direction
	Value Expr       // value type
}

// ChanDir is the direction of a channel type.
type ChanDir int

const (
	// SendRecv is a bidirectional channel (chan T).
	SendRecv ChanDir = iota
	// SendOnly is a send-only channel (chan<- T).
	SendOnly
	// RecvOnly is a receive-only channel (<-chan T).
	RecvOnly
)

func (e *ChanType) Pos() token.Pos { return e.Begin }
func (e *ChanType) End() token.Pos {
	if e.Value != nil {
		return e.Value.End()
	}
	return e.Begin
}
func (*ChanType) exprNode() {}

// Ellipsis is a "..." node, used for variadic parameter types (...Elt).
type Ellipsis struct {
	Ellipsis token.Pos // position of "..."
	Elt      Expr      // element type; or nil
}

func (e *Ellipsis) Pos() token.Pos { return e.Ellipsis }
func (e *Ellipsis) End() token.Pos {
	if e.Elt != nil {
		return e.Elt.End()
	}
	return token.Pos{Offset: e.Ellipsis.Offset + 3, Line: e.Ellipsis.Line, Col: e.Ellipsis.Col + 3}
}
func (*Ellipsis) exprNode() {}

// ----------------------------------------------------------------------------
// Statements

// BlockStmt is a braced statement list { ... }.
type BlockStmt struct {
	Lbrace token.Pos // position of "{"
	List   []Stmt    // statements
	Rbrace token.Pos // position of "}"
}

func (s *BlockStmt) Pos() token.Pos { return s.Lbrace }
func (s *BlockStmt) End() token.Pos {
	return token.Pos{Offset: s.Rbrace.Offset + 1, Line: s.Rbrace.Line, Col: s.Rbrace.Col + 1}
}
func (*BlockStmt) stmtNode() {}

// ExprStmt is a standalone expression used as a statement.
type ExprStmt struct {
	X Expr // expression
}

func (s *ExprStmt) Pos() token.Pos { return s.X.Pos() }
func (s *ExprStmt) End() token.Pos { return s.X.End() }
func (*ExprStmt) stmtNode()        {}

// AssignStmt is an assignment or short variable declaration. Tok is token.ASSIGN
// or token.DEFINE (or a compound-assign operator).
type AssignStmt struct {
	Lhs    []Expr     // left-hand side
	TokPos token.Pos  // position of Tok
	Tok    token.Kind // assignment token (ASSIGN, DEFINE, ADD_ASSIGN, ...)
	Rhs    []Expr     // right-hand side
}

func (s *AssignStmt) Pos() token.Pos {
	if len(s.Lhs) > 0 {
		return s.Lhs[0].Pos()
	}
	return s.TokPos
}
func (s *AssignStmt) End() token.Pos {
	if n := len(s.Rhs); n > 0 {
		return s.Rhs[n-1].End()
	}
	return s.TokPos
}
func (*AssignStmt) stmtNode() {}

// IncDecStmt is an increment or decrement statement (x++ / x--).
type IncDecStmt struct {
	X      Expr       // operand
	TokPos token.Pos  // position of Tok
	Tok    token.Kind // INC or DEC
}

func (s *IncDecStmt) Pos() token.Pos { return s.X.Pos() }
func (s *IncDecStmt) End() token.Pos {
	return token.Pos{Offset: s.TokPos.Offset + 2, Line: s.TokPos.Line, Col: s.TokPos.Col + 2}
}
func (*IncDecStmt) stmtNode() {}

// ReturnStmt is a return statement.
type ReturnStmt struct {
	Return  token.Pos // position of "return"
	Results []Expr    // returned expressions; or nil
}

func (s *ReturnStmt) Pos() token.Pos { return s.Return }
func (s *ReturnStmt) End() token.Pos {
	if n := len(s.Results); n > 0 {
		return s.Results[n-1].End()
	}
	return token.Pos{Offset: s.Return.Offset + 6, Line: s.Return.Line, Col: s.Return.Col + 6}
}
func (*ReturnStmt) stmtNode() {}

// IfStmt is an if statement, optionally with an init statement and an else branch.
type IfStmt struct {
	If   token.Pos // position of "if"
	Init Stmt      // initialization statement; or nil
	Cond Expr      // condition
	Body *BlockStmt
	Else Stmt // else branch (an *IfStmt or *BlockStmt); or nil
}

func (s *IfStmt) Pos() token.Pos { return s.If }
func (s *IfStmt) End() token.Pos {
	if s.Else != nil {
		return s.Else.End()
	}
	if s.Body != nil {
		return s.Body.End()
	}
	return s.If
}
func (*IfStmt) stmtNode() {}

// ForStmt is a for statement (Init/Cond/Post all optional).
type ForStmt struct {
	For  token.Pos // position of "for"
	Init Stmt      // initialization statement; or nil
	Cond Expr      // condition; or nil
	Post Stmt      // post statement; or nil
	Body *BlockStmt
}

func (s *ForStmt) Pos() token.Pos { return s.For }
func (s *ForStmt) End() token.Pos {
	if s.Body != nil {
		return s.Body.End()
	}
	return s.For
}
func (*ForStmt) stmtNode() {}

// RangeStmt is a for-range statement.
type RangeStmt struct {
	For    token.Pos  // position of "for"
	Key    Expr       // key/index iteration variable; or nil
	Value  Expr       // value iteration variable; or nil
	TokPos token.Pos  // position of Tok (ASSIGN or DEFINE)
	Tok    token.Kind // ASSIGN or DEFINE; 0 if no assignment
	X      Expr       // value to range over
	Body   *BlockStmt
}

func (s *RangeStmt) Pos() token.Pos { return s.For }
func (s *RangeStmt) End() token.Pos {
	if s.Body != nil {
		return s.Body.End()
	}
	return s.For
}
func (*RangeStmt) stmtNode() {}

// SwitchStmt is an expression switch statement.
type SwitchStmt struct {
	Switch token.Pos  // position of "switch"
	Init   Stmt       // initialization statement; or nil
	Tag    Expr       // switch tag expression; or nil
	Body   *BlockStmt // CaseClauses wrapped in a block
}

func (s *SwitchStmt) Pos() token.Pos { return s.Switch }
func (s *SwitchStmt) End() token.Pos {
	if s.Body != nil {
		return s.Body.End()
	}
	return s.Switch
}
func (*SwitchStmt) stmtNode() {}

// CaseClause is one case (or default) clause within a switch body.
type CaseClause struct {
	Case  token.Pos // position of "case" or "default"
	List  []Expr    // case expressions; nil for the default clause
	Colon token.Pos // position of ":"
	Body  []Stmt    // clause statements
}

func (s *CaseClause) Pos() token.Pos { return s.Case }
func (s *CaseClause) End() token.Pos {
	if n := len(s.Body); n > 0 {
		return s.Body[n-1].End()
	}
	return token.Pos{Offset: s.Colon.Offset + 1, Line: s.Colon.Line, Col: s.Colon.Col + 1}
}
func (*CaseClause) stmtNode() {}

// DeferStmt is a defer statement.
type DeferStmt struct {
	Defer token.Pos // position of "defer"
	Call  *CallExpr // the deferred call
}

func (s *DeferStmt) Pos() token.Pos { return s.Defer }
func (s *DeferStmt) End() token.Pos {
	if s.Call != nil {
		return s.Call.End()
	}
	return s.Defer
}
func (*DeferStmt) stmtNode() {}

// GoStmt is a go statement.
type GoStmt struct {
	Go   token.Pos // position of "go"
	Call *CallExpr // the called goroutine
}

func (s *GoStmt) Pos() token.Pos { return s.Go }
func (s *GoStmt) End() token.Pos {
	if s.Call != nil {
		return s.Call.End()
	}
	return s.Go
}
func (*GoStmt) stmtNode() {}

// BranchStmt is a break, continue, goto, or fallthrough statement.
type BranchStmt struct {
	TokPos token.Pos  // position of Tok
	Tok    token.Kind // BREAK, CONTINUE, GOTO, or FALLTHROUGH
	Label  *Ident     // label; or nil
}

func (s *BranchStmt) Pos() token.Pos { return s.TokPos }
func (s *BranchStmt) End() token.Pos {
	if s.Label != nil {
		return s.Label.End()
	}
	return token.Pos{Offset: s.TokPos.Offset + len(s.Tok.String()), Line: s.TokPos.Line, Col: s.TokPos.Col + len(s.Tok.String())}
}
func (*BranchStmt) stmtNode() {}

// DeclStmt is a declaration (const/var/type GenDecl) used as a statement.
type DeclStmt struct {
	Decl Decl // a *GenDecl
}

func (s *DeclStmt) Pos() token.Pos { return s.Decl.Pos() }
func (s *DeclStmt) End() token.Pos { return s.Decl.End() }
func (*DeclStmt) stmtNode()        {}

// EmptyStmt is an empty statement (a lone ";").
type EmptyStmt struct {
	Semicolon token.Pos // position of the ";"
}

func (s *EmptyStmt) Pos() token.Pos { return s.Semicolon }
func (s *EmptyStmt) End() token.Pos {
	return token.Pos{Offset: s.Semicolon.Offset + 1, Line: s.Semicolon.Line, Col: s.Semicolon.Col + 1}
}
func (*EmptyStmt) stmtNode() {}

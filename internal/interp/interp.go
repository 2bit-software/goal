package interp

// This file establishes the interpreter ENTRY seam: goscript is a back-end over
// the SHARED front-end (REWRITE-ARCHITECTURE.md §3.1). The interpreter is
// constructed from an already-parsed *ast.File plus its resolved *sema.Info —
// exactly the artifacts the Go transpiler back-end consumes — and runs the
// program's `func main`. It deliberately does NOT consume the Go backend's
// lowered output (Result->(T,error), Option->*T); it reads the typed AST and the
// native sema facts directly.
//
// Statement and expression evaluation is intentionally minimal here: an empty
// `main` body is a successful no-op. The evaluation stories (US-005 onward) fill
// in literals, operators, control flow, and the goal-specific runtime mechanics.

import (
	"errors"
	"fmt"

	"goal/internal/ast"
	"goal/internal/sema"
	"goal/internal/token"
)

// ErrNoMain reports that the program declares no top-level `func main`. It is a
// loud, named refusal rather than a silent successful run.
var ErrNoMain = errors.New("interp: no func main declared")

// Interp runs a parsed, sema-resolved goal program under interpretation. It
// holds the shared front-end artifacts (the AST + native semantic facts) and the
// root lexical scope.
type Interp struct {
	file *ast.File
	info *sema.Info
	root *Env
}

// New constructs an interpreter over the shared AST + sema front-end. file is the
// parsed program (from internal/parser) and info its resolved semantic facts
// (from internal/sema). Neither is the Go backend's lowered form.
func New(file *ast.File, info *sema.Info) *Interp {
	return &Interp{file: file, info: info, root: NewEnv()}
}

// Run executes the program's entry point: the top-level `func main` (a plain
// function with no receiver). It returns ErrNoMain when no such function is
// declared. An empty body is a successful no-op; richer statement evaluation is
// added by later stories. Run returns nil on successful completion.
func (ip *Interp) Run() error {
	main := ip.findMain()
	if main == nil {
		return ErrNoMain
	}
	scope := ip.root.NewChild()
	return ip.execBlock(main.Body, scope)
}

// findMain returns the top-level `func main` declaration, or nil if none exists.
// A method named main (Recv != nil) is not an entry point and is ignored.
func (ip *Interp) findMain() *ast.FuncDecl {
	if ip.file == nil {
		return nil
	}
	for _, d := range ip.file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Recv != nil || fn.Name == nil {
			continue
		}
		if fn.Name.Name == "main" {
			return fn
		}
	}
	return nil
}

// execBlock executes a block's statements in the given scope. This is the
// statement-dispatch seam later stories extend. US-005 wires expression
// statements (an expression evaluated for its effect/value); declarations,
// assignment, control flow, and the goal-specific forms arrive in US-006+.
func (ip *Interp) execBlock(block *ast.BlockStmt, scope *Env) error {
	if block == nil {
		return nil
	}
	for _, stmt := range block.List {
		if err := ip.execStmt(stmt, scope); err != nil {
			return err
		}
	}
	return nil
}

// execStmt executes a single statement. Unsupported statement forms are a
// descriptive, named refusal rather than a silent no-op; later stories add the
// remaining forms.
func (ip *Interp) execStmt(stmt ast.Stmt, scope *Env) error {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		_, err := ip.evalExpr(s.X, scope)
		return err
	case *ast.DeclStmt:
		return ip.execDecl(s, scope)
	case *ast.AssignStmt:
		return ip.execAssign(s, scope)
	case *ast.EmptyStmt:
		return nil
	default:
		return fmt.Errorf("interp: unsupported statement %T", stmt)
	}
}

// execDecl evaluates a const/var declaration statement, binding each declared
// name in the current scope. A var spec with no initializer binds the safe zero
// value for its declared type. A non-value declaration (import/type) in a
// statement position is a descriptive refusal.
func (ip *Interp) execDecl(s *ast.DeclStmt, scope *Env) error {
	gen, ok := s.Decl.(*ast.GenDecl)
	if !ok {
		return fmt.Errorf("interp: unsupported declaration %T", s.Decl)
	}
	if gen.Tok != token.VAR && gen.Tok != token.CONST {
		return fmt.Errorf("interp: unsupported %s declaration in statement position", gen.Tok)
	}
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			return fmt.Errorf("interp: unsupported spec %T in %s declaration", spec, gen.Tok)
		}
		switch {
		case len(vs.Values) == 0:
			// `var a, b int` — bind each name to its declared zero value.
			zero := zeroValue(vs.Type)
			for _, name := range vs.Names {
				scope.Define(name.Name, zero)
			}
		case len(vs.Values) == len(vs.Names):
			for i, name := range vs.Names {
				v, err := ip.evalExpr(vs.Values[i], scope)
				if err != nil {
					return err
				}
				scope.Define(name.Name, v)
			}
		default:
			return fmt.Errorf("interp: %s spec has %d names but %d values", gen.Tok, len(vs.Names), len(vs.Values))
		}
	}
	return nil
}

// execAssign evaluates an assignment statement: a short variable declaration
// (`:=`), a plain assignment (`=`), or a compound assignment (`+=`, `-=`, ...).
// All right-hand sides are evaluated BEFORE any binding so a parallel assignment
// like `a, b = b, a` swaps correctly. Short-var binds in the current scope; a
// plain or compound assignment updates the existing binding through the scope
// chain (and errors if the target is undeclared).
func (ip *Interp) execAssign(s *ast.AssignStmt, scope *Env) error {
	if len(s.Lhs) != len(s.Rhs) {
		return fmt.Errorf("interp: assignment has %d targets but %d values", len(s.Lhs), len(s.Rhs))
	}
	// Evaluate every RHS first (parallel-assignment order).
	vals := make([]Value, len(s.Rhs))
	for i, rhs := range s.Rhs {
		v, err := ip.evalExpr(rhs, scope)
		if err != nil {
			return err
		}
		vals[i] = v
	}
	for i, lhs := range s.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			return fmt.Errorf("interp: unsupported assignment target %T", lhs)
		}
		switch {
		case s.Tok == token.DEFINE:
			scope.Define(ident.Name, vals[i])
		case s.Tok == token.ASSIGN:
			if err := scope.Assign(ident.Name, vals[i]); err != nil {
				return err
			}
		default:
			op, ok := compoundBinOp(s.Tok)
			if !ok {
				return fmt.Errorf("interp: unsupported assignment operator %s", s.Tok)
			}
			cur, err := scope.Lookup(ident.Name)
			if err != nil {
				return err
			}
			res, err := applyBinary(op, cur, vals[i])
			if err != nil {
				return err
			}
			if err := scope.Assign(ident.Name, res); err != nil {
				return err
			}
		}
	}
	return nil
}

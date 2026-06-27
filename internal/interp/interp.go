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

// returnSignal is the non-local control signal raised by a `return` statement.
// It is threaded UP through execBlock/execStmt as an error so a return nested in
// arbitrarily deep blocks unwinds to the enclosing call boundary, which recovers
// it (errors.As) and reads its values. It is control flow, not a real error.
type returnSignal struct {
	vals []Value
}

// Error implements error so returnSignal can ride the (… error) return channel.
func (returnSignal) Error() string { return "interp: return outside function" }

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
//
// Every top-level plain function (no receiver, with a body) is registered into
// the root scope as a callable function value BEFORE evaluation begins, so a
// function is visible to its own body (recursion), to forward references, and to
// unit tests that evaluate a call against the root scope.
func New(file *ast.File, info *sema.Info) *Interp {
	ip := &Interp{file: file, info: info, root: NewEnv()}
	ip.registerFuncs()
	return ip
}

// registerFuncs binds every top-level plain function declaration in the root
// scope as a function value closing over that root scope. Methods (Recv != nil),
// nameless decls, and bodyless declarations are skipped.
func (ip *Interp) registerFuncs() {
	if ip.file == nil {
		return
	}
	for _, d := range ip.file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name == nil || fn.Body == nil {
			continue
		}
		ip.root.Define(fn.Name.Name, FuncDeclVal(fn, ip.root))
	}
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
	_, err := ip.callFunc(&FuncValue{Name: "main", Decl: main, Env: ip.root}, nil)
	return err
}

// callFunc invokes a function value with the already-evaluated argument values,
// returning its result values. It binds each positional parameter in a fresh
// child of the function's defining scope, runs the body, and recovers the
// returnSignal raised by a `return` (treating it as the normal result, not an
// error). A genuine error from the body propagates. An argument-count mismatch
// is a descriptive, named refusal.
func (ip *Interp) callFunc(fn *FuncValue, args []Value) ([]Value, error) {
	if fn == nil || fn.Decl == nil {
		name := ""
		if fn != nil {
			name = fn.Name
		}
		return nil, fmt.Errorf("interp: call of non-callable function value %q", name)
	}
	params := flattenParams(fn.Decl.Type)
	if len(args) != len(params) {
		return nil, fmt.Errorf("interp: %s expects %d args, got %d", fn.Name, len(params), len(args))
	}
	defScope := fn.Env
	if defScope == nil {
		defScope = ip.root
	}
	scope := defScope.NewChild()
	for i, name := range params {
		scope.Define(name, args[i])
	}
	if err := ip.execBlock(fn.Decl.Body, scope); err != nil {
		var ret returnSignal
		if errors.As(err, &ret) {
			return ret.vals, nil
		}
		return nil, err
	}
	// Fell off the end of the body without an explicit return: no values.
	return nil, nil
}

// flattenParams returns the positional parameter names of a function signature,
// flattening grouped declarations (`a, b int` -> ["a","b"]). An unnamed
// parameter contributes an empty name (its binding is unreachable by name but
// keeps the positional count correct).
func flattenParams(ft *ast.FuncType) []string {
	if ft == nil || ft.Params == nil {
		return nil
	}
	var names []string
	for _, f := range ft.Params.List {
		if len(f.Names) == 0 {
			names = append(names, "")
			continue
		}
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
	}
	return names
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
	case *ast.ReturnStmt:
		return ip.execReturn(s, scope)
	case *ast.IfStmt:
		return ip.execIf(s, scope)
	case *ast.EmptyStmt:
		return nil
	default:
		return fmt.Errorf("interp: unsupported statement %T", stmt)
	}
}

// execReturn evaluates a return statement's result expressions and raises a
// returnSignal carrying the values, unwinding to the enclosing call boundary. A
// bare `return` (no results) carries no values. A single result that is itself a
// multi-value call (`return f()`) is spread into the returned values.
func (ip *Interp) execReturn(s *ast.ReturnStmt, scope *Env) error {
	if len(s.Results) == 1 {
		if call, ok := s.Results[0].(*ast.CallExpr); ok {
			vals, err := ip.evalCallMulti(call, scope)
			if err != nil {
				return err
			}
			return returnSignal{vals: vals}
		}
	}
	vals := make([]Value, 0, len(s.Results))
	for _, r := range s.Results {
		v, err := ip.evalExpr(r, scope)
		if err != nil {
			return err
		}
		vals = append(vals, v)
	}
	return returnSignal{vals: vals}
}

// execIf evaluates an if statement: the optional init runs first in the if's own
// child scope, the bool condition selects the branch, and the taken branch runs
// in a further nested scope. A returnSignal raised by a branch propagates. A
// non-bool condition is a descriptive refusal.
func (ip *Interp) execIf(s *ast.IfStmt, scope *Env) error {
	ifScope := scope.NewChild()
	if s.Init != nil {
		if err := ip.execStmt(s.Init, ifScope); err != nil {
			return err
		}
	}
	cond, err := ip.evalExpr(s.Cond, ifScope)
	if err != nil {
		return err
	}
	if cond.Kind != KindBool {
		return fmt.Errorf("interp: if condition must be bool, got %s", cond.Kind)
	}
	if cond.Bool {
		return ip.execBlock(s.Body, ifScope.NewChild())
	}
	switch e := s.Else.(type) {
	case nil:
		return nil
	case *ast.BlockStmt:
		return ip.execBlock(e, ifScope.NewChild())
	case *ast.IfStmt:
		return ip.execIf(e, ifScope)
	default:
		return ip.execStmt(s.Else, ifScope.NewChild())
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
	// `q, r := f()` / `q, r = f()`: a single multi-value call spread across
	// several targets. The call's result count must match the target count.
	if len(s.Lhs) > 1 && len(s.Rhs) == 1 {
		if call, ok := s.Rhs[0].(*ast.CallExpr); ok {
			vals, err := ip.evalCallMulti(call, scope)
			if err != nil {
				return err
			}
			if len(vals) != len(s.Lhs) {
				return fmt.Errorf("interp: assignment has %d targets but call returned %d values", len(s.Lhs), len(vals))
			}
			return ip.bindTargets(s, vals, scope)
		}
	}
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
	return ip.bindTargets(s, vals, scope)
}

// bindTargets binds already-evaluated values to an assignment's targets using
// the statement's operator: `:=` defines in the current scope, `=` assigns
// through the scope chain, and a compound operator (`+=`, ...) reads-modifies-
// writes the existing binding. It is shared by the ordinary and the multi-value-
// call paths of execAssign. A non-identifier target is a descriptive refusal.
func (ip *Interp) bindTargets(s *ast.AssignStmt, vals []Value, scope *Env) error {
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

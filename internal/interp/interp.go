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

// breakSignal is the non-local control signal raised by a `break` statement. It
// rides the (… error) return channel exactly like returnSignal and is recovered
// at the nearest enclosing loop or switch boundary (execFor / execSwitch), which
// stops iterating. It is control flow, not a real error.
type breakSignal struct{}

// Error implements error so breakSignal can ride the (… error) return channel.
func (breakSignal) Error() string { return "interp: break outside loop or switch" }

// continueSignal is the non-local control signal raised by a `continue`
// statement. It is recovered at the nearest enclosing loop boundary (execFor),
// which advances to the post clause and the next iteration. A switch does NOT
// recover it — a continue inside a switch propagates to the enclosing loop.
type continueSignal struct{}

// Error implements error so continueSignal can ride the (… error) return channel.
func (continueSignal) Error() string { return "interp: continue outside loop" }

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
	case *ast.ForStmt:
		return ip.execFor(s, scope)
	case *ast.SwitchStmt:
		return ip.execSwitch(s, scope)
	case *ast.BranchStmt:
		return ip.execBranch(s, scope)
	case *ast.IncDecStmt:
		return ip.execIncDec(s, scope)
	case *ast.BlockStmt:
		// A bare nested block runs its statements in its own child scope, so a
		// variable declared inside it does not leak to the enclosing scope.
		return ip.execBlock(s, scope.NewChild())
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

// execFor evaluates a for statement: a three-clause loop (`for init; cond; post`),
// a condition-only loop (`for cond`), or an infinite loop (`for {}`). The optional
// init binds in the loop's own scope (so it persists across iterations); each
// iteration's body runs in a fresh child of that scope; the optional post runs
// after every iteration. A nil condition is always true; a non-bool condition is
// a descriptive refusal. A `break` ends the loop, a `continue` skips to the post
// clause, and a returnSignal (or a real error) propagates to the caller.
func (ip *Interp) execFor(s *ast.ForStmt, scope *Env) error {
	loopScope := scope.NewChild()
	if s.Init != nil {
		if err := ip.execStmt(s.Init, loopScope); err != nil {
			return err
		}
	}
	for {
		if s.Cond != nil {
			cond, err := ip.evalExpr(s.Cond, loopScope)
			if err != nil {
				return err
			}
			if cond.Kind != KindBool {
				return fmt.Errorf("interp: for condition must be bool, got %s", cond.Kind)
			}
			if !cond.Bool {
				return nil
			}
		}
		if err := ip.execBlock(s.Body, loopScope.NewChild()); err != nil {
			var brk breakSignal
			if errors.As(err, &brk) {
				return nil
			}
			var cont continueSignal
			if !errors.As(err, &cont) {
				// returnSignal or a genuine error: unwind out of the loop.
				return err
			}
			// continue: fall through to the post clause and next iteration.
		}
		if s.Post != nil {
			if err := ip.execStmt(s.Post, loopScope); err != nil {
				return err
			}
		}
	}
}

// execSwitch evaluates an expression switch. The optional init and the optional
// tag are evaluated in the switch's own scope. A tagged switch selects the first
// case clause with a list expression equal to the tag; a tagless switch selects
// the first case clause with a true (bool) list expression. A default clause runs
// when no case matches. The selected clause's body runs in a further child scope
// and does NOT fall through. A `break` ends the switch; a `continue` or
// returnSignal propagates to the enclosing loop / caller.
func (ip *Interp) execSwitch(s *ast.SwitchStmt, scope *Env) error {
	swScope := scope.NewChild()
	if s.Init != nil {
		if err := ip.execStmt(s.Init, swScope); err != nil {
			return err
		}
	}
	var tag Value
	hasTag := s.Tag != nil
	if hasTag {
		v, err := ip.evalExpr(s.Tag, swScope)
		if err != nil {
			return err
		}
		tag = v
	}

	var def *ast.CaseClause
	var selected *ast.CaseClause
	if s.Body != nil {
		for _, stmt := range s.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok {
				return fmt.Errorf("interp: unsupported switch clause %T", stmt)
			}
			if cc.List == nil {
				def = cc
				continue
			}
			matched, err := ip.caseMatches(cc, hasTag, tag, swScope)
			if err != nil {
				return err
			}
			if matched {
				selected = cc
				break
			}
		}
	}
	if selected == nil {
		selected = def
	}
	if selected == nil {
		return nil
	}
	if err := ip.execClauseBody(selected.Body, swScope.NewChild()); err != nil {
		var brk breakSignal
		if errors.As(err, &brk) {
			return nil
		}
		return err
	}
	return nil
}

// caseMatches reports whether a case clause is selected: for a tagged switch, any
// list expression equals the tag; for a tagless switch, any list expression
// evaluates to bool true (a non-bool tagless case is a descriptive refusal).
func (ip *Interp) caseMatches(cc *ast.CaseClause, hasTag bool, tag Value, scope *Env) (bool, error) {
	for _, expr := range cc.List {
		v, err := ip.evalExpr(expr, scope)
		if err != nil {
			return false, err
		}
		if hasTag {
			if v.Equal(tag) {
				return true, nil
			}
			continue
		}
		if v.Kind != KindBool {
			return false, fmt.Errorf("interp: tagless switch case must be bool, got %s", v.Kind)
		}
		if v.Bool {
			return true, nil
		}
	}
	return false, nil
}

// execClauseBody runs a switch clause's statement list in the given scope. A
// clause body is a flat statement slice (not a *ast.BlockStmt), so it cannot use
// execBlock directly.
func (ip *Interp) execClauseBody(body []ast.Stmt, scope *Env) error {
	for _, stmt := range body {
		if err := ip.execStmt(stmt, scope); err != nil {
			return err
		}
	}
	return nil
}

// execBranch evaluates a break or continue statement, raising the corresponding
// control signal. goto, fallthrough, and labelled branches are descriptive
// refusals (out of scope for this story).
func (ip *Interp) execBranch(s *ast.BranchStmt, scope *Env) error {
	switch s.Tok {
	case token.BREAK:
		return breakSignal{}
	case token.CONTINUE:
		return continueSignal{}
	default:
		return fmt.Errorf("interp: unsupported branch statement %s", s.Tok)
	}
}

// execIncDec evaluates an `x++` / `x--` statement by reading the current binding,
// adding or subtracting one (numeric only), and writing it back through the scope
// chain. A non-identifier target or a non-numeric operand is a descriptive
// refusal.
func (ip *Interp) execIncDec(s *ast.IncDecStmt, scope *Env) error {
	ident, ok := s.X.(*ast.Ident)
	if !ok {
		return fmt.Errorf("interp: unsupported %s target %T", s.Tok, s.X)
	}
	cur, err := scope.Lookup(ident.Name)
	if err != nil {
		return err
	}
	var one Value
	switch cur.Kind {
	case KindInt:
		one = IntVal(1)
	case KindFloat:
		one = FloatVal(1)
	default:
		return fmt.Errorf("interp: %s requires numeric operand, got %s", s.Tok, cur.Kind)
	}
	op := token.ADD
	if s.Tok == token.DEC {
		op = token.SUB
	}
	res, err := applyBinary(op, cur, one)
	if err != nil {
		return err
	}
	return scope.Assign(ident.Name, res)
}

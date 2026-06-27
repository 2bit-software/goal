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
	"sort"
	"strconv"
	"strings"

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

// panicSignal is the non-local control signal raised by the `panic` builtin. It
// rides the (… error) return channel like the other signals, but — unlike
// returnSignal/breakSignal/continueSignal — NO loop, switch, or call boundary
// recovers it: it propagates straight up to Run and out to the host, which is
// where it is observed (a "recovered panic" is one caught at that Go boundary).
// It carries the panic operand value.
type panicSignal struct {
	value Value
}

// Error implements error so panicSignal can ride the (… error) return channel.
func (p panicSignal) Error() string { return "interp: panic: " + p.value.String() }

// Interp runs a parsed, sema-resolved goal program under interpretation. It
// holds the shared front-end artifacts (the AST + native semantic facts) and the
// root lexical scope.
type Interp struct {
	file *ast.File
	info *sema.Info
	root *Env

	// methods indexes every method declaration by its receiver type name
	// (star-stripped) then method name, so a method call x.M(...) can resolve
	// the concrete declaration for the runtime type of x. Both value-receiver
	// and pointer-receiver methods are registered here.
	methods map[string]map[string]*ast.FuncDecl

	// imports maps each imported package's local name (the alias, or the last
	// element of the import path when unaliased) to its import path. It lets a
	// selector call `pkg.Sym(...)` be recognized as a host-package call and
	// routed to the host-function bridge (host.go).
	imports map[string]string
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
	ip := &Interp{file: file, info: info, root: NewEnv(), methods: map[string]map[string]*ast.FuncDecl{}, imports: map[string]string{}}
	ip.registerImports()
	ip.registerFuncs()
	ip.registerMethods()
	return ip
}

// registerImports records every import's local name -> import path, so a
// selector call whose receiver names an imported package is recognized as a
// host-package call. The local name is the explicit alias when present (an "_"
// blank or "." dot import contributes no usable name) and otherwise the last
// path element. A spec with no parseable path is skipped.
func (ip *Interp) registerImports() {
	if ip.file == nil {
		return
	}
	for _, spec := range ip.file.Imports {
		if spec == nil || spec.Path == nil {
			continue
		}
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path == "" {
			continue
		}
		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "" {
			name = path[strings.LastIndex(path, "/")+1:]
		}
		if name == "" || name == "_" || name == "." {
			continue
		}
		ip.imports[name] = path
	}
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

// registerMethods indexes every top-level method declaration (Recv != nil, with
// a name and a body) under its receiver type name (star-stripped, so a value
// receiver `(s T)` and a pointer receiver `(s *T)` register under the same T)
// and method name. A method whose receiver type cannot be resolved to a plain
// name is skipped.
func (ip *Interp) registerMethods() {
	if ip.file == nil {
		return
	}
	for _, d := range ip.file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil || fn.Body == nil {
			continue
		}
		typeName := recvTypeName(fn)
		if typeName == "" {
			continue
		}
		byName := ip.methods[typeName]
		if byName == nil {
			byName = map[string]*ast.FuncDecl{}
			ip.methods[typeName] = byName
		}
		byName[fn.Name.Name] = fn
	}
}

// recvTypeName returns the star-stripped receiver type name of a method
// declaration (`(s T)` and `(s *T)` both yield "T"), or "" if the receiver is
// absent or not a plain (optionally pointer-to) named type.
func recvTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	t := fn.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// recvName returns the receiver parameter name of a method declaration (the `s`
// in `(s T)`), or "" if the receiver is unnamed.
func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 || len(fn.Recv.List[0].Names) == 0 {
		return ""
	}
	return fn.Recv.List[0].Names[0].Name
}

// recvIsPointer reports whether a method has a pointer receiver (`(s *T)`). A
// pointer receiver shares the caller's pointer-backed struct, so field mutations
// are visible to the caller; a value receiver operates on a copy.
func recvIsPointer(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}
	_, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	return ok
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

// callMethod invokes a method declaration with an already-evaluated receiver
// value and argument values. The receiver binds in a fresh child of the root
// (package) scope under the method's receiver name: a pointer receiver shares
// the caller's pointer-backed struct (so field mutations are visible to the
// caller), while a value receiver binds a shallow COPY (so its mutations do not
// leak, matching Go value semantics). It then binds parameters, runs the body,
// and recovers the returnSignal like callFunc; a panicSignal (or a genuine
// error) propagates.
func (ip *Interp) callMethod(decl *ast.FuncDecl, recv Value, args []Value) ([]Value, error) {
	params := flattenParams(decl.Type)
	if len(args) != len(params) {
		return nil, fmt.Errorf("interp: %s.%s expects %d args, got %d", recvTypeName(decl), decl.Name.Name, len(params), len(args))
	}
	if !recvIsPointer(decl) {
		recv = copyStructValue(recv)
	}
	scope := ip.root.NewChild()
	if name := recvName(decl); name != "" && name != "_" {
		scope.Define(name, recv)
	}
	for i, name := range params {
		scope.Define(name, args[i])
	}
	if err := ip.execBlock(decl.Body, scope); err != nil {
		var ret returnSignal
		if errors.As(err, &ret) {
			return ret.vals, nil
		}
		return nil, err
	}
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
		// A statement-position match parses to an ExprStmt wrapping a MatchExpr;
		// it is intercepted here (NOT in evalExpr — value-position match is a
		// later story) and dispatched by variant tag.
		if m, ok := s.X.(*ast.MatchExpr); ok {
			return ip.execMatch(m, scope)
		}
		// A call in statement position may produce zero or several values (e.g. a
		// void method or function call); evaluate it through the multi-value path
		// and discard the results rather than forcing a single-value context.
		if c, ok := s.X.(*ast.CallExpr); ok {
			_, err := ip.evalCallMulti(c, scope)
			return err
		}
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
	case *ast.RangeStmt:
		return ip.execRange(s, scope)
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

// execMatch evaluates a statement-position match: it evaluates the scrutinee to
// a tagged-union value, dispatches on its variant tag to the matching arm, binds
// the matched payload into a fresh child scope, and runs the selected arm. A
// `_` rest arm is the explicit catch-all. A proven-exhaustive match (sema has
// verified every variant is covered) always selects a variant or rest arm; the
// defensive default — reached only if a value's tag matches no arm, which a
// sema-checked program cannot produce — is a loud `unreachable` panic, never a
// silent fall-through (the "erase the guarantee, panic-not-silent" stance).
func (ip *Interp) execMatch(m *ast.MatchExpr, scope *Env) error {
	subj, err := ip.evalExpr(m.Subject, scope)
	if err != nil {
		return err
	}
	if subj.Kind != KindVariant || subj.Variant == nil {
		return fmt.Errorf("interp: match subject must be a variant, got %s", subj.Kind)
	}
	arm, vp := selectMatchArm(m, subj)
	if arm == nil {
		return unreachableMatch(subj)
	}
	return ip.execArm(arm, vp, subj, scope)
}

// selectMatchArm chooses the arm a variant subject dispatches to: the variant arm
// whose pattern tag equals the subject's tag (returned with its VariantPattern),
// else a `_` rest arm (returned with a nil VariantPattern), else (nil, nil) —
// signalling no arm matched. It is shared by statement-position (execMatch) and
// value-position (evalMatch) match so dispatch stays in lock-step.
func selectMatchArm(m *ast.MatchExpr, subj Value) (*ast.MatchArm, *ast.VariantPattern) {
	var restArm *ast.MatchArm
	for _, arm := range m.Arms {
		switch p := arm.Pattern.(type) {
		case *ast.VariantPattern:
			if p.Variant != nil && p.Variant.Name == subj.Variant.Tag {
				return arm, p
			}
		case *ast.RestPattern:
			restArm = arm
		}
	}
	return restArm, nil
}

// unreachableMatch is the loud, defensive default raised when a variant's tag
// matches no arm — a state a sema-proven-exhaustive program cannot reach. It is a
// panic, never a silent fall-through ("erase the guarantee, panic-not-silent").
func unreachableMatch(subj Value) error {
	return panicSignal{value: StrVal(fmt.Sprintf(
		"unreachable: non-exhaustive match on %s (compiler invariant violated)", subj.Variant.TypeID))}
}

// armScopeFor opens a fresh child scope for a match arm and binds the arm's
// payload binding (when its VariantPattern names one) to the whole variant value
// — so payload fields read through it as `binding.field` (evalSelector reads
// variant fields). A rest arm (vp == nil) binds nothing. Shared by statement- and
// value-position match.
func armScopeFor(vp *ast.VariantPattern, subj Value, scope *Env) *Env {
	armScope := scope.NewChild()
	if vp != nil && vp.Binding != nil {
		armScope.Define(vp.Binding.Name, subj)
	}
	return armScope
}

// execArm binds the arm's payload (via armScopeFor) then runs the arm body.
func (ip *Interp) execArm(arm *ast.MatchArm, vp *ast.VariantPattern, subj Value, scope *Env) error {
	return ip.execArmBody(arm.Body, armScopeFor(vp, subj, scope))
}

// execArmBody runs a match arm body, which is a generic ast.Node: a statement
// (incl. a block, a `return`, or an assignment) executes via execStmt, and an
// expression (typically a void call) is evaluated for its effect. A non-local
// control signal (return/break/continue/panic) raised by the body propagates.
func (ip *Interp) execArmBody(body ast.Node, scope *Env) error {
	switch b := body.(type) {
	case nil:
		return nil
	case ast.Stmt:
		return ip.execStmt(b, scope)
	case *ast.CallExpr:
		_, err := ip.evalCallMulti(b, scope)
		return err
	case ast.Expr:
		_, err := ip.evalExpr(b, scope)
		return err
	default:
		return fmt.Errorf("interp: unsupported match arm body %T", body)
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
		if err := ip.assignTarget(lhs, vals[i], s.Tok, scope); err != nil {
			return err
		}
	}
	return nil
}

// assignTarget binds a single already-evaluated value to one assignment target
// using the statement operator. An identifier target uses the lexical scope
// (`:=` defines, `=` assigns through the chain, a compound operator
// read-modify-writes the binding). An index target writes a slice element or a
// map entry, and a selector target writes a struct field — each through the
// collection's shared/pointer-backed identity, with a compound operator applied
// to the current element/field/entry. A non-identifier compound on a missing map
// key, or any other target form, is a descriptive refusal.
func (ip *Interp) assignTarget(lhs ast.Expr, v Value, tok token.Kind, scope *Env) error {
	switch t := lhs.(type) {
	case *ast.Ident:
		switch {
		case tok == token.DEFINE:
			scope.Define(t.Name, v)
			return nil
		case tok == token.ASSIGN:
			return scope.Assign(t.Name, v)
		default:
			cur, err := scope.Lookup(t.Name)
			if err != nil {
				return err
			}
			res, err := ip.compoundApply(tok, cur, v)
			if err != nil {
				return err
			}
			return scope.Assign(t.Name, res)
		}
	case *ast.IndexExpr:
		return ip.assignIndex(t, v, tok, scope)
	case *ast.SelectorExpr:
		return ip.assignField(t, v, tok, scope)
	default:
		return fmt.Errorf("interp: unsupported assignment target %T", lhs)
	}
}

// compoundApply applies a compound-assignment operator (`+=`, ...) to the current
// value and the right-hand value, reusing the binary-operator evaluator. An
// unsupported operator is a descriptive refusal.
func (ip *Interp) compoundApply(tok token.Kind, cur, rhs Value) (Value, error) {
	op, ok := compoundBinOp(tok)
	if !ok {
		return Value{}, fmt.Errorf("interp: unsupported assignment operator %s", tok)
	}
	return applyBinary(op, cur, rhs)
}

// assignIndex writes an indexed assignment target: a slice element (bounds-
// checked) or a map entry (insert or update). A compound operator reads the
// current element/entry first; a compound on an absent map key is a descriptive
// refusal. A `:=` index target is rejected (you cannot declare an index target).
func (ip *Interp) assignIndex(t *ast.IndexExpr, v Value, tok token.Kind, scope *Env) error {
	if tok == token.DEFINE {
		return fmt.Errorf("interp: cannot use := with an index target")
	}
	recv, err := ip.evalExpr(t.X, scope)
	if err != nil {
		return err
	}
	idx, err := ip.evalExpr(t.Index, scope)
	if err != nil {
		return err
	}
	switch recv.Kind {
	case KindSlice:
		if idx.Kind != KindInt {
			return fmt.Errorf("interp: slice index must be int, got %s", idx.Kind)
		}
		if idx.Int < 0 || idx.Int >= int64(len(recv.Slice)) {
			return fmt.Errorf("interp: slice index %d out of range (len %d)", idx.Int, len(recv.Slice))
		}
		if tok != token.ASSIGN {
			res, err := ip.compoundApply(tok, recv.Slice[idx.Int], v)
			if err != nil {
				return err
			}
			v = res
		}
		recv.Slice[idx.Int] = v
		return nil
	case KindMap:
		if recv.Map == nil {
			return fmt.Errorf("interp: assignment to entry in nil map")
		}
		key, err := mapKeyString(idx)
		if err != nil {
			return err
		}
		if tok != token.ASSIGN {
			cur, ok := recv.Map.Entries[key]
			if !ok {
				return fmt.Errorf("interp: compound assignment to absent map key %q", key)
			}
			res, err := ip.compoundApply(tok, cur, v)
			if err != nil {
				return err
			}
			v = res
		}
		recv.Map.Entries[key] = v
		return nil
	default:
		return fmt.Errorf("interp: cannot index-assign %s", recv.Kind)
	}
}

// assignField writes a struct field assignment target (`x.field = v`). The
// receiver must evaluate to a struct value (which is pointer-backed, so the write
// is visible through every binding that shares it). A compound operator reads the
// current field first. A non-struct receiver or a `:=` field target is a
// descriptive refusal.
func (ip *Interp) assignField(t *ast.SelectorExpr, v Value, tok token.Kind, scope *Env) error {
	if tok == token.DEFINE {
		return fmt.Errorf("interp: cannot use := with a field target")
	}
	recv, err := ip.evalExpr(t.X, scope)
	if err != nil {
		return err
	}
	if recv.Kind != KindStruct || recv.Struct == nil {
		return fmt.Errorf("interp: cannot assign field %s on %s", t.Sel.Name, recv.Kind)
	}
	if tok != token.ASSIGN {
		cur, ok := recv.Struct.Fields[t.Sel.Name]
		if !ok {
			return fmt.Errorf("interp: %s has no field %s", recv.Struct.TypeID, t.Sel.Name)
		}
		res, err := ip.compoundApply(tok, cur, v)
		if err != nil {
			return err
		}
		v = res
	}
	recv.Struct.Fields[t.Sel.Name] = v
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

// execRange evaluates a for-range statement over a slice or a map. A slice
// iterates in ascending index order (key = index, value = element); a map
// iterates its entries with keys sorted for deterministic ordering (key = the
// string key, value = the entry). The key/value targets are bound each iteration
// in a fresh child scope — `:=` defines fresh loop variables and `=` assigns
// existing ones; a `_` or omitted target is skipped. A `break` ends the loop, a
// `continue` advances to the next entry, and a returnSignal (or a real error)
// propagates to the caller. Ranging any other kind is a descriptive refusal.
func (ip *Interp) execRange(s *ast.RangeStmt, scope *Env) error {
	subject, err := ip.evalExpr(s.X, scope)
	if err != nil {
		return err
	}
	rangeScope := scope.NewChild()

	iterate := func(key, val Value) (stop bool, err error) {
		iterScope := rangeScope.NewChild()
		if err := ip.rangeBind(s.Key, key, s.Tok, iterScope); err != nil {
			return true, err
		}
		if err := ip.rangeBind(s.Value, val, s.Tok, iterScope); err != nil {
			return true, err
		}
		if err := ip.execBlock(s.Body, iterScope); err != nil {
			var brk breakSignal
			if errors.As(err, &brk) {
				return true, nil
			}
			var cont continueSignal
			if errors.As(err, &cont) {
				return false, nil
			}
			return true, err
		}
		return false, nil
	}

	switch subject.Kind {
	case KindSlice:
		for i, elem := range subject.Slice {
			stop, err := iterate(IntVal(int64(i)), elem)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}
		return nil
	case KindMap:
		if subject.Map == nil {
			return nil
		}
		keys := make([]string, 0, len(subject.Map.Entries))
		for k := range subject.Map.Entries {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			stop, err := iterate(StrVal(k), subject.Map.Entries[k])
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}
		return nil
	default:
		return fmt.Errorf("interp: cannot range over %s", subject.Kind)
	}
}

// rangeBind binds one range loop variable (the key or the value). A nil target
// or the blank identifier `_` is skipped; `:=` defines in the iteration scope and
// `=` assigns through the scope chain. A non-identifier target is a descriptive
// refusal.
func (ip *Interp) rangeBind(target ast.Expr, v Value, tok token.Kind, scope *Env) error {
	if target == nil {
		return nil
	}
	ident, ok := target.(*ast.Ident)
	if !ok {
		return fmt.Errorf("interp: unsupported range target %T", target)
	}
	if ident.Name == "_" {
		return nil
	}
	if tok == token.ASSIGN {
		return scope.Assign(ident.Name, v)
	}
	scope.Define(ident.Name, v)
	return nil
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

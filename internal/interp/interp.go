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

	"goal/internal/ast"
	"goal/internal/sema"
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

// execBlock executes a block's statements in the given scope. In this story the
// body of the trivial entry point is empty, so this is a no-op; US-005 onward
// dispatch each statement form here.
func (ip *Interp) execBlock(block *ast.BlockStmt, scope *Env) error {
	if block == nil {
		return nil
	}
	_ = scope
	for range block.List {
		// Statement evaluation is added by later stories (US-005+).
	}
	return nil
}

package interp

// This file implements the host-function bridge (US-011): a registry that
// resolves a curated set of Go standard-library calls to native Go
// implementations so corpus programs that touch the stdlib can run under
// interpretation. A package-qualified call `pkg.Sym(...)` whose receiver names
// an imported package is routed here; a symbol with no registered shim is an
// explicit, LOCATED, NAMED refusal rather than a silent nil — an unshimmed
// stdlib symbol must fail loudly and by name.
//
// Host effects are capability-mediated (US-023): the effectful fmt.Println is
// NOT a pure registry shim — it needs the interpreter's configurable sink and
// capability set — so evalHostCall routes it through ip.emitStdout (gated by
// cap.Stdout). The pure value-producing shims (Sprintf/Sprint/Errorf/errors.New)
// stay in the package-level registry below.

import (
	"errors"
	"fmt"
	"io"

	"goal/internal/ast"
)

// hostFunc is a native implementation of an imported stdlib function. It takes
// the already-evaluated argument values and returns the call's result values
// (empty for a void effect), or a descriptive error.
type hostFunc func(args []Value) ([]Value, error)

// hostFuncs is the host-function registry, keyed by "<import-path>.<Symbol>"
// (e.g. "fmt.Sprintf"). Keying on the real import path — not the local alias —
// means an aliased import (`f "fmt"`) still resolves f.Sprintf to "fmt.Sprintf".
var hostFuncs = map[string]hostFunc{
	"fmt.Sprintf": func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("interp: fmt.Sprintf expects at least 1 argument, got %d", len(args))
		}
		if args[0].Kind != KindString {
			return nil, fmt.Errorf("interp: fmt.Sprintf format must be string, got %s", args[0].Kind)
		}
		return []Value{StrVal(fmt.Sprintf(args[0].Str, goArgs(args[1:])...))}, nil
	},
	"fmt.Sprint": func(args []Value) ([]Value, error) {
		return []Value{StrVal(fmt.Sprint(goArgs(args)...))}, nil
	},
	// fmt.Println is an EFFECT, not a pure shim: it is intercepted in
	// evalHostCall and routed through ip.emitStdout (cap.Stdout-gated), so it is
	// deliberately absent from this pure registry.
	"fmt.Errorf": func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("interp: fmt.Errorf expects at least 1 argument, got %d", len(args))
		}
		if args[0].Kind != KindString {
			return nil, fmt.Errorf("interp: fmt.Errorf format must be string, got %s", args[0].Kind)
		}
		return []Value{errVal(fmt.Errorf(args[0].Str, goArgs(args[1:])...).Error())}, nil
	},
	"errors.New": func(args []Value) ([]Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("interp: errors.New expects 1 argument, got %d", len(args))
		}
		if args[0].Kind != KindString {
			return nil, fmt.Errorf("interp: errors.New argument must be string, got %s", args[0].Kind)
		}
		return []Value{errVal(args[0].Str)}, nil
	},
}

// errVal constructs the interpreter's runtime error value: a struct with the
// reserved TypeID "error" carrying its message. Result/Option get their own
// tagged-union encodings in later stories (US-015/US-016); this lightweight
// error struct is what fmt.Errorf and errors.New produce in the meantime.
func errVal(msg string) Value {
	return StructVal("error", map[string]Value{"message": StrVal(msg)})
}

// isErrorValue reports whether v is the interpreter's error struct.
func isErrorValue(v Value) bool {
	return v.Kind == KindStruct && v.Struct != nil && v.Struct.TypeID == "error"
}

// goArg converts a runtime Value into a Go value suitable for fmt formatting.
// Primitives pass through to their Go scalar; the error struct becomes a real
// Go error (so %v/%s render the message and %w wraps correctly); any other
// composite renders via Value.String().
func goArg(v Value) any {
	switch v.Kind {
	case KindNil:
		return nil
	case KindInt:
		return v.Int
	case KindFloat:
		return v.Float
	case KindString:
		return v.Str
	case KindBool:
		return v.Bool
	case KindStruct:
		if isErrorValue(v) {
			msg, _ := v.Struct.Fields["message"]
			return errors.New(msg.Str)
		}
		return v.String()
	default:
		return v.String()
	}
}

// goArgs converts a slice of runtime Values to Go formatting arguments.
func goArgs(vs []Value) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = goArg(v)
	}
	return out
}

// evalHostCall resolves and invokes a host-package call `pkg.Sym(...)`. The
// receiver identifier has already been recognized as an imported package by
// evalCallMulti. It evaluates the arguments, looks the symbol up in hostFuncs
// by "<import-path>.<Symbol>", and invokes the shim. An unregistered symbol is
// a LOCATED, NAMED refusal — it names the missing pkg.Symbol and its source
// position — never a silent nil.
func (ip *Interp) evalHostCall(sel *ast.SelectorExpr, call *ast.CallExpr, scope *Env) ([]Value, error) {
	pkg := sel.X.(*ast.Ident)
	path := ip.imports[pkg.Name]
	key := path + "." + sel.Sel.Name

	args := make([]Value, len(call.Args))
	for i, a := range call.Args {
		v, err := ip.evalExpr(a, scope)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}

	// Effectful host calls are mediated by the capability model rather than the
	// pure registry: fmt.Println is a stdout EFFECT routed through emitStdout
	// (gated by cap.Stdout) so it reaches the interpreter's configurable sink.
	if key == "fmt.Println" {
		return nil, ip.emitStdout(func(w io.Writer) error {
			_, err := fmt.Fprintln(w, goArgs(args)...)
			return err
		})
	}

	fn, ok := hostFuncs[key]
	if !ok {
		return nil, fmt.Errorf("interp: %s: unresolved imported call %s (no host function registered)", sel.Pos().String(), key)
	}
	return fn(args)
}

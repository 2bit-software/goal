package interp

// This file implements expression evaluation (US-005): the interpreter turns an
// AST expression node into a runtime Value with Go semantics. It covers the
// primitive literals (int/float/string/bool) and the arithmetic, comparison,
// logical, and unary operators. Logical && / || short-circuit — the right
// operand is evaluated only when the left operand does not already decide the
// result.
//
// Identifiers (other than the predeclared true/false), calls, and composite
// forms are deferred to later evaluation stories (US-006 onward); encountering
// one yields a descriptive, located error rather than a silent nil.

import (
	"fmt"
	"strconv"

	"goal/internal/ast"
	"goal/internal/token"
)

// evalExpr evaluates an expression node in the given scope, returning its
// runtime Value. An unsupported node, a divide/remainder by zero, or an
// operator applied to mismatched operand kinds yields a descriptive error.
func (ip *Interp) evalExpr(expr ast.Expr, scope *Env) (Value, error) {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return ip.evalExpr(e.X, scope)
	case *ast.BasicLit:
		return evalBasicLit(e)
	case *ast.Ident:
		switch e.Name {
		case "true":
			return BoolVal(true), nil
		case "false":
			return BoolVal(false), nil
		default:
			// Any other identifier resolves to its current binding in the
			// lexical scope chain; an undefined name surfaces the located
			// *NotFoundError rather than a silent zero Value.
			return scope.Lookup(e.Name)
		}
	case *ast.BinaryExpr:
		return ip.evalBinary(e, scope)
	case *ast.UnaryExpr:
		return ip.evalUnary(e, scope)
	case *ast.CallExpr:
		return ip.evalCall(e, scope)
	default:
		return Value{}, fmt.Errorf("interp: unsupported expression %T", expr)
	}
}

// evalCall evaluates a call in single-value position: it requires the callee to
// produce exactly one result. A call producing zero or several values in a
// single-value context is a descriptive refusal (a multi-value call is only
// legal as the sole right-hand side of a multi-assignment or a return).
func (ip *Interp) evalCall(call *ast.CallExpr, scope *Env) (Value, error) {
	vals, err := ip.evalCallMulti(call, scope)
	if err != nil {
		return Value{}, err
	}
	if len(vals) != 1 {
		return Value{}, fmt.Errorf("interp: multi-value call used in single-value context (%d values)", len(vals))
	}
	return vals[0], nil
}

// evalCallMulti evaluates a call and returns all of the callee's result values.
// It resolves the callee to a function value, evaluates each argument in order,
// and dispatches to callFunc (which binds parameters and runs the body). A
// non-function callee is a descriptive refusal; an undefined callee surfaces the
// located *NotFoundError from the scope lookup.
func (ip *Interp) evalCallMulti(call *ast.CallExpr, scope *Env) ([]Value, error) {
	callee, err := ip.evalExpr(call.Fun, scope)
	if err != nil {
		return nil, err
	}
	if callee.Kind != KindFunc || callee.Func == nil {
		return nil, fmt.Errorf("interp: cannot call %s", callee.Kind)
	}
	args := make([]Value, len(call.Args))
	for i, a := range call.Args {
		v, err := ip.evalExpr(a, scope)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	return ip.callFunc(callee.Func, args)
}

// evalBasicLit decodes a basic literal token into a runtime Value. Integer
// literals decode in base 0 (so 0x/0o/0b forms are accepted), characters decode
// to their rune value as an int, and strings/chars are unquoted.
func evalBasicLit(lit *ast.BasicLit) (Value, error) {
	switch lit.Kind {
	case token.INT:
		n, err := strconv.ParseInt(lit.Value, 0, 64)
		if err != nil {
			return Value{}, fmt.Errorf("interp: invalid int literal %q: %w", lit.Value, err)
		}
		return IntVal(n), nil
	case token.FLOAT:
		f, err := strconv.ParseFloat(lit.Value, 64)
		if err != nil {
			return Value{}, fmt.Errorf("interp: invalid float literal %q: %w", lit.Value, err)
		}
		return FloatVal(f), nil
	case token.STRING:
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return Value{}, fmt.Errorf("interp: invalid string literal %q: %w", lit.Value, err)
		}
		return StrVal(s), nil
	case token.CHAR:
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return Value{}, fmt.Errorf("interp: invalid char literal %q: %w", lit.Value, err)
		}
		r := []rune(s)
		if len(r) != 1 {
			return Value{}, fmt.Errorf("interp: invalid char literal %q", lit.Value)
		}
		return IntVal(int64(r[0])), nil
	default:
		return Value{}, fmt.Errorf("interp: unsupported literal kind %s", lit.Kind)
	}
}

// evalBinary evaluates a binary expression. The logical operators short-circuit:
// the right operand is evaluated only when the left does not already decide the
// result. All other operators evaluate both operands first.
func (ip *Interp) evalBinary(b *ast.BinaryExpr, scope *Env) (Value, error) {
	// Logical operators short-circuit on the left operand.
	if b.Op == token.LAND || b.Op == token.LOR {
		left, err := ip.evalExpr(b.X, scope)
		if err != nil {
			return Value{}, err
		}
		if left.Kind != KindBool {
			return Value{}, fmt.Errorf("interp: operator %s requires bool, got %s", b.Op, left.Kind)
		}
		// && short-circuits when left is false; || when left is true.
		if b.Op == token.LAND && !left.Bool {
			return BoolVal(false), nil
		}
		if b.Op == token.LOR && left.Bool {
			return BoolVal(true), nil
		}
		right, err := ip.evalExpr(b.Y, scope)
		if err != nil {
			return Value{}, err
		}
		if right.Kind != KindBool {
			return Value{}, fmt.Errorf("interp: operator %s requires bool, got %s", b.Op, right.Kind)
		}
		return BoolVal(right.Bool), nil
	}

	left, err := ip.evalExpr(b.X, scope)
	if err != nil {
		return Value{}, err
	}
	right, err := ip.evalExpr(b.Y, scope)
	if err != nil {
		return Value{}, err
	}
	return applyBinary(b.Op, left, right)
}

// applyBinary applies a non-short-circuiting binary operator to two already
// evaluated operands.
func applyBinary(op token.Kind, left, right Value) (Value, error) {
	switch op {
	case token.EQL:
		return BoolVal(left.Equal(right)), nil
	case token.NEQ:
		return BoolVal(!left.Equal(right)), nil
	}

	if left.Kind != right.Kind {
		return Value{}, fmt.Errorf("interp: operator %s on mismatched kinds %s and %s", op, left.Kind, right.Kind)
	}

	switch left.Kind {
	case KindInt:
		return intBinary(op, left.Int, right.Int)
	case KindFloat:
		return floatBinary(op, left.Float, right.Float)
	case KindString:
		return stringBinary(op, left.Str, right.Str)
	default:
		return Value{}, fmt.Errorf("interp: operator %s not supported on %s", op, left.Kind)
	}
}

func intBinary(op token.Kind, a, b int64) (Value, error) {
	switch op {
	case token.ADD:
		return IntVal(a + b), nil
	case token.SUB:
		return IntVal(a - b), nil
	case token.MUL:
		return IntVal(a * b), nil
	case token.QUO:
		if b == 0 {
			return Value{}, fmt.Errorf("interp: integer divide by zero")
		}
		return IntVal(a / b), nil
	case token.REM:
		if b == 0 {
			return Value{}, fmt.Errorf("interp: integer divide by zero")
		}
		return IntVal(a % b), nil
	case token.LSS:
		return BoolVal(a < b), nil
	case token.LEQ:
		return BoolVal(a <= b), nil
	case token.GTR:
		return BoolVal(a > b), nil
	case token.GEQ:
		return BoolVal(a >= b), nil
	default:
		return Value{}, fmt.Errorf("interp: operator %s not supported on int", op)
	}
}

func floatBinary(op token.Kind, a, b float64) (Value, error) {
	switch op {
	case token.ADD:
		return FloatVal(a + b), nil
	case token.SUB:
		return FloatVal(a - b), nil
	case token.MUL:
		return FloatVal(a * b), nil
	case token.QUO:
		return FloatVal(a / b), nil
	case token.LSS:
		return BoolVal(a < b), nil
	case token.LEQ:
		return BoolVal(a <= b), nil
	case token.GTR:
		return BoolVal(a > b), nil
	case token.GEQ:
		return BoolVal(a >= b), nil
	default:
		return Value{}, fmt.Errorf("interp: operator %s not supported on float", op)
	}
}

func stringBinary(op token.Kind, a, b string) (Value, error) {
	switch op {
	case token.ADD:
		return StrVal(a + b), nil
	case token.LSS:
		return BoolVal(a < b), nil
	case token.LEQ:
		return BoolVal(a <= b), nil
	case token.GTR:
		return BoolVal(a > b), nil
	case token.GEQ:
		return BoolVal(a >= b), nil
	default:
		return Value{}, fmt.Errorf("interp: operator %s not supported on string", op)
	}
}

// evalUnary evaluates a unary expression: numeric negation (-), boolean
// negation (!), and a no-op unary plus (+).
func (ip *Interp) evalUnary(u *ast.UnaryExpr, scope *Env) (Value, error) {
	x, err := ip.evalExpr(u.X, scope)
	if err != nil {
		return Value{}, err
	}
	switch u.Op {
	case token.ADD:
		switch x.Kind {
		case KindInt, KindFloat:
			return x, nil
		default:
			return Value{}, fmt.Errorf("interp: unary + requires numeric, got %s", x.Kind)
		}
	case token.SUB:
		switch x.Kind {
		case KindInt:
			return IntVal(-x.Int), nil
		case KindFloat:
			return FloatVal(-x.Float), nil
		default:
			return Value{}, fmt.Errorf("interp: unary - requires numeric, got %s", x.Kind)
		}
	case token.NOT:
		if x.Kind != KindBool {
			return Value{}, fmt.Errorf("interp: unary ! requires bool, got %s", x.Kind)
		}
		return BoolVal(!x.Bool), nil
	default:
		return Value{}, fmt.Errorf("interp: unsupported unary operator %s", u.Op)
	}
}

// zeroValue returns the safe runtime zero for a declared type expression, used
// to initialize a `var x T` with no explicit initializer. Numeric types zero to
// 0, string to "", and bool to false; any other (including composite or unknown)
// type zeroes to nil. Composite zero values are refined by later eval stories
// (US-009/US-010).
func zeroValue(typeExpr ast.Expr) Value {
	id, ok := typeExpr.(*ast.Ident)
	if !ok {
		return NilVal()
	}
	switch id.Name {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune":
		return IntVal(0)
	case "float32", "float64":
		return FloatVal(0)
	case "string":
		return StrVal("")
	case "bool":
		return BoolVal(false)
	default:
		return NilVal()
	}
}

// compoundBinOp maps a compound-assignment token to the binary operator it
// applies (so `x += y` reuses the `+` evaluator). ok is false for any token
// that is not a supported arithmetic compound assignment; the caller turns that
// into a descriptive error rather than guessing.
func compoundBinOp(tok token.Kind) (token.Kind, bool) {
	switch tok {
	case token.ADD_ASSIGN:
		return token.ADD, true
	case token.SUB_ASSIGN:
		return token.SUB, true
	case token.MUL_ASSIGN:
		return token.MUL, true
	case token.QUO_ASSIGN:
		return token.QUO, true
	case token.REM_ASSIGN:
		return token.REM, true
	default:
		return token.ILLEGAL, false
	}
}

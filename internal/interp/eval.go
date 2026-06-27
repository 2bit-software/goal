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
			return Value{}, fmt.Errorf("interp: identifier %q not supported yet (US-006/US-007)", e.Name)
		}
	case *ast.BinaryExpr:
		return ip.evalBinary(e, scope)
	case *ast.UnaryExpr:
		return ip.evalUnary(e, scope)
	default:
		return Value{}, fmt.Errorf("interp: unsupported expression %T", expr)
	}
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

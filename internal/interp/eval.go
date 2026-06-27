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
	"goal/internal/sema"
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
	case *ast.CompositeLit:
		return ip.evalCompositeLit(e, scope)
	case *ast.SelectorExpr:
		return ip.evalSelector(e, scope)
	case *ast.VariantLit:
		return ip.evalVariantLit(e, scope)
	case *ast.IndexExpr:
		return ip.evalIndex(e, scope)
	case *ast.MatchExpr:
		// A value-position match (`return match`, `x := match`, `var x = match`)
		// yields the selected arm's value. A statement-position match is
		// intercepted earlier in execStmt and never reaches here.
		return ip.evalMatch(e, scope)
	case *ast.UnwrapExpr:
		// Postfix `?`: unwrap-and-continue on Ok/Some, or non-local early return
		// on Err/None. Reached from every statement position (`x := expr?`,
		// `_ := expr?`, bare `expr?`) through evalExpr's existing callers.
		return ip.evalUnwrap(e, scope)
	default:
		return Value{}, fmt.Errorf("interp: unsupported expression %T", expr)
	}
}

// evalMatch evaluates a value-position match: it evaluates the scrutinee to a
// tagged-union value, dispatches on its variant tag to the matching arm (sharing
// selectMatchArm/armScopeFor with statement-position match so dispatch stays
// uniform), binds the matched payload into a fresh child scope, evaluates that
// arm's body as an expression, and returns its value. A non-variant scrutinee is
// a descriptive refusal. A tag matching no arm — unreachable in a
// sema-proven-exhaustive program — raises a loud `unreachable` panic rather than
// producing a silent zero value.
func (ip *Interp) evalMatch(m *ast.MatchExpr, scope *Env) (Value, error) {
	subj, err := ip.evalExpr(m.Subject, scope)
	if err != nil {
		return Value{}, err
	}
	if subj.Kind != KindVariant || subj.Variant == nil {
		return Value{}, fmt.Errorf("interp: match subject must be a variant, got %s", subj.Kind)
	}
	arm, vp := selectMatchArm(m, subj)
	if arm == nil {
		return Value{}, unreachableMatch(subj)
	}
	return ip.evalArmValue(arm.Body, armScopeFor(vp, subj, scope))
}

// evalArmValue evaluates a value-position match arm body and returns its value.
// A value-position arm body is an expression (the parser produces `=> expr`); a
// statement/block body in value position is a descriptive refusal.
func (ip *Interp) evalArmValue(body ast.Node, scope *Env) (Value, error) {
	if e, ok := body.(ast.Expr); ok {
		return ip.evalExpr(e, scope)
	}
	return Value{}, fmt.Errorf("interp: value-position match arm body must be an expression, got %T", body)
}

// evalUnwrap evaluates the postfix `?` operator (the second of the two
// genuinely non-Go runtime mechanics). It evaluates the operand to a Result or
// Option tagged-union value and either yields the unwrapped success
// (Ok/Some) so evaluation continues, or performs a NON-LOCAL early return on
// failure (Err/None) by raising the enclosing function's own error/none result
// as a returnSignal — recovered at the call boundary exactly like an explicit
// `return Result.Err(...)`. An operand that is not a Result or Option variant is
// a located, descriptive refusal rather than a silent value.
func (ip *Interp) evalUnwrap(u *ast.UnwrapExpr, scope *Env) (Value, error) {
	v, err := ip.evalExpr(u.X, scope)
	if err != nil {
		return Value{}, err
	}
	if v.Kind != KindVariant || v.Variant == nil {
		return Value{}, fmt.Errorf("interp: %s: cannot use ? on %s (operand is not a Result or Option)", u.Question, v.Kind)
	}
	sig, _ := ip.curSig()
	switch v.Variant.TypeID {
	case resultTypeID:
		// `?` on a Result requires a Result-returning enclosing function — checked
		// up front so the erased static guarantee fails loudly on BOTH the success
		// and failure paths, not only on propagation.
		if sig.Mode != sema.ModeResult && sig.Mode != sema.ModeResultClosed {
			return Value{}, fmt.Errorf("interp: %s: ? used on a Result outside a Result-returning function", u.Question)
		}
		if v.Variant.Tag == resultOkTag {
			if pv, ok := payloadValue(v.Variant); ok {
				return pv, nil
			}
			return NilVal(), nil
		}
		errVal, _ := payloadValue(v.Variant)
		return Value{}, ip.propagateErr(u, errVal, sig)
	case optionTypeID:
		if sig.Mode != sema.ModeOption {
			return Value{}, fmt.Errorf("interp: %s: ? used on an Option outside an Option-returning function", u.Question)
		}
		if v.Variant.Tag == optionSomeTag {
			if pv, ok := payloadValue(v.Variant); ok {
				return pv, nil
			}
			return NilVal(), nil
		}
		return Value{}, ip.propagateNone()
	default:
		return Value{}, fmt.Errorf("interp: %s: cannot use ? on %s.%s (operand is not a Result or Option)", u.Question, v.Variant.TypeID, v.Variant.Tag)
	}
}

// propagateErr performs the early-return half of `?` on a failed Result: it
// builds the ENCLOSING function's own `Result.Err(...)` (its validated signature
// sig) and raises it as a returnSignal, recovered at the call boundary exactly
// like an explicit `return Result.Err(...)`. For an open-E (`Result[T, error]`)
// caller the error propagates unchanged; for a closed-E (`Result[T, E]`) caller
// whose error type differs from the failing callee's, the registered `from func`
// conversion is applied to the error before it is re-wrapped.
func (ip *Interp) propagateErr(u *ast.UnwrapExpr, errVal Value, sig sema.FuncSig) error {
	out := errVal
	if sig.Mode == sema.ModeResultClosed {
		calleeE := ip.calleeErrType(u.X)
		if calleeE != "" && calleeE != sig.E {
			conv, ok := ip.info.FromRegistry[[2]string{calleeE, sig.E}]
			if !ok {
				return fmt.Errorf("interp: %s: ? cannot propagate %s as %s (no from conversion registered)", u.Question, calleeE, sig.E)
			}
			converted, err := ip.callConversion(conv.Name, errVal, u)
			if err != nil {
				return err
			}
			out = converted
		}
	}
	return returnSignal{vals: []Value{VariantVal(resultTypeID, resultErrTag, map[string]Value{resultErrField: out})}}
}

// propagateNone performs the early-return half of `?` on Option.None: it raises
// the enclosing function's own `Option.None` as a returnSignal. The enclosing
// function has already been validated as Option-returning by evalUnwrap.
func (ip *Interp) propagateNone() error {
	return returnSignal{vals: []Value{VariantVal(optionTypeID, optionNoneTag, nil)}}
}

// callConversion invokes a registered `from func` conversion (an ordinary
// callable bound in the root scope by registerFuncs) on a single error value,
// returning its single result. A missing or non-callable conversion, or a
// conversion that does not return exactly one value, is a located refusal.
func (ip *Interp) callConversion(name string, errVal Value, u *ast.UnwrapExpr) (Value, error) {
	fn, err := ip.root.Lookup(name)
	if err != nil {
		return Value{}, fmt.Errorf("interp: %s: from conversion %s is not callable: %w", u.Question, name, err)
	}
	if fn.Kind != KindFunc || fn.Func == nil {
		return Value{}, fmt.Errorf("interp: %s: from conversion %s is not a function", u.Question, name)
	}
	out, err := ip.callFunc(fn.Func, []Value{errVal})
	if err != nil {
		return Value{}, err
	}
	if len(out) != 1 {
		return Value{}, fmt.Errorf("interp: %s: from conversion %s returned %d values (want 1)", u.Question, name, len(out))
	}
	return out[0], nil
}

// calleeErrType returns the analyzed error type (the E in Result[T, E]) of the
// function called in a `?` operand, used to decide whether a closed-E `from`
// conversion is needed. It resolves only a direct call of a named top-level
// function (`f(...)?`); any other operand shape yields "" (no conversion).
func (ip *Interp) calleeErrType(x ast.Expr) string {
	call, ok := x.(*ast.CallExpr)
	if !ok {
		return ""
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return ""
	}
	return ip.sigFor(id.Name).E
}

// evalCompositeLit evaluates a composite literal into a struct, slice, or map
// value, selecting on the literal's declared type. A slice/array type yields a
// slice value (positional elements), a map type yields a map value (key: value
// elements), and a named (Ident) type yields a struct value (keyed
// field: value elements). v1 maps are string-keyed; positional struct literals
// and an elided/unsupported type are descriptive refusals.
func (ip *Interp) evalCompositeLit(c *ast.CompositeLit, scope *Env) (Value, error) {
	switch t := c.Type.(type) {
	case *ast.ArrayType:
		// Slices and arrays both evaluate to an ordered slice value in v1.
		elems := make([]Value, 0, len(c.Elts))
		for _, elt := range c.Elts {
			if _, ok := elt.(*ast.KeyValueExpr); ok {
				return Value{}, fmt.Errorf("interp: indexed slice element not supported")
			}
			v, err := ip.evalExpr(elt, scope)
			if err != nil {
				return Value{}, err
			}
			elems = append(elems, v)
		}
		return SliceVal(elems...), nil
	case *ast.MapType:
		entries := make(map[string]Value, len(c.Elts))
		for _, elt := range c.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				return Value{}, fmt.Errorf("interp: map literal element must be key: value, got %T", elt)
			}
			keyVal, err := ip.evalExpr(kv.Key, scope)
			if err != nil {
				return Value{}, err
			}
			key, err := mapKeyString(keyVal)
			if err != nil {
				return Value{}, err
			}
			val, err := ip.evalExpr(kv.Value, scope)
			if err != nil {
				return Value{}, err
			}
			entries[key] = val
		}
		return MapVal(entries), nil
	case *ast.Ident:
		fields := make(map[string]Value, len(c.Elts))
		for _, elt := range c.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				return Value{}, fmt.Errorf("interp: struct literal %s requires keyed field: value elements", t.Name)
			}
			name, ok := kv.Key.(*ast.Ident)
			if !ok {
				return Value{}, fmt.Errorf("interp: struct literal field name must be an identifier, got %T", kv.Key)
			}
			val, err := ip.evalExpr(kv.Value, scope)
			if err != nil {
				return Value{}, err
			}
			fields[name.Name] = val
		}
		return StructVal(t.Name, fields), nil
	default:
		return Value{}, fmt.Errorf("interp: unsupported composite literal type %T", c.Type)
	}
}

// evalSelector evaluates a field selector x.field. A data-less enum variant
// construction (`Status.Pending`) is intercepted first: the receiver names an
// enum type (not shadowed by a value binding) and the selected name is one of
// its variants, so it constructs a tagged-union Value. Otherwise the receiver
// must evaluate to a struct value and the named field is read from it. A
// non-struct receiver or an absent field is a descriptive refusal.
func (ip *Interp) evalSelector(s *ast.SelectorExpr, scope *Env) (Value, error) {
	// Data-less enum construction: `Enum.Variant` with no parens parses to a
	// selector. It is enum construction only when the receiver is an enum type
	// name not shadowed by a value binding and the selected name is a declared
	// variant; otherwise it is an ordinary field read.
	if id, ok := s.X.(*ast.Ident); ok {
		if enum, ok := ip.enumByName(id.Name); ok && enum.VSet[s.Sel.Name] {
			if _, err := scope.Lookup(id.Name); err != nil {
				return VariantVal(enum.Name, s.Sel.Name, nil), nil
			}
		}
		// Data-less Option construction: `Option.None` with no parens parses to a
		// selector. Option is a built-in (not in info.Enums), so it needs its own
		// guard — the receiver is `Option`, not shadowed by a value binding, and
		// the selected name is the None variant.
		if id.Name == optionTypeID && s.Sel.Name == optionNoneTag {
			if _, err := scope.Lookup(id.Name); err != nil {
				return VariantVal(optionTypeID, optionNoneTag, nil), nil
			}
		}
	}
	recv, err := ip.evalExpr(s.X, scope)
	if err != nil {
		return Value{}, err
	}
	switch recv.Kind {
	case KindStruct:
		if recv.Struct == nil {
			return Value{}, fmt.Errorf("interp: cannot select field %s on %s", s.Sel.Name, recv.Kind)
		}
		v, ok := recv.Struct.Fields[s.Sel.Name]
		if !ok {
			return Value{}, fmt.Errorf("interp: %s has no field %s", recv.Struct.TypeID, s.Sel.Name)
		}
		return v, nil
	case KindVariant:
		// A match arm binds the whole variant value to its payload name, so a
		// payload field is read as `binding.field` off the tagged union.
		if recv.Variant == nil {
			return Value{}, fmt.Errorf("interp: cannot select field %s on %s", s.Sel.Name, recv.Kind)
		}
		v, ok := recv.Field(s.Sel.Name)
		if !ok {
			return Value{}, fmt.Errorf("interp: %s.%s has no payload field %s", recv.Variant.TypeID, recv.Variant.Tag, s.Sel.Name)
		}
		return v, nil
	default:
		return Value{}, fmt.Errorf("interp: cannot select field %s on %s", s.Sel.Name, recv.Kind)
	}
}

// evalVariantLit evaluates a payload-carrying enum construction
// (`Status.Active(since: now())`) into a tagged-union Value. The enum reference
// must name a known enum and the variant must be one of its declared variants;
// each labeled argument fills the named payload field, and a positional argument
// fills the declared field at its index (matching the variant's declared field
// order). An unknown enum, unknown variant, unknown field, or out-of-range
// positional argument is a descriptive refusal rather than a silent value.
func (ip *Interp) evalVariantLit(vl *ast.VariantLit, scope *Env) (Value, error) {
	id, ok := vl.Enum.(*ast.Ident)
	if !ok {
		if vl.Enum == nil {
			return Value{}, fmt.Errorf("interp: variant construction requires an enum reference")
		}
		return Value{}, fmt.Errorf("interp: unsupported enum reference %T in variant construction", vl.Enum)
	}
	enum, ok := ip.enumByName(id.Name)
	if !ok {
		return Value{}, fmt.Errorf("interp: unknown enum %s in variant construction", id.Name)
	}
	if vl.Variant == nil {
		return Value{}, fmt.Errorf("interp: variant construction on enum %s is missing a variant tag", enum.Name)
	}
	tag := vl.Variant.Name
	if !enum.VSet[tag] {
		return Value{}, fmt.Errorf("interp: enum %s has no variant %s", enum.Name, tag)
	}
	declared := variantFields(enum, tag)
	fields := make(map[string]Value, len(vl.Args))
	for i, arg := range vl.Args {
		switch a := arg.(type) {
		case *ast.LabeledArg:
			if a.Label == nil {
				return Value{}, fmt.Errorf("interp: %s.%s has an unlabeled argument", enum.Name, tag)
			}
			v, err := ip.evalExpr(a.Value, scope)
			if err != nil {
				return Value{}, err
			}
			fields[a.Label.Name] = v
		default:
			// A positional argument binds to the declared field at this index.
			if i >= len(declared) {
				return Value{}, fmt.Errorf("interp: %s.%s has too many arguments (declares %d field(s))", enum.Name, tag, len(declared))
			}
			v, err := ip.evalExpr(arg, scope)
			if err != nil {
				return Value{}, err
			}
			fields[declared[i].Name] = v
		}
	}
	return VariantVal(enum.Name, tag, fields), nil
}

// enumByName returns the resolved enum with the given name, or ok=false when the
// interpreter has no sema info or the name is not an enum. It is the nil-safe
// gate every enum-construction path consults.
func (ip *Interp) enumByName(name string) (*sema.Enum, bool) {
	if ip.info == nil || ip.info.Enums == nil {
		return nil, false
	}
	enum, ok := ip.info.Enums[name]
	if !ok || enum == nil {
		return nil, false
	}
	return enum, true
}

// variantFields returns the declared fields of the named variant of enum, in
// source order, used to resolve positional construction arguments.
func variantFields(enum *sema.Enum, tag string) []sema.Field {
	for _, v := range enum.Variants {
		if v.Name == tag {
			return v.Fields
		}
	}
	return nil
}

// evalResultCtor evaluates a Result construction `Result.Ok(x)` / `Result.Err(e)`
// into the universal tagged-union Value. Ok carries the success value, Err the
// error value, each as a single payload field — the SAME encoding whether the
// error type is `error` (open-E) or an enum (closed-E); the interpreter applies no
// (T, error) optimization. An unknown Result constructor name or other than one
// argument is a located, descriptive refusal rather than a silent value.
func (ip *Interp) evalResultCtor(ctor string, call *ast.CallExpr, scope *Env) ([]Value, error) {
	var tag, field string
	switch ctor {
	case resultOkTag:
		tag, field = resultOkTag, resultOkField
	case resultErrTag:
		tag, field = resultErrTag, resultErrField
	default:
		return nil, fmt.Errorf("interp: %s: unknown Result constructor %s.%s (expected Ok or Err)", call.Pos(), resultTypeID, ctor)
	}
	if len(call.Args) != 1 {
		return nil, fmt.Errorf("interp: %s: %s.%s expects 1 argument, got %d", call.Pos(), resultTypeID, ctor, len(call.Args))
	}
	v, err := ip.evalExpr(call.Args[0], scope)
	if err != nil {
		return nil, err
	}
	return []Value{VariantVal(resultTypeID, tag, map[string]Value{field: v})}, nil
}

// evalOptionCtor evaluates an Option construction `Option.Some(x)` into the
// universal tagged-union Value, carrying the present value as its single payload
// — the interpreter applies no *T optimization. `Option.None` is the data-less
// form (no parens) and is handled in evalSelector, so only the Some constructor
// reaches here; any other constructor name or other than one argument is a
// located, descriptive refusal rather than a silent value.
func (ip *Interp) evalOptionCtor(ctor string, call *ast.CallExpr, scope *Env) ([]Value, error) {
	if ctor != optionSomeTag {
		return nil, fmt.Errorf("interp: %s: unknown Option constructor %s.%s (expected Some or None)", call.Pos(), optionTypeID, ctor)
	}
	if len(call.Args) != 1 {
		return nil, fmt.Errorf("interp: %s: %s.%s expects 1 argument, got %d", call.Pos(), optionTypeID, ctor, len(call.Args))
	}
	v, err := ip.evalExpr(call.Args[0], scope)
	if err != nil {
		return nil, err
	}
	return []Value{VariantVal(optionTypeID, optionSomeTag, map[string]Value{optionSomeField: v})}, nil
}

// evalIndex evaluates an index expression x[i]. A slice is indexed by an integer
// (bounds-checked); a map is indexed by a string key (an absent key reads the nil
// value, the defined absent-read result). Indexing any other kind is a
// descriptive refusal.
func (ip *Interp) evalIndex(e *ast.IndexExpr, scope *Env) (Value, error) {
	recv, err := ip.evalExpr(e.X, scope)
	if err != nil {
		return Value{}, err
	}
	idx, err := ip.evalExpr(e.Index, scope)
	if err != nil {
		return Value{}, err
	}
	switch recv.Kind {
	case KindSlice:
		if idx.Kind != KindInt {
			return Value{}, fmt.Errorf("interp: slice index must be int, got %s", idx.Kind)
		}
		if idx.Int < 0 || idx.Int >= int64(len(recv.Slice)) {
			return Value{}, fmt.Errorf("interp: slice index %d out of range (len %d)", idx.Int, len(recv.Slice))
		}
		return recv.Slice[idx.Int], nil
	case KindMap:
		key, err := mapKeyString(idx)
		if err != nil {
			return Value{}, err
		}
		if recv.Map == nil {
			return NilVal(), nil
		}
		if v, ok := recv.Map.Entries[key]; ok {
			return v, nil
		}
		return NilVal(), nil
	default:
		return Value{}, fmt.Errorf("interp: cannot index %s", recv.Kind)
	}
}

// mapKeyString returns the string key for a v1 (string-keyed) map. A non-string
// key is a descriptive refusal; non-string map keys are deferred to a later
// evaluation story.
func mapKeyString(v Value) (string, error) {
	if v.Kind != KindString {
		return "", fmt.Errorf("interp: map key must be string, got %s", v.Kind)
	}
	return v.Str, nil
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
	// A builtin call (len/append/make/panic) is intercepted before the generic
	// function-value path, but only when the name is not shadowed by a binding
	// in scope (a user value bound to that name wins, matching Go's shadowing).
	if id, ok := call.Fun.(*ast.Ident); ok && isBuiltin(id.Name) {
		if _, err := scope.Lookup(id.Name); err != nil {
			return ip.evalBuiltin(id.Name, call, scope)
		}
	}
	// A Result construction (`Result.Ok(x)` / `Result.Err(e)`) is a selector call
	// whose receiver is the built-in `Result` (not shadowed by a local binding).
	// Bare positional args mean the parser produced an ordinary *ast.CallExpr, so
	// intercept it here and build the universal tagged-union value — uniformly for
	// open-E and closed-E, with no (T, error) optimization.
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if recv, ok := sel.X.(*ast.Ident); ok && recv.Name == resultTypeID {
			if _, err := scope.Lookup(recv.Name); err != nil {
				return ip.evalResultCtor(sel.Sel.Name, call, scope)
			}
		}
	}
	// An Option construction (`Option.Some(x)`) is a selector call whose receiver
	// is the built-in `Option` (not shadowed by a local binding), the same node
	// shape as a Result construction. `Option.None` has no parens and is handled
	// in evalSelector instead.
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if recv, ok := sel.X.(*ast.Ident); ok && recv.Name == optionTypeID {
			if _, err := scope.Lookup(recv.Name); err != nil {
				return ip.evalOptionCtor(sel.Sel.Name, call, scope)
			}
		}
	}
	// A selector call whose receiver names an imported package — and is not
	// shadowed by a local binding of that name — is a host-package call
	// (fmt.Sprintf, errors.New, ...). Route it to the host-function bridge,
	// which resolves a registered shim or raises a located, named refusal.
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if pkg, ok := sel.X.(*ast.Ident); ok && ip.imports[pkg.Name] != "" {
			if _, err := scope.Lookup(pkg.Name); err != nil {
				return ip.evalHostCall(sel, call, scope)
			}
		}
	}
	// A method call x.M(...) is a selector whose receiver evaluates to a struct
	// value with a method M declared on its type. If that resolves, dispatch the
	// method; otherwise fall through to the generic path (so a struct field
	// holding a function value is handled as before).
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if vals, handled, err := ip.tryMethodCall(sel, call, scope); handled {
			return vals, err
		}
	}

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

// isBuiltin reports whether name is one of the interpreter's builtin functions.
func isBuiltin(name string) bool {
	switch name {
	case "len", "append", "make", "panic":
		return true
	default:
		return false
	}
}

// evalBuiltin evaluates a call to a builtin function (len/append/make/panic).
// make reads its first argument as a TYPE expression; the others evaluate their
// arguments as ordinary values. Each builtin validates its argument count and
// operand kinds and yields a descriptive refusal rather than a silent nil.
func (ip *Interp) evalBuiltin(name string, call *ast.CallExpr, scope *Env) ([]Value, error) {
	switch name {
	case "len":
		return ip.builtinLen(call, scope)
	case "append":
		return ip.builtinAppend(call, scope)
	case "make":
		return ip.builtinMake(call, scope)
	case "panic":
		return ip.builtinPanic(call, scope)
	default:
		return nil, fmt.Errorf("interp: unknown builtin %s", name)
	}
}

// builtinLen evaluates len(x): the number of elements in a slice, bytes in a
// string, or entries in a map. Any other operand kind is a descriptive refusal.
func (ip *Interp) builtinLen(call *ast.CallExpr, scope *Env) ([]Value, error) {
	if len(call.Args) != 1 {
		return nil, fmt.Errorf("interp: len expects 1 argument, got %d", len(call.Args))
	}
	v, err := ip.evalExpr(call.Args[0], scope)
	if err != nil {
		return nil, err
	}
	switch v.Kind {
	case KindSlice:
		return []Value{IntVal(int64(len(v.Slice)))}, nil
	case KindString:
		return []Value{IntVal(int64(len(v.Str)))}, nil
	case KindMap:
		if v.Map == nil {
			return []Value{IntVal(0)}, nil
		}
		return []Value{IntVal(int64(len(v.Map.Entries)))}, nil
	default:
		return nil, fmt.Errorf("interp: len of %s is not defined", v.Kind)
	}
}

// builtinAppend evaluates append(s, elems...): it copies the first (slice)
// argument and appends each subsequent argument value, returning a NEW slice
// value (the interpreter does not model backing-array aliasing). A non-slice
// first argument is a descriptive refusal.
func (ip *Interp) builtinAppend(call *ast.CallExpr, scope *Env) ([]Value, error) {
	if len(call.Args) < 1 {
		return nil, fmt.Errorf("interp: append expects at least 1 argument, got %d", len(call.Args))
	}
	base, err := ip.evalExpr(call.Args[0], scope)
	if err != nil {
		return nil, err
	}
	if base.Kind != KindSlice {
		return nil, fmt.Errorf("interp: append of %s (first argument must be a slice)", base.Kind)
	}
	out := make([]Value, len(base.Slice), len(base.Slice)+len(call.Args)-1)
	copy(out, base.Slice)
	for _, a := range call.Args[1:] {
		v, err := ip.evalExpr(a, scope)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return []Value{SliceVal(out...)}, nil
}

// builtinMake evaluates make(T, sizes...): its first argument is a TYPE
// expression. A map type yields an empty map; an array/slice type yields a slice
// of length n (the optional second argument, default 0) filled with the element
// type's safe zero value. Any other type is a descriptive refusal.
func (ip *Interp) builtinMake(call *ast.CallExpr, scope *Env) ([]Value, error) {
	if len(call.Args) < 1 {
		return nil, fmt.Errorf("interp: make expects at least 1 argument, got %d", len(call.Args))
	}
	switch t := call.Args[0].(type) {
	case *ast.MapType:
		return []Value{MapVal(nil)}, nil
	case *ast.ArrayType:
		n := 0
		if len(call.Args) >= 2 {
			sz, err := ip.evalExpr(call.Args[1], scope)
			if err != nil {
				return nil, err
			}
			if sz.Kind != KindInt {
				return nil, fmt.Errorf("interp: make length must be int, got %s", sz.Kind)
			}
			if sz.Int < 0 {
				return nil, fmt.Errorf("interp: make length %d is negative", sz.Int)
			}
			n = int(sz.Int)
		}
		zero := zeroValue(t.Elt)
		elems := make([]Value, n)
		for i := range elems {
			elems[i] = zero
		}
		return []Value{SliceVal(elems...)}, nil
	default:
		return nil, fmt.Errorf("interp: make of %T is not supported", call.Args[0])
	}
}

// builtinPanic evaluates panic(x): it evaluates the operand and raises a
// panicSignal carrying its value, unwinding past every loop, switch, and call
// boundary to the host (which observes it as the "recovered panic").
func (ip *Interp) builtinPanic(call *ast.CallExpr, scope *Env) ([]Value, error) {
	if len(call.Args) != 1 {
		return nil, fmt.Errorf("interp: panic expects 1 argument, got %d", len(call.Args))
	}
	v, err := ip.evalExpr(call.Args[0], scope)
	if err != nil {
		return nil, err
	}
	return nil, panicSignal{value: v}
}

// tryMethodCall attempts to dispatch a selector call x.M(...) as a method on a
// struct receiver. handled is true only when the receiver evaluates to a struct
// value whose type declares a method M; otherwise (a non-struct receiver, an
// unknown method, or a receiver-evaluation error) it returns handled=false so
// evalCallMulti falls through to its generic function-value path, which surfaces
// the right error or calls a struct field that holds a function value.
func (ip *Interp) tryMethodCall(sel *ast.SelectorExpr, call *ast.CallExpr, scope *Env) (vals []Value, handled bool, err error) {
	recv, rerr := ip.evalExpr(sel.X, scope)
	if rerr != nil {
		return nil, false, nil
	}
	if recv.Kind != KindStruct || recv.Struct == nil {
		return nil, false, nil
	}
	byName := ip.methods[recv.Struct.TypeID]
	if byName == nil {
		return nil, false, nil
	}
	decl, ok := byName[sel.Sel.Name]
	if !ok {
		return nil, false, nil
	}
	args := make([]Value, len(call.Args))
	for i, a := range call.Args {
		v, aerr := ip.evalExpr(a, scope)
		if aerr != nil {
			return nil, true, aerr
		}
		args[i] = v
	}
	out, merr := ip.callMethod(decl, recv, args)
	return out, true, merr
}

// copyStructValue returns a shallow copy of a struct value (a fresh
// StructValue with a fresh Fields map sharing the field values), so a
// value-receiver method's field writes do not leak to the caller. A non-struct
// value is returned unchanged.
func copyStructValue(v Value) Value {
	if v.Kind != KindStruct || v.Struct == nil {
		return v
	}
	fields := make(map[string]Value, len(v.Struct.Fields))
	for k, fv := range v.Struct.Fields {
		fields[k] = fv
	}
	return StructVal(v.Struct.TypeID, fields)
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

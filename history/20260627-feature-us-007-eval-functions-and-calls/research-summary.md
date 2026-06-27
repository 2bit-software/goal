# Research Summary — US-007 Eval functions and calls

## Summary

This is an internal interpreter feature, not an external-library question. The
established pattern for a tree-walking interpreter is:

1. **Function values carry their declaration** (a closure over the defining
   scope). internal/interp already has `FuncValue{Name}`; extend it to hold the
   `*ast.FuncDecl` and the defining `*Env`.
2. **Top-level functions are pre-registered** in the root scope so recursion and
   forward references resolve via the ordinary name-lookup path.
3. **Return is a non-local control signal.** The canonical Go approach (used by
   gopher-lua, monkey/"Writing An Interpreter In Go", yaegi) is a sentinel value
   threaded up through statement execution that the call boundary intercepts.
   Here we use a typed sentinel error (`returnSignal` carrying `[]Value`) so a
   `return` nested in blocks unwinds cleanly to the enclosing call.

## Confidence

High. The seams already exist (execStmt dispatch, evalExpr dispatch, Env chain);
this story extends them with CallExpr/ReturnStmt/IfStmt and a richer FuncValue.

## Open questions

None blocking. Multiple-return modelling: a call producing N>1 results yields a
tuple Value (KindSlice acts as the carrier at the call site only when consumed
positionally); for US-007 the consuming sites are assignment RHS lists handled
by execAssign and direct factorial/fibonacci single returns.

## Recommended next steps

Proceed to plan: register funcs, extend FuncValue, add CallExpr/ReturnStmt/
IfStmt evaluation with a returnSignal, and a multi-value return test.

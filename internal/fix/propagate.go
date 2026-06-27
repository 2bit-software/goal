package fix

import (
	"strings"

	"goal/internal/analyze"
	"goal/internal/ast"
	"goal/internal/scan"
	"goal/internal/sema"
	"goal/internal/token"
)

// fixPropagate collapses manual error/nil propagation into the `?` operator. Inside a
// function that returns Result[T, error] or Option[T], the Go shape
//
//	v, err := g(args)
//	if err != nil {
//	    return zero, err          // or: return Result.Err(err)
//	}
//
// becomes `v := g(args)?`, and the Option shape
//
//	o := g(args)
//	if o == nil {
//	    return Option.None        // or: return nil
//	}
//
// becomes `o := g(args)?` with later `*o` dereferences rewritten to `o`. A block that does
// anything other than this exact propagation (wrapping, logging, a non-zero return, a
// comment in the way, a multi-line binding, an escaping Option pointer) is left untouched
// and recorded as a Skip — `?` is only applied where the rewrite is provably equivalent.
//
// The candidate is a binding statement immediately followed by the guarding `if`, found as
// consecutive statements in a block of the AST.
func fixPropagate(src string, file *ast.File, info *sema.Info, decls map[string]string, changes *[]Change, reports *[]Report) []scan.Replacement {
	var reps []scan.Replacement
	for _, fn := range resultOptionFuncs(file, info) {
		isResult := info.FuncSignatures[fn.Name.Name].Mode == sema.ModeResult
		sigT := info.FuncSignatures[fn.Name.Name].T
		forEachBlock(fn.Body, func(list []ast.Stmt) {
			for i := 0; i+1 < len(list); i++ {
				as, ok := list[i].(*ast.AssignStmt)
				if !ok {
					continue
				}
				ifs, ok := list[i+1].(*ast.IfStmt)
				if !ok {
					continue
				}
				rs := tryCollapseBinding(src, fn, as, ifs, isResult, sigT, decls, changes, reports)
				reps = append(reps, rs...)
			}
		})
	}
	return reps
}

// tryCollapseBinding attempts the value-binding propagation collapse for the AssignStmt /
// IfStmt pair (as, ifs). It returns the replacements to apply, or nil when the shape does
// not match (recording a Skip when a near-match is refused for safety).
func tryCollapseBinding(src string, fn *ast.FuncDecl, as *ast.AssignStmt, ifs *ast.IfStmt, isResult bool, sigT string, decls map[string]string, changes *[]Change, reports *[]Report) []scan.Replacement {
	// The if must be a bare `condVar != nil` (Result) / `condVar == nil` (Option) guard
	// with no init clause and no else.
	if ifs.Init != nil || ifs.Else != nil {
		return nil
	}
	condVar, ok := nilGuardVar(ifs.Cond, isResult)
	if !ok {
		return nil
	}
	// The body must be exactly one propagation return.
	if ifs.Body == nil || len(ifs.Body.List) != 1 {
		return nil
	}
	ret, ok := ifs.Body.List[0].(*ast.ReturnStmt)
	if !ok || !validPropagationReturn(src, ret, isResult, condVar, sigT, decls) {
		return nil
	}
	// The binding must be `value, condVar := rhs` (Result) / `condVar := rhs` (Option), a
	// single-line `:=` directly above the if.
	if as.Tok != token.DEFINE || len(as.Rhs) == 0 {
		return nil
	}
	value, ok := propagationLHS(as, condVar, isResult)
	if !ok {
		return nil
	}
	if !adjacentSingleLine(src, as, ifs) {
		return nil
	}

	ifLineStart := lineStartBefore(src, ifs.If.Offset)
	if ifLineStart == 0 {
		return nil
	}
	bindLineStart := lineStartBefore(src, as.Pos().Offset)

	// Safety guard: never drop a comment hiding in the collapsed region.
	if spanHasComment(src, bindLineStart, ifs.End().Offset) {
		*reports = append(*reports, Report{lineOf(src, ifs.If.Offset), Skip, "propagate",
			"propagation block has a comment; left as-is to avoid dropping it"})
		return nil
	}

	rhs := strings.TrimSpace(src[as.Rhs[0].Pos().Offset:as.Rhs[len(as.Rhs)-1].End().Offset])
	indent := indentOf(src, bindLineStart)
	reps := []scan.Replacement{{
		Start: bindLineStart,
		End:   ifs.End().Offset,
		Text:  indent + value + " := " + rhs + "?",
	}}

	// Option: the value is now the unwrapped T, so rewrite later `*o` uses to `o`.
	if !isResult {
		derefReps, derefOK := optionDerefRewrites(fn, condVar, ifs.End().Offset)
		if !derefOK {
			// The pointer escapes (used other than as `*o`); abandon this collapse.
			*reports = append(*reports, Report{lineOf(src, ifs.If.Offset), Skip, "propagate",
				"Option value used other than `*" + condVar + "`; left as-is"})
			return nil
		}
		reps = append(reps, derefReps...)
	}
	*changes = append(*changes, Change{lineOf(src, ifs.If.Offset), "propagate"})
	return reps
}

// fixPropagateInit collapses the statement-context error guard
//
//	if err := g(args); err != nil {
//	    return Result.Err(err)        // or: return zero, err
//	}
//
// into `g(args)?` inside a function returning Result[T, error], for a call whose only output
// is the error (the init clause binds exactly the condition variable, nothing else). It
// complements fixPropagate, which handles the value-binding form where the call's result is
// bound on the line above the `if`. Like that rule it is conservative: a decorated or non-zero
// return, an `else`, a comment in the block, or a value bound alongside the error leaves the
// guard untouched — `?` is only applied where the rewrite is provably equivalent.
func fixPropagateInit(src string, file *ast.File, info *sema.Info, decls map[string]string, changes *[]Change, reports *[]Report) []scan.Replacement {
	var reps []scan.Replacement
	for _, fn := range resultOptionFuncs(file, info) {
		if info.FuncSignatures[fn.Name.Name].Mode != sema.ModeResult {
			continue // `?` only propagates the error of an open-E Result function
		}
		sigT := info.FuncSignatures[fn.Name.Name].T
		forEachBlock(fn.Body, func(list []ast.Stmt) {
			for _, s := range list {
				ifs, ok := s.(*ast.IfStmt)
				if !ok || ifs.Init == nil || ifs.Else != nil {
					continue
				}
				init, ok := ifs.Init.(*ast.AssignStmt)
				if !ok || init.Tok != token.DEFINE || len(init.Rhs) == 0 {
					continue
				}
				condVar, ok := nilGuardVar(ifs.Cond, true)
				if !ok {
					continue
				}
				// The init clause must bind exactly the condition variable, so the call's
				// sole output is the error and a bare `CALL?` discards nothing.
				names := identNames(init.Lhs)
				if len(names) != 1 || names[0] != condVar {
					continue
				}
				if ifs.Body == nil || len(ifs.Body.List) != 1 {
					continue
				}
				ret, ok := ifs.Body.List[0].(*ast.ReturnStmt)
				if !ok || !validPropagationReturn(src, ret, true, condVar, sigT, decls) {
					continue
				}
				if spanHasComment(src, ifs.If.Offset, ifs.End().Offset) {
					*reports = append(*reports, Report{lineOf(src, ifs.If.Offset), Skip, "propagate",
						"propagation block has a comment; left as-is to avoid dropping it"})
					continue
				}
				rhs := strings.TrimSpace(src[init.Rhs[0].Pos().Offset:init.Rhs[len(init.Rhs)-1].End().Offset])
				lineStart := lineStartBefore(src, ifs.If.Offset)
				reps = append(reps, scan.Replacement{
					Start: lineStart,
					End:   ifs.End().Offset,
					Text:  indentOf(src, lineStart) + rhs + "?",
				})
				*changes = append(*changes, Change{lineOf(src, ifs.If.Offset), "propagate"})
			}
		})
	}
	return reps
}

// resultOptionFuncs returns the file's Result/Option-returning top-level functions (those a
// propagation collapse can target), in source order.
func resultOptionFuncs(file *ast.File, info *sema.Info) []*ast.FuncDecl {
	var fns []*ast.FuncDecl
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Body == nil {
			continue
		}
		switch info.FuncSignatures[fn.Name.Name].Mode {
		case sema.ModeResult, sema.ModeOption:
			fns = append(fns, fn)
		}
	}
	return fns
}

// nilGuardVar returns the condition variable of a nil-guard `v != nil` (Result) / `v == nil`
// (Option) condition, with ok=false when cond is not that exact shape.
func nilGuardVar(cond ast.Expr, isResult bool) (string, bool) {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return "", false
	}
	wantOp := token.EQL
	if isResult {
		wantOp = token.NEQ
	}
	if bin.Op != wantOp || identName(bin.Y) != "nil" {
		return "", false
	}
	v := identName(bin.X)
	if v == "" {
		return "", false
	}
	return v, true
}

// adjacentSingleLine reports whether the binding as occupies a single line directly above
// the if ifs, with only whitespace (one newline) between them — the layout the original
// line-adjacency rule required, so a blank line or a multi-line binding is never collapsed.
func adjacentSingleLine(src string, as *ast.AssignStmt, ifs *ast.IfStmt) bool {
	lo, hi := as.Pos().Offset, as.End().Offset
	if lo < 0 || hi > len(src) || strings.ContainsRune(src[lo:hi], '\n') {
		return false // multi-line binding
	}
	gap := src[as.End().Offset:ifs.If.Offset]
	return strings.Count(gap, "\n") == 1 && strings.Trim(gap, " \t\n") == ""
}

// validPropagationReturn reports whether ret is exactly the early return of a propagation
// block: `return zero, err` / `return Result.Err(err)` for Result, `return Option.None` /
// `return nil` for Option. For Result it also requires the returned zero to match the
// success type's computed zero (so a real value is never discarded).
func validPropagationReturn(src string, ret *ast.ReturnStmt, isResult bool, condVar, successT string, decls map[string]string) bool {
	ops := ret.Results
	if isResult {
		// return Result.Err(err)
		if len(ops) == 1 && isResultErrOf(ops[0], condVar) {
			return true
		}
		// return zero, err
		if len(ops) != 2 || identName(ops[1]) != condVar {
			return false
		}
		zeroActual := nodeText(src, ops[0])
		return zeroActual == analyze.ZeroLit(successT, decls, 0)
	}
	// Option: return Option.None | return nil
	if len(ops) != 1 {
		return false
	}
	return isSelector(ops[0], "Option", "None") || identName(ops[0]) == "nil"
}

// propagationLHS validates the binding's left-hand side against the condition variable and
// returns the value name to keep. For Result the LHS is `value, err` (err must be the
// condition variable); for Option it is the single pointer name.
func propagationLHS(as *ast.AssignStmt, condVar string, isResult bool) (value string, ok bool) {
	names := identNames(as.Lhs)
	if names == nil {
		return "", false
	}
	if isResult {
		if len(names) != 2 || names[1] != condVar {
			return "", false
		}
		return names[0], true
	}
	if len(names) != 1 || names[0] != condVar {
		return "", false
	}
	return names[0], true
}

// optionDerefRewrites returns the replacements turning each `*o` after byte offset from into
// `o`, now that `o` holds the unwrapped value. ok is false if `o` is referenced in any other
// shape (a bare `o`, `o.field`), which means the pointer escapes and the collapse must be
// abandoned.
func optionDerefRewrites(fn *ast.FuncDecl, optVar string, from int) (reps []scan.Replacement, ok bool) {
	derefAt := map[int]*ast.StarExpr{} // inner-ident offset -> enclosing `*o`
	var uses []*ast.Ident
	ast.Walk(visitFn(func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.StarExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == optVar && id.Pos().Offset >= from {
				derefAt[id.Pos().Offset] = x
			}
		case *ast.Ident:
			if x.Name == optVar && x.Pos().Offset >= from {
				uses = append(uses, x)
			}
		}
		return true
	}), fn.Body)

	for _, id := range uses {
		st, isDeref := derefAt[id.Pos().Offset]
		if !isDeref {
			return nil, false // bare use of the pointer — escapes
		}
		reps = append(reps, scan.Replacement{Start: st.Star.Offset, End: id.End().Offset, Text: optVar})
	}
	return reps, true
}

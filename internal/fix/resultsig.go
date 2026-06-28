package fix

import (
	"goal/internal/analyze"
	"goal/internal/ast"
	"goal/internal/scan"
	"goal/internal/sema"
	"goal/internal/token"
)

// fixResultSig converts a function written as a Go `(T, error)` tuple into one returning
// `Result[T, error]`, the keystone Go→goal migration. It rewrites the signature and turns
// each `return v, nil` success exit into `return Result.Ok(v)`; the manual `if err != nil`
// propagation blocks are left for fixPropagate to collapse on the next pass (once the
// signature change has marked the function Result-returning).
//
// The conversion is all-or-nothing (every return must be a recognized success or bare
// propagation): if any return decorates, wraps, or returns a non-zero value alongside the
// error, the signature is left unchanged and a Skip records why — a half-converted
// signature would be worse than none. Only the single-non-error-value shape `(T, error)` is
// mapped; `(A, B, error)` and bare `error` are reported as out of scope. An exported
// function additionally gets a Warn, since callers outside the scanned path may break.
func fixResultSig(src string, file *ast.File, info *sema.Info, decls map[string]string, changes *[]Change, reports *[]Report) []scan.Replacement {
	var reps []scan.Replacement
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Body == nil || fn.Type == nil {
			continue
		}
		if sig, ok := info.FuncSignatures[fn.Name.Name]; !ok || sig.Mode != sema.ModeNone {
			continue // only plain functions; Result/Option are already idiomatic
		}
		nameLine := lineOf(src, fn.Name.Pos().Offset)
		res := fn.Type.Results
		if res == nil || len(res.List) == 0 {
			continue // no results — not a tuple
		}
		// An unparenthesized single result has no Opening paren: the only shape worth a
		// note is a bare `error` return, which fix cannot auto-convert to Result.
		if res.Opening == (token.Pos{}) {
			if len(res.List) == 1 && len(res.List[0].Names) == 0 && identName(res.List[0].Type) == "error" {
				*reports = append(*reports, Report{nameLine, Skip, "result-sig",
					"`" + fn.Name.Name + "` returns a bare `error`; not auto-converted to Result"})
			}
			continue
		}
		// A parenthesized result list: convert only the all-unnamed `(T, error)` tuple.
		types := flattenResultTypes(res)
		if types == nil {
			continue // named results — out of the conservative scope
		}
		if len(types) < 2 {
			continue // single parenthesized return type, not a tuple
		}
		if identName(types[len(types)-1]) != "error" {
			continue // not error-terminated — not the shape fix targets
		}
		if len(types) > 2 {
			*reports = append(*reports, Report{nameLine, Skip, "result-sig",
				"`" + fn.Name.Name + "` returns multiple non-error values; not auto-converted to Result"})
			continue
		}
		successT := nodeText(src, types[0])

		// Every return in the body (excluding nested function literals) must be a
		// recognized success or bare propagation, or we abandon the whole function.
		successReps, conforms, badLine := classifyReturns(src, fn, successT, decls)
		if !conforms {
			*reports = append(*reports, Report{badLine, Skip, "result-sig",
				"`" + fn.Name.Name + "` has a non-propagating return; not auto-converted to Result"})
			continue
		}

		reps = append(reps, scan.Replacement{
			Start: res.Opening.Offset, End: res.Closing.Offset + 1,
			Text: "Result[" + successT + ", error]",
		})
		reps = append(reps, successReps...)
		*changes = append(*changes, Change{nameLine, "result-sig"})
		if isExported(fn.Name.Name) {
			*reports = append(*reports, Report{nameLine, Warn, "result-sig",
				"exported `" + fn.Name.Name + "` changed to Result[" + successT + ", error]; callers outside the scanned path may need manual updates"})
		}
	}
	return reps
}

// flattenResultTypes returns the result types of an all-unnamed parenthesized result list,
// or nil if any field carries a name (a named-result shape fix does not convert).
func flattenResultTypes(fl *ast.FieldList) []ast.Expr {
	out := make([]ast.Expr, 0, len(fl.List))
	for _, f := range fl.List {
		if len(f.Names) > 0 || f.Type == nil {
			return nil
		}
		out = append(out, f.Type)
	}
	return out
}

// classifyReturns inspects every top-level return of fn (skipping nested function literals)
// and returns the replacements that rewrite each `return v, nil` success into
// `return Result.Ok(v)`. conforms is false (with the offending line) if any return is
// neither a recognized success nor a bare propagation that fixPropagate will collapse.
func classifyReturns(src string, fn *ast.FuncDecl, successT string, decls map[string]string) (reps []scan.Replacement, conforms bool, badLine int) {
	conforms = true
	walkReturns(fn.Body, func(ret *ast.ReturnStmt) {
		if !conforms {
			return
		}
		line := lineOf(src, ret.Return.Offset)
		ops := ret.Results
		if len(ops) == 0 {
			conforms, badLine = false, line // bare `return` in a (T, error) fn
			return
		}
		// Already-idiomatic Result.Err(...) — leave untouched, still conforming.
		if len(ops) == 1 {
			if call, ok := ops[0].(*ast.CallExpr); ok && isSelector(call.Fun, "Result", "Err") {
				return
			}
			conforms, badLine = false, line
			return
		}
		if len(ops) != 2 { // exactly one value before the trailing err
			conforms, badLine = false, line
			return
		}
		last := ops[1]
		value := nodeText(src, ops[0])
		switch {
		case identName(last) == "nil": // success: return v, nil -> return Result.Ok(v)
			reps = append(reps, scan.Replacement{
				Start: ops[0].Pos().Offset, End: ops[1].End().Offset,
				Text: "Result.Ok(" + value + ")",
			})
		case isIdentExpr(last) && value == analyze.ZeroLit(successT, decls, 0):
			// bare propagation: return zero, err — left for fixPropagate.
		default:
			conforms, badLine = false, line
		}
	})
	return reps, conforms, badLine
}

// isIdentExpr reports whether e is a bare identifier (the error variable of a propagation).
func isIdentExpr(e ast.Expr) bool {
	_, ok := e.(*ast.Ident)
	return ok
}

// walkReturns calls f for every return statement in block that is not nested inside a
// function literal — descent stops at expression boundaries, so a closure's returns (which
// belong to that closure, not the enclosing function) are never visited.
func walkReturns(block *ast.BlockStmt, f func(*ast.ReturnStmt)) {
	if block == nil {
		return
	}
	var visitStmt func(ast.Stmt)
	visitList := func(ss []ast.Stmt) {
		for _, s := range ss {
			visitStmt(s)
		}
	}
	visitStmt = func(s ast.Stmt) {
		switch s := s.(type) {
		case *ast.ReturnStmt:
			f(s)
		case *ast.BlockStmt:
			visitList(s.List)
		case *ast.IfStmt:
			if s.Init != nil {
				visitStmt(s.Init)
			}
			if s.Body != nil {
				visitList(s.Body.List)
			}
			if s.Else != nil {
				visitStmt(s.Else)
			}
		case *ast.ForStmt:
			if s.Body != nil {
				visitList(s.Body.List)
			}
		case *ast.RangeStmt:
			if s.Body != nil {
				visitList(s.Body.List)
			}
		case *ast.SwitchStmt:
			if s.Body != nil {
				visitList(s.Body.List)
			}
		case *ast.CaseClause:
			visitList(s.Body)
		}
	}
	visitList(block.List)
}

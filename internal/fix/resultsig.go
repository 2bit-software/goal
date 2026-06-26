package fix

import (
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
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
func fixResultSig(src string, toks []scan.Token, t *analyze.Tables, changes *[]Change, reports *[]Report) []scan.Replacement {
	funcs := scan.ScanFuncs(toks)
	var reps []scan.Replacement
	for _, f := range funcs {
		if f.Name == "" {
			continue
		}
		if sig, ok := t.FuncSignatures[f.Name]; !ok || sig.Mode != analyze.ModeNone {
			continue // only plain functions; Result/Option are already idiomatic
		}
		// scan.ParamsClose skips a bracketed return type (Result[T, error]) but not a
		// parenthesized one ((T, error)) — exactly the shape fix targets — so locate the
		// parameter list's close paren directly from the name.
		pc := paramsClose(toks, f)
		if pc < 0 {
			continue
		}
		// The return must be a `(T, error)` tuple with exactly one non-error value.
		if pc+1 >= len(toks) || toks[pc+1].Text != "(" {
			if returnsBareError(toks, pc, f.BodyOpen) {
				*reports = append(*reports, Report{lineOf(src, toks[f.NameTok].Start), Skip, "result-sig",
					"`" + f.Name + "` returns a bare `error`; not auto-converted to Result"})
			}
			continue
		}
		rc := scan.MatchParen(toks, pc+1)
		comma := scan.TopLevelComma(toks, pc+1, rc)
		if comma < 0 {
			continue // single parenthesized return type, not a tuple
		}
		if scan.TopLevelComma(toks, comma, rc) >= 0 {
			*reports = append(*reports, Report{lineOf(src, toks[f.NameTok].Start), Skip, "result-sig",
				"`" + f.Name + "` returns multiple non-error values; not auto-converted to Result"})
			continue
		}
		if strings.TrimSpace(src[toks[comma+1].Start:toks[rc].Start]) != "error" {
			continue
		}
		successT := strings.TrimSpace(src[toks[pc+2].Start:toks[comma].Start])

		// Every return in the body (excluding nested function literals) must be a
		// recognized success or bare propagation, or we abandon the whole function.
		nested := nestedFuncRanges(toks, funcs, f)
		successReps, conforms, badLine := classifyReturns(src, toks, f, nested, successT, t)
		if !conforms {
			*reports = append(*reports, Report{badLine, Skip, "result-sig",
				"`" + f.Name + "` has a non-propagating return; not auto-converted to Result"})
			continue
		}

		reps = append(reps, scan.Replacement{
			Start: toks[pc+1].Start, End: toks[rc].End,
			Text: "Result[" + successT + ", error]",
		})
		reps = append(reps, successReps...)
		*changes = append(*changes, Change{lineOf(src, toks[f.NameTok].Start), "result-sig"})
		if isExported(f.Name) {
			*reports = append(*reports, Report{lineOf(src, toks[f.NameTok].Start), Warn, "result-sig",
				"exported `" + f.Name + "` changed to Result[" + successT + ", error]; callers outside the scanned path may need manual updates"})
		}
	}
	return reps
}

// classifyReturns inspects every top-level return of f (skipping nested function literals)
// and returns the replacements that rewrite each `return v, nil` success into
// `return Result.Ok(v)`. conforms is false (with the offending line) if any return is
// neither a recognized success nor a bare propagation that fixPropagate will collapse.
func classifyReturns(src string, toks []scan.Token, f scan.Func, nested [][2]int, successT string, t *analyze.Tables) (reps []scan.Replacement, conforms bool, badLine int) {
	for k := f.BodyOpen + 1; k < f.BodyClose; k++ {
		if toks[k].Text != "return" || !scan.IsLineStart(src, toks[k].Start) {
			continue
		}
		if inAnyByteRange(toks[k].Start, nested) {
			continue // belongs to a nested closure, not f
		}
		ops, opEnd := returnOperands(src, toks, k, f.BodyClose)
		if len(ops) == 0 {
			return nil, false, lineOf(src, toks[k].Start) // bare `return` in a (T,error) fn
		}
		// Already-idiomatic Result.Err(...) — leave untouched, still conforming.
		if len(ops) >= 4 && ops[0].Text == "Result" && ops[1].Text == "." && ops[2].Text == "Err" {
			continue
		}
		comma := topLevelComma(ops)
		if comma < 0 || comma != len(ops)-2 {
			return nil, false, lineOf(src, toks[k].Start)
		}
		last := ops[len(ops)-1]
		value := strings.TrimSpace(src[ops[0].Start:ops[comma].Start])
		switch {
		case last.Text == "nil": // success: return v, nil -> return Result.Ok(v)
			reps = append(reps, scan.Replacement{
				Start: ops[0].Start, End: opEnd,
				Text: "Result.Ok(" + value + ")",
			})
		case scan.IsIdent(last.Text) && value == analyze.ZeroLit(successT, t.TypeDecls, 0):
			// bare propagation: return zero, err — left for fixPropagate.
		default:
			return nil, false, lineOf(src, toks[k].Start)
		}
	}
	return reps, true, 0
}

// returnOperands returns the tokens of the return statement starting at index k (the
// operands after `return`, up to the end of its line) and the byte offset where they end.
func returnOperands(src string, toks []scan.Token, k, bodyClose int) (ops []scan.Token, opEnd int) {
	lineEnd := scan.NextNewline(src, toks[k].End)
	for j := k + 1; j < bodyClose && toks[j].Start < lineEnd; j++ {
		ops = append(ops, toks[j])
	}
	if len(ops) == 0 {
		return nil, toks[k].End
	}
	return ops, ops[len(ops)-1].End
}

// nestedFuncRanges returns the byte ranges of function literals nested inside f, so returns
// belonging to a closure are not mistaken for f's own.
func nestedFuncRanges(toks []scan.Token, funcs []scan.Func, f scan.Func) [][2]int {
	var ranges [][2]int
	for _, g := range funcs {
		if g.BodyOpen == f.BodyOpen {
			continue // f itself (body brace index identifies it)
		}
		if g.BodyOpen > f.BodyOpen && g.BodyClose <= f.BodyClose {
			ranges = append(ranges, [2]int{toks[g.BodyOpen].Start, toks[g.BodyClose].End})
		}
	}
	return ranges
}

// inAnyByteRange reports whether byteOff falls inside any [lo, hi) byte range.
func inAnyByteRange(byteOff int, ranges [][2]int) bool {
	for _, r := range ranges {
		if byteOff >= r[0] && byteOff < r[1] {
			return true
		}
	}
	return false
}

// paramsClose returns the token index of the `)` closing f's parameter list, located by
// walking forward from the name past an optional type-parameter list `[...]` to the params
// `(` and matching it. Unlike scan.ParamsClose this is correct when the return type is a
// parenthesized tuple `(T, error)`, the shape `goal fix` converts.
func paramsClose(toks []scan.Token, f scan.Func) int {
	k := f.NameTok + 1
	if k < len(toks) && toks[k].Text == "[" { // generic type parameters
		k = scan.MatchBracket(toks, k) + 1
	}
	if k >= len(toks) || toks[k].Text != "(" {
		return -1
	}
	return scan.MatchParen(toks, k)
}

// returnsBareError reports whether the function's single return type is `error`.
func returnsBareError(toks []scan.Token, pc, bodyOpen int) bool {
	return pc+1 < bodyOpen && pc+1 < len(toks) && toks[pc+1].Text == "error"
}

// isExported reports whether name begins with an uppercase letter (a Go exported symbol).
func isExported(name string) bool {
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}

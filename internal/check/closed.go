package check

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// checkClosed enforces feature 06 (closed-E Result): for a closed-E `Result[T, E]`
// (E is a named sum, not the builtin error), the error side must stay closed — every
// error value flowing into it must be a known variant of E — and any `?` propagation
// across error types must have a total `from func` conversion registered. A `?` whose
// From-conversion is missing, or an Err of a type outside E, is an Error.
//
// Reuse, not reinvention:
//   - analyze.Tables.FuncSignatures gives each function's Mode (ModeResultClosed) and
//     its T/E. This check rebuilds the per-function body spans the way the closed pass
//     does (internal/pass/closed.go via funcSpans/sigAt): re-lex, scan.ScanFuncs for the
//     bodies, look the signature up by name, and pair an offset to its enclosing sig.
//   - The closed pass shows the two sites this guarantee covers:
//     lowerClosedQuestions pairs a `?` to its caller's E and looks the callee's E up,
//     consulting FromRegistry[[2]string{callee.E, caller.E}] when they differ;
//     lowerClosedCtors locates each `Result.Err(X)` inside a closed-E function. This
//     check mirrors both walks but asserts the registry/variant membership instead of
//     splicing.
//   - analyze.Tables.FromRegistry is the (src, target) -> ConvEntry conversion table;
//     totality is "every E' that reaches a `?` into an E-returning function has a
//     registry entry to that E."
//   - analyze.Tables.Enums[E].VSet is the closed variant set for the Err side: an
//     `Result.Err(E.Variant)` is closed iff E is the enclosing function's error enum and
//     Variant is in its VSet.
//   - Must run pre-lowering: the closed pass rewrites `?` to a type switch and
//     `Result.Err(X)` to `Err[T,E]{Value: X}`, erasing both constructs.
//
// Defer-boundary (emit a located Warning, never a false Error):
//   - A `?` whose callee is not an in-file closed-E Result function (callee unknown, or
//     a non-direct-call scrutinee whose return type can't be read lexically): the
//     propagated error type is unresolvable, so From-totality can't be proven — deferred
//     (`unresolved-question-error`).
//   - A closed-E function whose error enum E is not an in-file enum: its variant set is
//     unknown, so closedness of its `Result.Err(...)` can't be checked — deferred
//     (`unresolved-error-enum`).
//   - A `Result.Err(X)` whose X is not a lexically-resolvable variant construction
//     `E.Variant` (a bound variable, a call, an already-typed value): the concrete error
//     type isn't readable here — deferred (`unresolved-err-value`).
//
// No analyze.Tables extension was needed: FuncSignatures, FromRegistry, and Enums carry
// every fact this guarantee reads.
func checkClosed(src string, t *analyze.Tables) ([]Diagnostic, error) {
	toks := scan.Lex(src)
	spans := closedSpans(toks, t)
	var diags []Diagnostic
	diags = append(diags, checkClosedQuestions(src, toks, t, spans)...)
	diags = append(diags, checkClosedErrs(src, toks, t, spans)...)
	return diags, nil
}

// closedFuncSpan pairs a function body's byte span with its analyzed signature, so an
// offset can be mapped to the enclosing function's mode and E. Mirrors the pass package's
// funcSpan/funcSpans (not importable from check).
type closedFuncSpan struct {
	lo, hi int
	sig    analyze.FuncSig
}

// closedSpans returns one span per function carrying its analyzed signature, the way
// pass.funcSpans does. Functions without a recorded signature are omitted.
func closedSpans(toks []scan.Token, t *analyze.Tables) []closedFuncSpan {
	var spans []closedFuncSpan
	for _, f := range scan.ScanFuncs(toks) {
		if sig, ok := t.FuncSignatures[f.Name]; ok {
			spans = append(spans, closedFuncSpan{lo: toks[f.BodyOpen].Start, hi: toks[f.BodyClose].End, sig: sig})
		}
	}
	return spans
}

// sigAtOffset returns the signature of the function whose body contains byte offset off.
func sigAtOffset(spans []closedFuncSpan, off int) (analyze.FuncSig, bool) {
	for _, s := range spans {
		if off >= s.lo && off < s.hi {
			return s.sig, true
		}
	}
	return analyze.FuncSig{}, false
}

// checkClosedQuestions verifies From-totality at each `?` inside a closed-E function: when
// the callee's error enum differs from the caller's, a `from func` for that pair must be
// registered. Mirrors lowerClosedQuestions's caller/callee pairing (internal/pass/closed.go).
func checkClosedQuestions(src string, toks []scan.Token, t *analyze.Tables, spans []closedFuncSpan) []Diagnostic {
	var diags []Diagnostic
	for q := range toks {
		if toks[q].Text != "?" {
			continue
		}
		p := toks[q].Start
		caller, ok := sigAtOffset(spans, p)
		if !ok || caller.Mode != analyze.ModeResultClosed {
			continue // open-E `?` (feature 03) / non-Result `?` — not this guarantee's concern
		}
		lineStart := strings.LastIndexByte(src[:p], '\n') + 1
		_, rhs, isAssign := scan.SplitAssign(src[lineStart:p])
		if !isAssign {
			// Malformed (`?` not the RHS of an assignment): the lowering rejects it; the
			// concrete propagated type isn't readable, so defer rather than assert.
			diags = append(diags, deferQuestion(toks[q].Start, "`?` is not the right-hand side of an assignment (`name := expr?`)"))
			continue
		}
		calleeName := scan.LeadIdent(rhs)
		callee, known := t.FuncSignatures[calleeName]
		if !known || callee.Mode != analyze.ModeResultClosed {
			// The scrutinee is not a direct call to an in-file closed-E Result function,
			// so the propagated error enum can't be read lexically. Defer.
			diags = append(diags, deferQuestion(toks[q].Start,
				fmt.Sprintf("callee `%s` is not an in-file closed-E `Result`-returning function (its error type is unresolvable here)", calleeNameOrExpr(calleeName, rhs))))
			continue
		}
		if callee.E == caller.E {
			continue // same error enum — passes through, no conversion needed
		}
		if _, found := t.FromRegistry[[2]string{callee.E, caller.E}]; !found {
			diags = append(diags, Diagnostic{
				Pos:      toks[q].Start,
				Severity: Error,
				Feature:  "06-error-e",
				Code:     "missing-from-conversion",
				Message: fmt.Sprintf("`?` propagates `%s` into a `Result[_, %s]` function but no `from func` converts `%s` to `%s` — declare `from func name(e %s) %s { … }`",
					callee.E, caller.E, callee.E, caller.E, callee.E, caller.E),
			})
		}
	}
	return diags
}

// checkClosedErrs verifies closedness at each `Result.Err(X)` inside a closed-E function:
// X must be a variant construction `E.Variant` of the enclosing function's error enum E.
// An Err of a foreign enum or an unknown variant is an Error; an X that isn't a
// lexically-resolvable variant construction is deferred. Mirrors lowerClosedCtors's
// `Result . Err (` location (internal/pass/closed.go).
func checkClosedErrs(src string, toks []scan.Token, t *analyze.Tables, spans []closedFuncSpan) []Diagnostic {
	var diags []Diagnostic
	funcs := scan.ScanFuncs(toks)
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Text != "Result" || toks[i+1].Text != "." || toks[i+2].Text != "Err" || toks[i+3].Text != "(" {
			continue
		}
		caller, ok := sigAtOffset(spans, toks[i].Start)
		if !ok || caller.Mode != analyze.ModeResultClosed {
			i += 3
			continue // not inside a closed-E function — open-E Err is a native return, not ours
		}
		closeIdx := scan.MatchParen(toks, i+3)
		pos := toks[i].Start

		// A match-arm pattern `Result.Err(b) =>` destructures the scrutinee; it constructs
		// nothing, so it has no closedness to verify. Skip it (the match lowering, not this
		// check, owns arm patterns — mirrors lowerClosedCtors skipping in-match occurrences).
		if isMatchArmPattern(toks, closeIdx) {
			i = closeIdx
			continue
		}

		enum, known := t.Enums[caller.E]
		if !known {
			// The enclosing function's error enum is not declared in this file: its
			// variant set is unknown, so closedness can't be checked. Defer.
			diags = append(diags, Diagnostic{
				Pos:      pos,
				Severity: Warning,
				Feature:  "06-error-e",
				Code:     "unresolved-error-enum",
				Message: fmt.Sprintf("cannot verify closedness of `Result.Err` in a `Result[_, %s]` function: enum `%s` is not declared in this file — closedness deferred",
					caller.E, caller.E),
			})
			i = closeIdx
			continue
		}

		qual, variant, isVariant := errVariant(toks, i+3, closeIdx)
		switch {
		case !isVariant:
			// A passthrough re-wrap `return Result.Err(e)`, where e is provably the function's
			// error type E (a same-E match binding, or a parameter/var typed E), keeps the sum
			// closed over E — no diagnostic.
			if isClosedPassthrough(src, toks, t, caller, funcs, i, closeIdx) {
				break
			}
			// X isn't a `E.Variant` construction (a bound var, a call, …): its concrete
			// error type isn't readable lexically. Defer.
			diags = append(diags, Diagnostic{
				Pos:      pos,
				Severity: Warning,
				Feature:  "06-error-e",
				Code:     "unresolved-err-value",
				Message: fmt.Sprintf("cannot verify the `Result.Err` value is a variant of `%s`: its argument is not a lexically-resolvable `%s.Variant` construction — closedness deferred",
					caller.E, caller.E),
			})
		case qual != caller.E:
			// Err of a different enum than the function's declared E — the error side is
			// not closed over E.
			diags = append(diags, Diagnostic{
				Pos:      pos,
				Severity: Error,
				Feature:  "06-error-e",
				Code:     "err-outside-closed-enum",
				Message: fmt.Sprintf("`Result.Err(%s.%s)` escapes the closed error type: this function's error enum is `%s`, not `%s` — wrap or convert the value to a `%s` variant",
					qual, variant, caller.E, qual, caller.E),
			})
		case !enum.VSet[variant]:
			// Right enum, but a name that is not one of its variants.
			diags = append(diags, Diagnostic{
				Pos:      pos,
				Severity: Error,
				Feature:  "06-error-e",
				Code:     "unknown-error-variant",
				Message: fmt.Sprintf("`Result.Err(%s.%s)` names `%s`, which is not a variant of enum `%s` — its variants are %s",
					qual, variant, variant, caller.E, variantList(enum)),
			})
		}
		i = closeIdx
	}
	return diags
}

// errVariant reads the single argument of a `Result.Err(...)` whose "(" is toks[open] and
// ")" is toks[close]. It reports a variant construction `Qual.Variant` (the qualifier and
// variant name) only when the argument is exactly `IDENT . IDENT` — optionally followed by
// a payload "(" (a data-bearing variant `E.V(field: x)`). Anything else (a bare name, a
// call, a longer expression) is not a lexically-resolvable variant construction.
func errVariant(toks []scan.Token, open, close int) (qual, variant string, ok bool) {
	// Argument tokens are (open, close) exclusive.
	lo, hi := open+1, close
	if lo+2 >= hi {
		return "", "", false // too short to be `IDENT . IDENT`
	}
	if !scan.IsIdent(toks[lo].Text) || toks[lo+1].Text != "." || !scan.IsIdent(toks[lo+2].Text) {
		return "", "", false
	}
	// What follows `IDENT . IDENT` must be either the end of the argument, or a payload
	// paren that spans to the end — otherwise the argument is a larger expression we can't
	// reduce to one variant (e.g. `E.V.Other`, `f(E.V)`, `E.V + x`).
	rest := lo + 3
	if rest == hi {
		return toks[lo].Text, toks[lo+2].Text, true // bare `E.Variant`
	}
	if toks[rest].Text == "(" && scan.MatchParen(toks, rest) == hi-1 {
		return toks[lo].Text, toks[lo+2].Text, true // `E.Variant(payload…)`
	}
	return "", "", false
}

// isMatchArmPattern reports whether the `Result.Err(...)` whose ")" is at closeIdx is a
// match-arm pattern — i.e. immediately followed by the `=>` arrow (lexed as `=` `>`) —
// rather than a constructor call. A pattern binds the error out of the scrutinee; it does
// not build one, so it carries no closedness obligation.
func isMatchArmPattern(toks []scan.Token, closeIdx int) bool {
	return closeIdx >= 0 && closeIdx+2 < len(toks) &&
		toks[closeIdx+1].Text == "=" && toks[closeIdx+2].Text == ">"
}

// isClosedPassthrough reports whether `Result.Err(arg)` (arg spanning the parens at
// toks[open]..toks[close]) re-wraps a value that is provably the enclosing function's error
// type E, so the sum stays closed over E and needs no diagnostic. arg qualifies when it is a
// single identifier that is either:
//   - a parameter or `var` of the enclosing function declared with type E, or
//   - bound by an enclosing `match`'s `Err` arm whose scrutinee is a direct call to a
//     closed-E Result function with the SAME error enum (resolution mirrors
//     checkClosedQuestions: only direct in-file calls resolve).
//
// Anything unprovable lexically (a foreign-typed binding, a method-call scrutinee, a longer
// expression) returns false and stays deferred.
func isClosedPassthrough(src string, toks []scan.Token, t *analyze.Tables, caller analyze.FuncSig, funcs []scan.Func, open, close int) bool {
	arg, ok := singleIdentArg(toks, open+3, close)
	if !ok {
		return false
	}
	// arg is the enclosing function's own error type E (a parameter or typed local).
	if fn, ok := enclosingFunc(funcs, open); ok &&
		(paramTypedAs(toks, fn, arg, caller.E) || varTypedAs(toks, fn.BodyOpen, fn.BodyClose, arg, caller.E)) {
		return true
	}
	// Walk enclosing matches outward from the construction; the binding may come from the
	// directly enclosing Err arm or an outer one. Suppress only when one is confirmed.
	for mi := open - 1; mi >= 0; mi-- {
		if toks[mi].Text != "match" {
			continue
		}
		bo := scan.MatchBodyBrace(toks, mi)
		if bo < 0 || bo >= open {
			continue // not this match's arm block, or it opens after the construction
		}
		bc := scan.MatchBrace(toks, bo)
		if bc <= close {
			continue // the construction is not inside this match's arm block
		}
		if matchErrArmBinds(toks, bo, bc, arg) && scrutineeIsSameClosedE(src, toks, t, caller, mi, bo) {
			return true
		}
	}
	return false
}

// singleIdentArg returns the lone identifier argument between the parens at toks[open]
// ("(") and toks[close] (")"), reporting false when the argument is empty, several tokens,
// or not an identifier.
func singleIdentArg(toks []scan.Token, open, close int) (string, bool) {
	if close == open+2 && scan.IsIdent(toks[open+1].Text) {
		return toks[open+1].Text, true
	}
	return "", false
}

// enclosingFunc returns the function whose body token range (BodyOpen, BodyClose) contains
// token index i.
func enclosingFunc(funcs []scan.Func, i int) (scan.Func, bool) {
	for _, f := range funcs {
		if f.BodyOpen < i && i < f.BodyClose {
			return f, true
		}
	}
	return scan.Func{}, false
}

// paramTypedAs reports whether function fn declares a parameter named name with the single
// identifier type typ (e.g. `e ProvisionError`). It reads the parameter list between the "("
// after the name and fn.ParamsClose, matching `name typ` immediately before a "," or the
// closing ")", which also covers the trailing member of a grouped parameter (`a, e E`).
func paramTypedAs(toks []scan.Token, fn scan.Func, name, typ string) bool {
	open := fn.NameTok + 1
	if open >= len(toks) || toks[open].Text != "(" || fn.ParamsClose <= open {
		return false
	}
	for k := open + 1; k+1 < fn.ParamsClose; k++ {
		if toks[k].Text == name && toks[k+1].Text == typ &&
			(k+2 == fn.ParamsClose || toks[k+2].Text == ",") {
			return true
		}
	}
	return false
}

// varTypedAs reports whether the token range [lo, hi) declares `var name typ` — a local
// variable explicitly typed as typ.
func varTypedAs(toks []scan.Token, lo, hi int, name, typ string) bool {
	for k := lo; k+2 < hi; k++ {
		if toks[k].Text == "var" && toks[k+1].Text == name && toks[k+2].Text == typ {
			return true
		}
	}
	return false
}

// matchErrArmBinds reports whether the match arm block (its "{" at bo, "}" at bc) has an
// `Err` arm pattern `Result.Err(arg) =>` that binds the identifier arg.
func matchErrArmBinds(toks []scan.Token, bo, bc int, arg string) bool {
	for j := bo + 1; j+7 < bc; j++ {
		if toks[j].Text == "Result" && toks[j+1].Text == "." && toks[j+2].Text == "Err" &&
			toks[j+3].Text == "(" && toks[j+4].Text == arg && toks[j+5].Text == ")" &&
			toks[j+6].Text == "=" && toks[j+7].Text == ">" {
			return true
		}
	}
	return false
}

// scrutineeIsSameClosedE reports whether the match at toks[mi] (arm block "{" at bo) has a
// scrutinee that is a direct call to a closed-E Result function whose error enum equals the
// enclosing caller's. Mirrors checkClosedQuestions's lexical scrutinee resolution.
func scrutineeIsSameClosedE(src string, toks []scan.Token, t *analyze.Tables, caller analyze.FuncSig, mi, bo int) bool {
	scrut := strings.TrimSpace(src[toks[mi].End:toks[bo].Start])
	callee, ok := t.FuncSignatures[scan.LeadIdent(scrut)]
	return ok && callee.Mode == analyze.ModeResultClosed && callee.E == caller.E
}

// deferQuestion builds the located Warning for a `?` whose propagated error type cannot be
// resolved lexically (From-totality unprovable here).
func deferQuestion(pos int, reason string) Diagnostic {
	return Diagnostic{
		Pos:      pos,
		Severity: Warning,
		Feature:  "06-error-e",
		Code:     "unresolved-question-error",
		Message: fmt.Sprintf("cannot verify the `?` error conversion: %s — From-totality deferred", reason),
	}
}

// calleeNameOrExpr renders the callee for a deferral message: its leading identifier when
// present, else the trimmed scrutinee expression.
func calleeNameOrExpr(name, rhs string) string {
	if name != "" {
		return name
	}
	return strings.TrimSpace(rhs)
}

// variantList renders an enum's declared variants as a comma-separated, backtick-quoted,
// qualified list in declaration order, for the unknown-variant diagnostic.
func variantList(enum *analyze.Enum) string {
	names := make([]string, len(enum.Variants))
	for i, v := range enum.Variants {
		names[i] = "`" + enum.Name + "." + v.Name + "`"
	}
	return strings.Join(names, ", ")
}

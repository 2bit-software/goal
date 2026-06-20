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
	diags = append(diags, checkClosedErrs(toks, t, spans)...)
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
func checkClosedErrs(toks []scan.Token, t *analyze.Tables, spans []closedFuncSpan) []Diagnostic {
	var diags []Diagnostic
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

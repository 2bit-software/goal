package fix

import (
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
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
func fixPropagate(src string, toks []scan.Token, spans []analyze.FuncSpan, t *analyze.Tables, changes *[]Change, reports *[]Report) []scan.Replacement {
	var reps []scan.Replacement
	for i := range toks {
		if toks[i].Text != "if" || !scan.IsLineStart(src, toks[i].Start) {
			continue
		}
		sig, ok := analyze.SigAt(spans, toks[i].Start)
		if !ok || (sig.Mode != analyze.ModeResult && sig.Mode != analyze.ModeOption) {
			continue
		}
		// The lexer splits multi-char operators, so `if err != nil {` is the token run
		// if · err · ! · = · nil · {.
		if i+5 >= len(toks) || toks[i+5].Text != "{" {
			continue
		}
		condVar := toks[i+1].Text
		op := toks[i+2].Text + toks[i+3].Text
		isResult := sig.Mode == analyze.ModeResult
		if (isResult && op != "!=") || (!isResult && op != "==") {
			continue
		}
		if toks[i+4].Text != "nil" || !scan.IsIdent(condVar) {
			continue
		}
		braceOpen := i + 5
		braceClose := scan.MatchBrace(toks, braceOpen)
		if braceClose+1 < len(toks) && toks[braceClose+1].Text == "else" {
			continue // not a clean early return
		}
		// The body must be exactly one propagation return.
		if !validPropagationReturn(src, toks, braceOpen+1, braceClose, isResult, condVar, sig.T, t) {
			continue
		}

		// Locate the binding statement on the line immediately above the `if`.
		ifLineStart := lineStartBefore(src, toks[i].Start)
		if ifLineStart == 0 {
			continue
		}
		bindLineEnd := ifLineStart - 1
		bindLineStart := lineStartBefore(src, bindLineEnd)
		bindText := src[bindLineStart:bindLineEnd]
		name, rhs, isAssign := scan.SplitAssign(bindText)
		if !isAssign || rhs == "" {
			continue
		}
		value, valid := propagationLHS(name, condVar, isResult)
		if !valid {
			continue
		}
		// Safety guards: never drop a comment, never collapse a multi-line binding.
		if spanHasComment(src, bindLineStart, toks[braceClose].End) {
			*reports = append(*reports, Report{lineOf(src, toks[i].Start), Skip, "propagate",
				"propagation block has a comment; left as-is to avoid dropping it"})
			continue
		}
		if strings.Contains(strings.TrimRight(strings.TrimLeft(bindText, " \t"), " \t"), "\n") {
			continue // defensive: a multi-line binding cannot reach here, but never collapse one
		}

		indent := indentOf(src, bindLineStart)
		reps = append(reps, scan.Replacement{
			Start: bindLineStart,
			End:   toks[braceClose].End,
			Text:  indent + value + " := " + rhs + "?",
		})
		*changes = append(*changes, Change{lineOf(src, toks[i].Start), "propagate"})

		// Option: the value is now the unwrapped T, so rewrite later `*o` uses to `o`.
		if !isResult {
			derefReps, derefOK := optionDerefRewrites(toks, braceClose+1, sig, condVar, spans)
			if !derefOK {
				// The pointer escapes (used other than as `*o`); abandon this collapse.
				reps = reps[:len(reps)-1]
				*changes = (*changes)[:len(*changes)-1]
				*reports = append(*reports, Report{lineOf(src, toks[i].Start), Skip, "propagate",
					"Option value used other than `*" + condVar + "`; left as-is"})
				continue
			}
			reps = append(reps, derefReps...)
		}
	}
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
func fixPropagateInit(src string, toks []scan.Token, spans []analyze.FuncSpan, t *analyze.Tables, changes *[]Change, reports *[]Report) []scan.Replacement {
	var reps []scan.Replacement
	for i := range toks {
		if toks[i].Text != "if" || !scan.IsLineStart(src, toks[i].Start) {
			continue
		}
		sig, ok := analyze.SigAt(spans, toks[i].Start)
		if !ok || sig.Mode != analyze.ModeResult {
			continue // `?` only propagates the error of an open-E Result function
		}
		braceOpen := ifBodyBrace(toks, i)
		if braceOpen < 0 {
			continue
		}
		// An init clause is what distinguishes this shape from the one fixPropagate handles:
		// a top-level `;` between the `if` and its body brace. No `;` means there is no init
		// clause — leave it to fixPropagate.
		semi := topLevelSemicolon(toks, i+1, braceOpen)
		if semi < 0 {
			continue
		}
		// The condition must be exactly `condVar != nil` (operators lex char by char).
		cond := toks[semi+1 : braceOpen]
		if len(cond) != 4 || !scan.IsIdent(cond[0].Text) ||
			cond[1].Text+cond[2].Text != "!=" || cond[3].Text != "nil" {
			continue
		}
		condVar := cond[0].Text
		braceClose := scan.MatchBrace(toks, braceOpen)
		if braceClose+1 < len(toks) && toks[braceClose+1].Text == "else" {
			continue // not a clean early return
		}
		if !validPropagationReturn(src, toks, braceOpen+1, braceClose, true, condVar, sig.T, t) {
			continue
		}
		// The init clause must be `condVar := CALL` — only the error is bound, so the call's
		// sole output is the error and a bare `CALL?` discards nothing.
		initText := src[toks[i+1].Start:toks[semi].Start]
		name, rhs, isAssign := scan.SplitAssign(initText)
		if !isAssign || rhs == "" || name != condVar {
			continue
		}
		if spanHasComment(src, toks[i].Start, toks[braceClose].End) {
			*reports = append(*reports, Report{lineOf(src, toks[i].Start), Skip, "propagate",
				"propagation block has a comment; left as-is to avoid dropping it"})
			continue
		}

		lineStart := lineStartBefore(src, toks[i].Start)
		reps = append(reps, scan.Replacement{
			Start: lineStart,
			End:   toks[braceClose].End,
			Text:  indentOf(src, lineStart) + rhs + "?",
		})
		*changes = append(*changes, Change{lineOf(src, toks[i].Start), "propagate"})
	}
	return reps
}

// ifBodyBrace returns the token index of the `{` opening an if statement's body: the first `{`
// at paren/bracket depth 0 after the `if` at ifIdx (a call or index in the init clause sits
// inside parens/brackets, so its delimiters are not mistaken for the body). Returns -1 if none.
func ifBodyBrace(toks []scan.Token, ifIdx int) int {
	depth := 0
	for k := ifIdx + 1; k < len(toks); k++ {
		switch toks[k].Text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case "{":
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// topLevelSemicolon returns the index of the first `;` at paren/bracket/brace depth 0 in
// toks[lo:hi], or -1 if there is none (so an if statement with no init clause is recognized).
func topLevelSemicolon(toks []scan.Token, lo, hi int) int {
	depth := 0
	for k := lo; k < hi && k < len(toks); k++ {
		switch toks[k].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ";":
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// validPropagationReturn reports whether the tokens in [lo, hi) form exactly the early
// return of a propagation block: `return zero, err` / `return Result.Err(err)` for Result,
// `return Option.None` / `return nil` for Option. For Result it also requires the returned
// zero to match the success type's computed zero (so a real value is never discarded).
func validPropagationReturn(src string, toks []scan.Token, lo, hi int, isResult bool, condVar, successT string, t *analyze.Tables) bool {
	if lo >= hi || toks[lo].Text != "return" {
		return false
	}
	inner := toks[lo+1 : hi]
	if isResult {
		// return Result.Err(err)
		if len(inner) == 6 && inner[0].Text == "Result" && inner[1].Text == "." &&
			inner[2].Text == "Err" && inner[3].Text == "(" && inner[4].Text == condVar && inner[5].Text == ")" {
			return true
		}
		// return zero, err
		comma := topLevelComma(inner)
		if comma < 0 || inner[len(inner)-1].Text != condVar {
			return false
		}
		if comma != len(inner)-2 { // exactly one value before the trailing err
			return false
		}
		zeroActual := strings.TrimSpace(src[inner[0].Start:inner[comma-1].End])
		return zeroActual == analyze.ZeroLit(successT, t.TypeDecls, 0)
	}
	// Option: return Option.None | return nil
	if len(inner) == 3 && inner[0].Text == "Option" && inner[1].Text == "." && inner[2].Text == "None" {
		return true
	}
	return len(inner) == 1 && inner[0].Text == "nil"
}

// propagationLHS validates the binding's left-hand side against the condition variable and
// returns the value name to keep. For Result the LHS is `value, err` (err must be the
// condition variable); for Option it is the single pointer name.
func propagationLHS(name, condVar string, isResult bool) (value string, ok bool) {
	parts := splitTopLevel(name, ',')
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if isResult {
		if len(parts) != 2 || parts[1] != condVar {
			return "", false
		}
		return parts[0], true
	}
	if len(parts) != 1 || parts[0] != condVar {
		return "", false
	}
	return parts[0], true
}

// optionDerefRewrites returns the replacements turning each `*o` in the enclosing function
// (after offset from) into `o`, now that `o` holds the unwrapped value. ok is false if `o`
// is referenced in any other shape (bare `o`, `o.field`), which means the pointer escapes
// and the collapse must be abandoned.
func optionDerefRewrites(toks []scan.Token, from int, sig analyze.FuncSig, optVar string, spans []analyze.FuncSpan) (reps []scan.Replacement, ok bool) {
	// Determine the enclosing function's body end.
	var hi int
	for _, s := range spans {
		if s.Sig.Name == sig.Name {
			hi = s.Hi
			break
		}
	}
	for k := from; k < len(toks) && toks[k].Start < hi; k++ {
		if toks[k].Text != optVar {
			continue
		}
		if k > 0 && toks[k-1].Text == "*" {
			reps = append(reps, scan.Replacement{Start: toks[k-1].Start, End: toks[k].End, Text: optVar})
			continue
		}
		return nil, false // bare use of the pointer — escapes
	}
	return reps, true
}

// topLevelComma returns the index (within inner) of the first comma at bracket/paren depth
// 0, or -1 if there is none.
func topLevelComma(inner []scan.Token) int {
	depth := 0
	for k := range inner {
		switch inner[k].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// splitTopLevel splits s on sep at bracket/paren depth 0 (so a type like map[K]V or a call
// arg list is not split mid-expression).
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

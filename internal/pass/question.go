package pass

import (
	"fmt"
	"strconv"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Question lowers postfix `?` propagation over Result and Option (spec §3.7, §8.3).
// `?` early-returns the Err/None of the enclosing function and unwraps the Ok/Some
// otherwise. It appears as `name := expr?` (keep the value), `_ := expr?` (discard it
// explicitly), or a standalone `expr?` statement (the binding-free discard form, for a
// call whose only output is the error); the failure is propagated in every shape.
//
// This pass runs after signature lowering, so the enclosing function's original
// return mode is no longer visible in the source (a Result has become named returns,
// an Option has become *T). The mode is recovered by function NAME from the name-
// keyed analyze.Tables — the table built from the original source survives the splice
// because lowering preserves function names. This is the composition keystone: the
// `?` pass and the signature passes meet only through the table, never through byte
// offsets.
func Question(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	spans := funcSpans(toks, t)

	var reps []scan.Replacement
	optCounter := 0
	for q := range toks {
		if toks[q].Text != "?" {
			continue
		}
		p := toks[q].Start
		sig, _ := sigAt(spans, p)
		if sig.Mode == analyze.ModeResultClosed {
			continue // closed-E `?` is lowered by the closed-E pass
		}
		lineStart := strings.LastIndexByte(src[:p], '\n') + 1
		name, rhs, ok := scan.SplitAssign(src[lineStart:p])
		if !ok && !scan.IsBareQuestionStmt(src, toks, q, lineStart) {
			return "", fmt.Errorf("`?` must be the right-hand side of an assignment (`name := expr?`) or a standalone `expr?` statement")
		}
		// A bare `expr?` statement (no `:=`) discards the unwrapped value and propagates only
		// the failure — the same lowering as `_ := expr?`.
		discard := !ok || name == "_"
		var text string
		switch sig.Mode {
		case analyze.ModeResult:
			csig, known := calleeSig(t, rhs)
			// A resolved plain/foreign callee is `?`-able in an open-E function only if it
			// returns a trailing error to propagate. Refuse a void or non-error callee rather
			// than emit a destructure that will not compile (the question check reports the
			// same at `goal check` time, with a located diagnostic).
			if known && csig.Mode == analyze.ModeNone {
				if csig.Arity == 0 {
					return "", fmt.Errorf("`?` callee `%s` returns nothing; `?` needs a callee that returns a trailing `error` (or a `Result`)", scan.CalleeKey(rhs))
				}
				if !csig.EndsInError {
					return "", fmt.Errorf("`?` callee `%s` does not return an `error` as its last result; `?` propagates an error", scan.CalleeKey(rhs))
				}
			}
			if discard {
				// Emit one blank identifier per discarded value: an error-only callee (arity 1)
				// needs none, a (value, error) callee needs one, and so on. Unresolved callees
				// keep today's two-value form.
				n := 2
				if known && csig.Arity >= 1 {
					n = csig.Arity
				}
				text = fmt.Sprintf("if %s%s := %s; %s != nil {\nreturn %s, %s\n}", strings.Repeat("_, ", n-1), errName, rhs, errName, okName, errName)
			} else {
				if known && csig.Arity != 2 {
					return "", fmt.Errorf("`?` binds a value but %s returns %d value(s); write a bare `…?` to propagate only the error", scan.CalleeKey(rhs), csig.Arity)
				}
				text = fmt.Sprintf("%s, %s := %s\nif %s != nil {\nreturn %s, %s\n}", name, errName, rhs, errName, okName, errName)
			}
		case analyze.ModeOption:
			optCounter++
			o := optBase + strconv.Itoa(optCounter)
			if discard {
				text = fmt.Sprintf("if %s := %s; %s == nil {\nreturn nil\n}", o, rhs, o)
			} else {
				text = fmt.Sprintf("%s := %s\nif %s == nil {\nreturn nil\n}\n%s := *%s", o, rhs, o, name, o)
			}
		default:
			return "", fmt.Errorf("`?` outside a Result- or Option-returning function (open-E only; closed-E `?` is feature 06)")
		}
		reps = append(reps, scan.Replacement{Start: lineStart, End: toks[q].End, Text: text})
	}
	return scan.Splice(src, 0, len(src), reps), nil
}

// calleeSig resolves the signature of the function called at the head of rhs from the analyzed
// tables (in-file by name, foreign by `alias.Func`). The bool is false when the callee cannot
// be resolved (an unknown name, or a method whose receiver type the analyzer does not infer),
// in which case the caller keeps today's two-value form.
func calleeSig(t *analyze.Tables, rhs string) (analyze.FuncSig, bool) {
	key := scan.CalleeKey(rhs)
	if key == "" {
		return analyze.FuncSig{}, false
	}
	sig, ok := t.FuncSignatures[key]
	return sig, ok
}

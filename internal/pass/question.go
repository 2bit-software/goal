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
// otherwise. It is always the RHS of an assignment: `name := expr?` keeps the value,
// `_ := expr?` discards it and propagates only the failure; a bare `expr?` is
// rejected.
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
		if !ok {
			return "", fmt.Errorf("`?` must be the right-hand side of an assignment; write `name := expr?` to keep the value or `_ := expr?` to discard it")
		}
		discard := name == "_"
		var text string
		switch sig.Mode {
		case analyze.ModeResult:
			if discard {
				text = fmt.Sprintf("if _, %s := %s; %s != nil {\nreturn %s, %s\n}", errName, rhs, errName, okName, errName)
			} else {
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

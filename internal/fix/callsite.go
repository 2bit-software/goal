package fix

import (
	"goal/internal/analyze"
	"goal/internal/scan"
)

// reportCallSites flags manual error-propagation blocks that fix cannot collapse because
// the enclosing function is not Result/Option-returning, so `?` is not legal there. Because
// reports are captured only once the rewrite has reached its fixed point, every (T, error)
// function fix could convert has already become Result-returning by the time this runs — so
// what remains are the genuinely manual sites: a propagation inside a function whose
// signature fix did not (could not) convert. Surfacing them tells the author exactly where a
// signature still needs a human decision.
func reportCallSites(src string, toks []scan.Token, spans []analyze.FuncSpan, reports *[]Report) {
	for i := range toks {
		if toks[i].Text != "if" || !scan.IsLineStart(src, toks[i].Start) {
			continue
		}
		// Shape: `if err != nil {` — the manual error guard (operators lex char by char).
		if i+5 >= len(toks) || toks[i+5].Text != "{" {
			continue
		}
		if toks[i+2].Text+toks[i+3].Text != "!=" || toks[i+4].Text != "nil" || !scan.IsIdent(toks[i+1].Text) {
			continue
		}
		sig, ok := analyze.SigAt(spans, toks[i].Start)
		if !ok || sig.Mode != analyze.ModeNone {
			continue // a Result/Option function's block was (or will be) collapsed
		}
		condVar := toks[i+1].Text
		ifLineStart := lineStartBefore(src, toks[i].Start)
		if ifLineStart == 0 {
			continue
		}
		bindText := src[lineStartBefore(src, ifLineStart-1) : ifLineStart-1]
		name, _, isAssign := scan.SplitAssign(bindText)
		if !isAssign {
			continue
		}
		parts := splitTopLevel(name, ',')
		last := parts[len(parts)-1]
		if trimSpace(last) != condVar {
			continue
		}
		*reports = append(*reports, Report{lineOf(src, toks[i].Start), Suggest, "call-site",
			"manual error handling in `" + sig.Name + "`, which is not Result-returning; convert its signature to use `?`"})
	}
}

// trimSpace trims leading/trailing ASCII spaces and tabs.
func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

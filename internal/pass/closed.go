package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// ResultPreamble is the generic sum encoding for closed-E Results (spec §8.1 / §8.3
// fork) that a closed-E program needs in scope. The closed pass injects it inline for a
// single-file transpile; the package driver emits it once per package (see
// NeedsResultPrelude and analyze.Tables.SuppressResultPrelude).
const ResultPreamble = `type Result[T, E any] interface{ isResult() }
type Ok[T, E any] struct{ Value T }
type Err[T, E any] struct{ Value E }

func (Ok[T, E]) isResult()  {}
func (Err[T, E]) isResult() {}`

// NeedsResultPrelude reports whether any function in the tables returns a closed-E
// Result, i.e. whether the program needs ResultPreamble in scope. The package driver
// uses it to decide whether to emit goal_prelude.go once for the package.
func NeedsResultPrelude(t *analyze.Tables) bool {
	for _, sig := range t.FuncSignatures {
		if sig.Mode == analyze.ModeResultClosed {
			return true
		}
	}
	return false
}

// ResultClosed lowers closed-E Results — a Result[T, E] whose E is not error (feature
// 06). Such a Result is the §8.1 sum (Ok[T,E]/Err[T,E]), not the open-E native tuple:
// the signature stays as-is (satisfied by an injected generic preamble), `Result.Ok/
// Err` become the sum constructors, `match` is a type switch, and `?` is type-switch-
// and-return with a From-conversion (declared `from func`) auto-invoked in the Err arm
// when the caller's error type differs from the callee's.
//
// It is the closed-E half of the partition with the open-E result/question passes:
// each construct is claimed by exactly one side based on the enclosing function's (or
// scrutinee callee's) mode. It runs before the enums pass so an enum error value
// inside a generated constructor is rewritten afterward.
func ResultClosed(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	spans := funcSpans(toks, t)

	if !NeedsResultPrelude(t) {
		return src, nil
	}

	var reps []scan.Replacement
	// Inject the generic sum encoding inline, unless the package driver has taken
	// responsibility for emitting it once (SuppressResultPrelude). Either way the
	// construction/match/`?` rewrites below still run.
	if !t.SuppressResultPrelude {
		off := injectOffset(src, toks)
		reps = append(reps, scan.Replacement{Start: off, End: off, Text: "\n" + ResultPreamble + "\n"})
	}

	// (The `from` modifier is stripped by the derive pass, which owns every
	// `from func` leaf — error conversions here and any-type conversions for derive.)

	// Closed-E Result match -> type switch on Ok/Err. Record the spans so the
	// constructor rewrite below skips Result.Ok/Err patterns inside a match.
	var matchSpans [][2]int
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "match" || scan.MatchQualifier(toks, i) != "Result" {
			continue
		}
		bo := scan.MatchBodyBrace(toks, i)
		scrut := strings.TrimSpace(src[toks[i].End:toks[bo].Start])
		callee := t.FuncSignatures[scan.LeadIdent(scrut)]
		if callee.Mode != analyze.ModeResultClosed {
			continue
		}
		rep, next := lowerClosedMatch(src, toks, scrut, callee, i, bo)
		matchSpans = append(matchSpans, [2]int{rep.Start, rep.End})
		reps = append(reps, rep)
		i = next
	}

	// Closed-E `?` -> type-switch-and-return with From-conversion.
	qReps, err := lowerClosedQuestions(src, toks, t, spans)
	if err != nil {
		return "", err
	}
	reps = append(reps, qReps...)

	// Closed-E `Result.Ok/Err(...)` constructors (outside a match) -> Ok/Err sum.
	reps = append(reps, lowerClosedCtors(src, toks, spans, matchSpans)...)

	return scan.Splice(src, 0, len(src), reps), nil
}

// lowerClosedMatch lowers a closed-E `match scrut { Result.Ok(b) => ...; Result.Err(b)
// => ... }` to a type switch over Ok[T,E]/Err[T,E]. bo is the arm-block brace index.
func lowerClosedMatch(src string, toks []scan.Token, scrut string, res analyze.FuncSig, mi, bo int) (scan.Replacement, int) {
	bc := scan.MatchBrace(toks, bo)
	arms := parseResultArms(toks, bo+1, bc)

	var okArm, errArm *resultArm
	for j := range arms {
		switch arms[j].variant {
		case "Ok":
			okArm = &arms[j]
		case "Err":
			errArm = &arms[j]
		}
	}
	okBody, okUse := closedArmBody(src, toks, okArm)
	errBody, errUse := closedArmBody(src, toks, errArm)

	var b strings.Builder
	if okUse || errUse {
		fmt.Fprintf(&b, "switch %s := %s.(type) {\n", evar, scrut)
	} else {
		fmt.Fprintf(&b, "switch %s.(type) {\n", scrut)
	}
	fmt.Fprintf(&b, "case Ok[%s, %s]:\n%s\n", res.T, res.E, okBody)
	fmt.Fprintf(&b, "case Err[%s, %s]:\n%s\n", res.T, res.E, errBody)
	fmt.Fprintf(&b, "default:\npanic(%q)\n}", fmt.Sprintf("unreachable: non-exhaustive Result[%s, %s] (compiler invariant violated)", res.T, res.E))
	return scan.Replacement{Start: toks[mi].Start, End: toks[bc].End, Text: b.String()}, bc + 1
}

// closedArmBody returns the lowered arm body. When the arm binds and uses its value,
// a `name := __goal_e.Value` alias is prepended. The bool reports guard-var use.
func closedArmBody(src string, toks []scan.Token, a *resultArm) (string, bool) {
	if a == nil {
		return "", false
	}
	body := bodySrc(src, toks, a.bodyLo, a.bodyHi)
	if a.binding != "" && bodyUses(toks, a.bodyLo, a.bodyHi, a.binding) {
		return fmt.Sprintf("%s := %s.Value\n%s", a.binding, evar, body), true
	}
	return body, false
}

// lowerClosedQuestions lowers each `?` inside a closed-E function.
func lowerClosedQuestions(src string, toks []scan.Token, t *analyze.Tables, spans []funcSpan) ([]scan.Replacement, error) {
	var reps []scan.Replacement
	for q := range toks {
		if toks[q].Text != "?" {
			continue
		}
		p := toks[q].Start
		caller, _ := sigAt(spans, p)
		if caller.Mode != analyze.ModeResultClosed {
			continue
		}
		lineStart := strings.LastIndexByte(src[:p], '\n') + 1
		name, rhs, ok := scan.SplitAssign(src[lineStart:p])
		if !ok {
			if !scan.IsBareQuestionStmt(src, toks, q, lineStart) {
				return nil, fmt.Errorf("`?` must be the right-hand side of an assignment (`name := expr?`) or a standalone `expr?` statement")
			}
			name = "_" // bare `expr?`: discard the unwrapped value, propagate only the failure
		}
		callee := t.FuncSignatures[scan.LeadIdent(rhs)]
		if callee.Mode != analyze.ModeResultClosed {
			return nil, fmt.Errorf("closed-E `?` needs a closed-E Result-returning callee")
		}
		errValue := evar + ".Value"
		if callee.E != caller.E {
			conv, found := t.FromRegistry[[2]string{callee.E, caller.E}]
			if !found {
				return nil, fmt.Errorf("no `from func` conversion declared for %s -> %s (required to `?` across closed error types)", callee.E, caller.E)
			}
			errValue = fmt.Sprintf("%s(%s.Value)", conv.Name, evar)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "var %s %s\n", name, callee.T)
		fmt.Fprintf(&b, "switch %s := %s.(type) {\n", evar, rhs)
		fmt.Fprintf(&b, "case Ok[%s, %s]:\n%s = %s.Value\n", callee.T, callee.E, name, evar)
		fmt.Fprintf(&b, "case Err[%s, %s]:\nreturn Err[%s, %s]{Value: %s}\n", callee.T, callee.E, caller.T, caller.E, errValue)
		fmt.Fprintf(&b, "default:\npanic(%q)\n}", fmt.Sprintf("unreachable: non-exhaustive Result[%s, %s] (compiler invariant violated)", callee.T, callee.E))
		reps = append(reps, scan.Replacement{Start: lineStart, End: toks[q].End, Text: b.String()})
	}
	return reps, nil
}

// lowerClosedCtors wraps `Result.Ok/Err(X)` in a closed-E function to the sum
// constructor Ok[T,E]{Value: X} / Err[T,E]{Value: X}, skipping occurrences inside a
// match (those are patterns the match lowering already consumed).
func lowerClosedCtors(src string, toks []scan.Token, spans []funcSpan, matchSpans [][2]int) []scan.Replacement {
	inMatch := func(off int) bool {
		for _, s := range matchSpans {
			if off >= s[0] && off < s[1] {
				return true
			}
		}
		return false
	}
	var reps []scan.Replacement
	for i := 0; i+4 < len(toks); i++ {
		if toks[i].Text != "Result" || toks[i+1].Text != "." {
			continue
		}
		variant := toks[i+2].Text
		if (variant != "Ok" && variant != "Err") || toks[i+3].Text != "(" {
			continue
		}
		if inMatch(toks[i].Start) {
			continue
		}
		res, ok := sigAt(spans, toks[i].Start)
		if !ok || res.Mode != analyze.ModeResultClosed {
			continue
		}
		closeIdx := scan.MatchParen(toks, i+3)
		reps = append(reps, scan.Replacement{Start: toks[i].Start, End: toks[i+3].End,
			Text: fmt.Sprintf("%s[%s, %s]{Value: ", variant, res.T, res.E)})
		reps = append(reps, scan.Replacement{Start: toks[closeIdx].Start, End: toks[closeIdx].End, Text: "}"})
		i = closeIdx
	}
	return reps
}

// injectOffset returns a byte offset just after the package clause / import block,
// where the generic Result encoding can be inserted (imports must precede decls).
func injectOffset(src string, toks []scan.Token) int {
	off := 0
	for i := range toks {
		switch toks[i].Text {
		case "package":
			if i+1 < len(toks) {
				if nl := strings.IndexByte(src[toks[i+1].End:], '\n'); nl >= 0 {
					off = toks[i+1].End + nl + 1
				}
			}
		case "import":
			if i+1 < len(toks) && toks[i+1].Text == "(" {
				cl := scan.MatchParen(toks, i+1)
				if nl := strings.IndexByte(src[toks[cl].End:], '\n'); nl >= 0 {
					off = toks[cl].End + nl + 1
				}
			} else if i+1 < len(toks) {
				if nl := strings.IndexByte(src[toks[i+1].End:], '\n'); nl >= 0 {
					off = toks[i+1].End + nl + 1
				}
			}
		}
	}
	return off
}

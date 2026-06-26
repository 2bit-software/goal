package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Result lowers the open-E Result[T, error] forms (spec §8.3): function signatures
// rewrite to named Go returns (__goal_ok T, __goal_err error), `return Result.Ok/Err`
// lowers to the native (T, error) pair, and a statement-position `match` on a Result
// becomes the idiomatic `if err != nil` / `else`.
//
// Scope is the immediate open-E case, exactly as feature 03's reference: value-
// position Result match and Results stored as first-class values are deferred with a
// located error. Closed-E (sum encoding) is a later pass.
func Result(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	spans := funcSpans(toks, t)
	var reps []scan.Replacement

	// Pass 1: rewrite `func ... Result[T, error] {` to named returns (open-E only;
	// closed-E signatures stay, satisfied by the closed-E pass's generic preamble).
	for i := range toks {
		if toks[i].Text == "func" {
			if rep, ok := rewriteResultSignature(src, toks, i); ok {
				reps = append(reps, rep)
			}
		}
	}

	// Pass 2: lower `return Result.Ok(X)` / `return Result.Err(X)` in open-E
	// functions. In a closed-E function these become the sum constructors instead, so
	// they are left for the closed-E pass.
	for i := 0; i+4 < len(toks); i++ {
		if toks[i].Text == "return" && toks[i+1].Text == "Result" && toks[i+2].Text == "." &&
			(toks[i+3].Text == "Ok" || toks[i+3].Text == "Err") && toks[i+4].Text == "(" {
			if sig, _ := sigAt(spans, toks[i].Start); sig.Mode != analyze.ModeResult {
				continue
			}
			closeIdx := scan.MatchParen(toks, i+4)
			inner := strings.TrimSpace(src[toks[i+4].End:toks[closeIdx].Start])
			var text string
			if toks[i+3].Text == "Ok" {
				text = inner + ", nil"
			} else {
				text = okName + ", " + inner
			}
			reps = append(reps, scan.Replacement{Start: toks[i+1].Start, End: toks[closeIdx].End, Text: text})
			i = closeIdx
		}
	}

	// Pass 3: lower statement-position `match call { Result.Ok/Err arms }`. Only
	// matches whose arms are Result patterns AND whose scrutinee calls an open-E
	// function are claimed here; closed-E Result matches, Option, and enum matches go
	// to their own passes.
	for i := 0; i < len(toks); i++ {
		if toks[i].Text == "match" && scan.MatchQualifier(toks, i) == "Result" && !calleeIsClosed(src, toks, t, i) {
			// An arm body that returns a Result is lowered here only when the enclosing
			// function is itself open-E; a closed-E enclosing function's `return Result.*`
			// is a sum construction left for the closed-E pass (which sees it as plain
			// if/else once this match is lowered).
			openE := false
			if sig, ok := sigAt(spans, toks[i].Start); ok {
				openE = sig.Mode == analyze.ModeResult
			}
			rep, next, err := lowerResultMatch(src, toks, i, openE)
			if err != nil {
				return "", err
			}
			reps = append(reps, rep)
			i = next
		}
	}

	return scan.Splice(src, 0, len(src), reps), nil
}

// calleeIsClosed reports whether the scrutinee of the match at mi is a direct call to
// a closed-E Result function (so the open-E result pass should not claim it).
func calleeIsClosed(src string, toks []scan.Token, t *analyze.Tables, mi int) bool {
	bo := scan.MatchBodyBrace(toks, mi)
	if bo < 0 {
		return false
	}
	scrut := strings.TrimSpace(src[toks[mi].End:toks[bo].Start])
	return t.FuncSignatures[scan.LeadIdent(scrut)].Mode == analyze.ModeResultClosed
}

// rewriteResultSignature rewrites a `func ... Result[T, error]` return type to named
// Go returns. The named __goal_ok return makes the Err-path zero value available
// without synthesizing a type-specific zero literal. ok is false for functions that
// do not return a Result.
func rewriteResultSignature(src string, toks []scan.Token, fi int) (scan.Replacement, bool) {
	body := scan.FirstBodyBrace(toks, fi)
	if body < 0 {
		return scan.Replacement{}, false
	}
	pc := scan.ParamsClose(toks, body)
	if pc < 0 || pc+2 >= body {
		return scan.Replacement{}, false
	}
	if toks[pc+1].Text != "Result" || toks[pc+2].Text != "[" {
		return scan.Replacement{}, false
	}
	rb := scan.MatchBracket(toks, pc+2)
	comma := scan.TopLevelComma(toks, pc+2, rb)
	if comma < 0 {
		return scan.Replacement{}, false
	}
	if e := strings.TrimSpace(src[toks[comma+1].Start:toks[rb].Start]); e != "error" {
		return scan.Replacement{}, false // closed-E signature stays as-is
	}
	tType := strings.TrimSpace(src[toks[pc+3].Start:toks[comma].Start])
	text := fmt.Sprintf("(%s %s, %s error)", okName, tType, errName)
	return scan.Replacement{Start: toks[pc+1].Start, End: toks[rb].End, Text: text}, true
}

// resultArm is one arm of a Result match: its variant, the optional payload binding,
// and the token span of its body.
type resultArm struct {
	variant string // "Ok" or "Err"
	binding string // "" if none
	bodyLo  int
	bodyHi  int
}

// lowerResultMatch lowers a statement-position `match scrut { Result.Ok(b) => ...;
// Result.Err(b) => ... }` to `v, err := scrut; if err != nil { ... } else { ... }`.
// It returns the replacement and the token index to continue scanning from. openE says
// whether the enclosing function is open-E, so an arm body's `return Result.*` is lowered
// to the (T, error) pair here rather than left for the closed-E pass.
func lowerResultMatch(src string, toks []scan.Token, mi int, openE bool) (scan.Replacement, int, error) {
	if mi > 0 {
		if p := toks[mi-1].Text; p == "return" || p == "=" {
			return scan.Replacement{}, 0, fmt.Errorf("value-position Result match is deferred; consume a Result with a statement-position match (open-E keystone, §8.3)")
		}
	}

	bo := mi + 1
	for depth := 0; ; bo++ {
		switch toks[bo].Text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case "{":
			if depth == 0 {
				goto found
			}
		}
	}
found:
	scrut := strings.TrimSpace(src[toks[mi].End:toks[bo].Start])
	bc := scan.MatchBrace(toks, bo)
	arms := parseResultArms(toks, bo+1, bc)

	var ok, errArm *resultArm
	for i := range arms {
		switch arms[i].variant {
		case "Ok":
			ok = &arms[i]
		case "Err":
			errArm = &arms[i]
		}
	}
	if ok == nil || errArm == nil {
		return scan.Replacement{}, 0, fmt.Errorf("Result match must have both Result.Ok and Result.Err arms")
	}

	okBody, okUsed := rewriteArmBody(src, toks, *ok, valName, openE)
	errBody, _ := rewriteArmBody(src, toks, *errArm, errName, openE)

	okLHS := "_"
	if okUsed {
		okLHS = valName
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s, %s := %s\n", okLHS, errName, scrut)
	fmt.Fprintf(&b, "if %s != nil {\n%s\n} else {\n%s\n}", errName, errBody, okBody)
	return scan.Replacement{Start: toks[mi].Start, End: toks[bc].End, Text: b.String()}, bc + 1, nil
}

// parseResultArms splits arm-block tokens [lo, hi) into arms, delimited by `=>`
// (lexed as `=` `>`).
func parseResultArms(toks []scan.Token, lo, hi int) []resultArm {
	var arrows []int
	for j, depth := lo, 0; j < hi; j++ {
		switch toks[j].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
		if depth == 0 && toks[j].Text == "=" && j+1 < hi && toks[j+1].Text == ">" {
			arrows = append(arrows, j)
		}
	}

	arms := make([]resultArm, len(arrows))
	for i, eq := range arrows {
		patStart := lo
		if i > 0 {
			patStart = patternStart(toks, arrows[i])
		}
		a := parseResultPattern(toks, patStart, eq)
		a.bodyLo = eq + 2
		if i+1 < len(arrows) {
			a.bodyHi = patternStart(toks, arrows[i+1])
		} else {
			a.bodyHi = hi
		}
		arms[i] = a
	}
	return arms
}

// patternStart finds where the arm pattern ending just before the arrow at eqIdx
// begins, from the token immediately before the arrow. It is shared by every
// qualified match (Result/Option/enum): `Qual.Variant`, `Qual.Variant(binding)`, and
// the bare `_` rest arm.
func patternStart(toks []scan.Token, eqIdx int) int {
	j := eqIdx - 1
	switch toks[j].Text {
	case ")":
		// Qual . Variant ( binding ) — walk back to "(" then to the qualifier.
		depth := 0
		k := j
		for ; k >= 0; k-- {
			switch toks[k].Text {
			case ")":
				depth++
			case "(":
				depth--
			}
			if depth == 0 {
				break
			}
		}
		return k - 3
	case "_":
		return j
	default:
		// Qual . Variant
		return j - 2
	}
}

// parseResultPattern reads `Result . Variant ( binding )` starting at start.
func parseResultPattern(toks []scan.Token, start, eqIdx int) resultArm {
	a := resultArm{variant: toks[start+2].Text}
	if start+3 < eqIdx && toks[start+3].Text == "(" {
		a.binding = toks[start+4].Text
	}
	return a
}

// rewriteArmBody renames the arm's payload binding to target throughout the body and,
// when lowerReturns is set, lowers the body's `return Result.Ok/Err` constructors. The
// bool reports whether the binding was referenced (so the caller knows whether to capture
// the value or discard it with `_`).
func rewriteArmBody(src string, toks []scan.Token, a resultArm, target string, lowerReturns bool) (string, bool) {
	lo, hi := a.bodyLo, a.bodyHi
	if lo >= hi {
		return "", false
	}
	used := false
	var reps []scan.Replacement
	if a.binding != "" {
		for j := lo; j < hi; j++ {
			if toks[j].Text == a.binding {
				used = true
				reps = append(reps, scan.Replacement{Start: toks[j].Start, End: toks[j].End, Text: target})
			}
		}
	}
	if lowerReturns {
		reps = append(reps, resultReturnReps(toks, lo, hi)...)
	}
	return scan.Splice(src, toks[lo].Start, toks[hi-1].End, reps), used
}

// resultReturnReps lowers every `return Result.Ok(X)` / `return Result.Err(X)` in the
// token range [lo, hi) to the open-E (T, error) pair: `return X, nil` and `return
// __goal_ok, X`. The argument tokens X are left untouched — only the `Result.Ok(`/`)`
// delimiters are rewritten — so a binding rename of identifiers inside X composes here
// without two replacements overlapping (Splice drops overlaps). Mirrors the whole-source
// Pass 2, scoped to one arm body whose constructor that pass's reps would lose to the
// enclosing match replacement.
func resultReturnReps(toks []scan.Token, lo, hi int) []scan.Replacement {
	var reps []scan.Replacement
	for i := lo; i+4 < hi; i++ {
		if toks[i].Text != "return" || toks[i+1].Text != "Result" || toks[i+2].Text != "." ||
			(toks[i+3].Text != "Ok" && toks[i+3].Text != "Err") || toks[i+4].Text != "(" {
			continue
		}
		closeIdx := scan.MatchParen(toks, i+4)
		if toks[i+3].Text == "Ok" {
			reps = append(reps, scan.Replacement{Start: toks[i+1].Start, End: toks[i+4].End, Text: ""})
			reps = append(reps, scan.Replacement{Start: toks[closeIdx].Start, End: toks[closeIdx].End, Text: ", nil"})
		} else {
			reps = append(reps, scan.Replacement{Start: toks[i+1].Start, End: toks[i+4].End, Text: okName + ", "})
			reps = append(reps, scan.Replacement{Start: toks[closeIdx].Start, End: toks[closeIdx].End, Text: ""})
		}
		i = closeIdx
	}
	return reps
}

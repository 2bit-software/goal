package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Result lowers the open-E Result[T, error] forms (spec §8.3): function signatures
// rewrite to named Go returns (__gop_ok T, __gop_err error), `return Result.Ok/Err`
// lowers to the native (T, error) pair, and a statement-position `match` on a Result
// becomes the idiomatic `if err != nil` / `else`.
//
// Scope is the immediate open-E case, exactly as feature 03's reference: value-
// position Result match and Results stored as first-class values are deferred with a
// located error. Closed-E (sum encoding) is a later pass.
func Result(src string, _ *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	var reps []scan.Replacement

	// Pass 1: rewrite `func ... Result[T, error] {` to named returns.
	for i := range toks {
		if toks[i].Text == "func" {
			if rep, ok := rewriteResultSignature(src, toks, i); ok {
				reps = append(reps, rep)
			}
		}
	}

	// Pass 2: lower `return Result.Ok(X)` / `return Result.Err(X)`.
	for i := 0; i+4 < len(toks); i++ {
		if toks[i].Text == "return" && toks[i+1].Text == "Result" && toks[i+2].Text == "." &&
			(toks[i+3].Text == "Ok" || toks[i+3].Text == "Err") && toks[i+4].Text == "(" {
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
	// matches whose arms are Result patterns are claimed here; Option/enum matches
	// are left untouched for their own passes.
	for i := 0; i < len(toks); i++ {
		if toks[i].Text == "match" && scan.MatchQualifier(toks, i) == "Result" {
			rep, next, err := lowerResultMatch(src, toks, i)
			if err != nil {
				return "", err
			}
			reps = append(reps, rep)
			i = next
		}
	}

	return scan.Splice(src, 0, len(src), reps), nil
}

// rewriteResultSignature rewrites a `func ... Result[T, error]` return type to named
// Go returns. The named __gop_ok return makes the Err-path zero value available
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
	t := strings.TrimSpace(src[toks[pc+3].Start:toks[comma].Start])
	text := fmt.Sprintf("(%s %s, %s error)", okName, t, errName)
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
// It returns the replacement and the token index to continue scanning from.
func lowerResultMatch(src string, toks []scan.Token, mi int) (scan.Replacement, int, error) {
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

	okBody, okUsed := rewriteArmBody(src, toks, *ok, valName)
	errBody, _ := rewriteArmBody(src, toks, *errArm, errName)

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

// rewriteArmBody renames the arm's payload binding to target throughout the body.
// The bool reports whether the binding was referenced (so the caller knows whether
// to capture the value or discard it with `_`).
func rewriteArmBody(src string, toks []scan.Token, a resultArm, target string) (string, bool) {
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
	return scan.Splice(src, toks[lo].Start, toks[hi-1].End, reps), used
}

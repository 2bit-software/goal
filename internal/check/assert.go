package check

import (
	"strconv"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// checkAssert enforces the static-provable subset of feature 10 (assert): an `assert`
// whose condition the checker can decide at compile time and prove false is an Error
// (a guaranteed panic), and a tautological always-true assert is reported as a
// dead-code Warning. Conditions that are not statically decidable are left entirely to
// the runtime check the lowering already emits.
//
// The feature audit deliberately *reserved* (did not build) this static subset (spec
// §4.3, §8.6; SYNTAX.md "Reserved"), refusing general Dafny-style proving. This check is
// the minimal, conservative slice of that reservation: it only constant-folds conditions
// with no free names, and only in two shapes.
//
// Reuse, not reinvention:
//   - The assert pass (internal/pass/assert.go) locates each statement: `assert` is the
//     keyword only as the first token on its line (scan.IsLineStart), the statement runs
//     to the next newline (scan.NextNewline), and the condition is everything left of the
//     first top-level comma. This check mirrors that bounding exactly, then folds the
//     condition instead of lowering it.
//
// What it proves (constant-foldable conditions only — no free identifiers):
//   - A bare boolean literal: `assert false` is a guaranteed panic → Error
//     `assert-always-false`; `assert true` is a no-op tautology → Warning
//     `assert-always-true` (dead code, the runtime check can never fire).
//   - A comparison of two integer literals: `assert 2 > 3` folds to a constant. False →
//     Error `assert-always-false`; true → Warning `assert-always-true`. Operators
//     handled: `<`, `<=`, `>`, `>=`, `==`, `!=`.
//
// Defer-boundary (emit nothing — the runtime assert stands):
//   - Any condition mentioning an identifier, a call, or a field access — i.e. anything
//     not constant. This is the by-design out-of-scope case (SYNTAX.md): a v1 assert over
//     a variable is always runtime-checked. The spec refuses general static verification,
//     so we do not theorem-prove; an undecidable condition draws no diagnostic at all
//     (not even a Warning — there is nothing unresolved to surface, the runtime check is
//     the intended behavior).
//   - Float comparisons, unary operators (`!`, unary `-`), parenthesized or multi-term
//     expressions, and non-decimal/over-large integer literals: kept out to stay
//     conservative — folding them risks diverging from Go's runtime evaluation, and a
//     false "this assert always fails" is worse than leaving a decidable case unflagged.
//
// No analyze.Tables extension is needed: constant folding reads only the source tokens.
func checkAssert(src string, t *analyze.Tables) ([]Diagnostic, error) {
	toks := scan.Lex(src)
	var diags []Diagnostic

	for i := range toks {
		if toks[i].Text != "assert" || !scan.IsLineStart(src, toks[i].Start) {
			continue
		}
		lineEnd := scan.NextNewline(src, toks[i].End)

		// Bound the condition the same way the assert pass does: everything between
		// the keyword and the first top-level comma (or the line end).
		condEnd := lineEnd
		if c := firstAssertComma(toks, i+1, lineEnd); c >= 0 {
			condEnd = c
		}
		condToks := tokensIn(toks, toks[i].End, condEnd)
		condText := strings.TrimSpace(src[toks[i].End:condEnd])

		verdict, ok := foldAssertCond(condToks)
		if !ok {
			continue // not statically decidable — runtime check stands
		}
		if verdict {
			diags = append(diags, Diagnostic{
				Pos:      toks[i].Start,
				Severity: Warning,
				Feature:  "10-assert",
				Code:     "assert-always-true",
				Message: "assert condition `" + condText + "` is always true (dead code: " +
					"the runtime check can never fail)",
			})
		} else {
			diags = append(diags, Diagnostic{
				Pos:      toks[i].Start,
				Severity: Error,
				Feature:  "10-assert",
				Code:     "assert-always-false",
				Message: "assert condition `" + condText + "` is statically false: " +
					"this assert always panics",
			})
		}
	}
	return diags, nil
}

// firstAssertComma returns the byte offset of the first comma at bracket depth 0 between
// token index `from` and byte offset `lineEnd`, or -1 if none. Mirrors the assert pass's
// condition/message split.
func firstAssertComma(toks []scan.Token, from, lineEnd int) int {
	depth := 0
	for k := from; k < len(toks) && toks[k].Start < lineEnd; k++ {
		switch toks[k].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			if depth == 0 {
				return toks[k].Start
			}
		}
	}
	return -1
}

// tokensIn returns the tokens whose span falls within the byte range [lo, hi).
func tokensIn(toks []scan.Token, lo, hi int) []scan.Token {
	var out []scan.Token
	for k := range toks {
		if toks[k].Start >= lo && toks[k].End <= hi {
			out = append(out, toks[k])
		}
	}
	return out
}

// foldAssertCond constant-folds the two in-scope condition shapes and returns
// (value, true) when decidable, or (_, false) when the condition is not a constant the
// checker proves. In scope: a bare boolean literal, or a comparison of two integer
// literals. Everything else (any identifier, call, float, unary op, paren, or multi-term
// expression) returns ok=false so the runtime assert stands.
func foldAssertCond(cond []scan.Token) (value bool, ok bool) {
	switch len(cond) {
	case 1:
		switch cond[0].Text {
		case "true":
			return true, true
		case "false":
			return false, true
		}
		return false, false
	case 3, 4:
		// `LIT OP LIT`, where OP is one token (`<`, `>`) or two (`<=`, `>=`, `==`, `!=`).
		lhs, lok := constInt(cond[0].Text)
		if !lok {
			return false, false
		}
		var op string
		var rhsTok string
		if len(cond) == 3 {
			op = cond[1].Text
			rhsTok = cond[2].Text
		} else {
			op = cond[1].Text + cond[2].Text
			rhsTok = cond[3].Text
		}
		rhs, rok := constInt(rhsTok)
		if !rok {
			return false, false
		}
		return evalIntCompare(lhs, op, rhs)
	}
	return false, false
}

// constInt parses a single integer-literal token (decimal, 0x, 0o, 0b, or octal) into an
// int64, reporting false for anything that is not such a literal (identifiers, floats,
// over-large values). No sign: a leading `-` lexes as a separate token and is out of
// scope by design.
func constInt(s string) (int64, bool) {
	if s == "" {
		return 0, false
	}
	if c := s[0]; c != '0' && (c < '1' || c > '9') {
		return 0, false
	}
	v, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// evalIntCompare evaluates `lhs op rhs` for the comparison operators in scope, returning
// (result, true), or (_, false) for an operator this check does not fold.
func evalIntCompare(lhs int64, op string, rhs int64) (bool, bool) {
	switch op {
	case "<":
		return lhs < rhs, true
	case "<=":
		return lhs <= rhs, true
	case ">":
		return lhs > rhs, true
	case ">=":
		return lhs >= rhs, true
	case "==":
		return lhs == rhs, true
	case "!=":
		return lhs != rhs, true
	}
	return false, false
}

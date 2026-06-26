package fix

import (
	"goal/internal/analyze"
	"goal/internal/scan"
)

// fixSwitchToMatch reports a `switch` whose case labels name variants of an in-file enum as
// a candidate for `match`, which the checker can then verify for exhaustiveness. It does not
// rewrite: goal's `match` arms are single expressions, whereas Go `switch` clauses hold
// statement bodies, so a faithful mechanical rewrite is not expressible for the general
// case. Surfacing the opportunity (so the author converts the arms and gains the
// missing-variant check) is the safe, useful step. Returns no replacements.
func fixSwitchToMatch(src string, toks []scan.Token, t *analyze.Tables, changes *[]Change, reports *[]Report) []scan.Replacement {
	for i := range toks {
		if toks[i].Text != "switch" || !scan.IsLineStart(src, toks[i].Start) {
			continue
		}
		bo := switchBodyBrace(toks, i)
		if bo < 0 {
			continue
		}
		enum := switchEnumQualifier(toks, bo, t)
		if enum == "" {
			continue
		}
		*reports = append(*reports, Report{lineOf(src, toks[i].Start), Suggest, "match",
			"`switch` over enum `" + enum + "`; a `match` would be checked for exhaustiveness"})
	}
	return nil
}

// switchBodyBrace returns the token index of the `{` opening a switch's clause block: the
// first `{` at paren/bracket depth 0 after the switch keyword (a composite literal in the
// scrutinee sits inside parens, so it is not mistaken for the block). Returns -1 if none.
func switchBodyBrace(toks []scan.Token, si int) int {
	depth := 0
	for k := si + 1; k < len(toks); k++ {
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

// switchEnumQualifier returns the enum type name a switch's first `case` label qualifies
// (`case Status.Active:` -> "Status") when that name is a declared in-file enum, or "" when
// the switch is not over an in-file enum (a plain value switch, a type switch, or a switch
// whose first label is not `Enum.Variant`).
func switchEnumQualifier(toks []scan.Token, bo int, t *analyze.Tables) string {
	bc := scan.MatchBrace(toks, bo)
	for k := bo + 1; k+2 < bc; k++ {
		if toks[k].Text != "case" {
			continue
		}
		// First label must be `Ident . Ident`.
		if !scan.IsIdent(toks[k+1].Text) || toks[k+2].Text != "." {
			return ""
		}
		q := toks[k+1].Text
		if _, ok := t.Enums[q]; ok {
			return q
		}
		return ""
	}
	return ""
}

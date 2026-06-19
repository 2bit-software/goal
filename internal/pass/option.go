package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Option lowers Option[T] to the pointer strategy *T (feature 04): the type appears
// as *T everywhere (including return position), `return Option.None` becomes
// `return nil`, and `return Option.Some(x)` returns the address of x.
//
// Result returns are handled by the Result pass, which runs first; this pass touches
// only Option forms.
func Option(src string, _ *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	var reps []scan.Replacement

	// `Option[T]` -> `*T` wherever it appears (including Option returns).
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text == "Option" && toks[i+1].Text == "[" {
			rb := scan.MatchBracket(toks, i+1)
			t := strings.TrimSpace(src[toks[i+2].Start:toks[rb].Start])
			reps = append(reps, scan.Replacement{Start: toks[i].Start, End: toks[rb].End, Text: "*" + t})
			i = rb
		}
	}

	// `return Option.None` / `return Option.Some(x)`.
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Text != "return" || toks[i+1].Text != "Option" || toks[i+2].Text != "." {
			continue
		}
		switch toks[i+3].Text {
		case "None":
			reps = append(reps, scan.Replacement{Start: toks[i+1].Start, End: toks[i+3].End, Text: "nil"})
			i += 3
		case "Some":
			if i+4 >= len(toks) || toks[i+4].Text != "(" {
				continue
			}
			closeIdx := scan.MatchParen(toks, i+4)
			x := strings.TrimSpace(src[toks[i+4].End:toks[closeIdx].Start])
			var text string
			if closeIdx == i+6 && scan.IsIdent(toks[i+5].Text) {
				text = "return &" + x // addressable identifier -> &x directly (§8.4)
			} else {
				text = fmt.Sprintf("%s := %s\nreturn &%s", someName, x, someName) // box
			}
			reps = append(reps, scan.Replacement{Start: toks[i].Start, End: toks[closeIdx].End, Text: text})
			i = closeIdx
		}
	}

	// Pass C: lower statement-position `match opt { Option.Some/None arms }`. Only
	// matches whose arms are Option patterns are claimed here.
	for i := 0; i < len(toks); i++ {
		if toks[i].Text == "match" && scan.MatchQualifier(toks, i) == "Option" {
			rep, next, err := lowerOptionMatch(src, toks, i)
			if err != nil {
				return "", err
			}
			reps = append(reps, rep)
			i = next
		}
	}

	return scan.Splice(src, 0, len(src), reps), nil
}

// optionArm is one arm of an Option match.
type optionArm struct {
	variant string // "Some" or "None"
	binding string // "" if none
	bodyLo  int
	bodyHi  int
}

// lowerOptionMatch lowers a statement-position `match opt { Option.Some(b) => ...;
// Option.None => ... }` to `if p := opt; p != nil { b := *p; ... } else { ... }`
// (spec §8.4). It returns the replacement and the token index to continue from.
func lowerOptionMatch(src string, toks []scan.Token, mi int) (scan.Replacement, int, error) {
	if mi > 0 {
		if p := toks[mi-1].Text; p == "return" || p == "=" {
			return scan.Replacement{}, 0, fmt.Errorf("value-position Option match is deferred; consume an Option with a statement-position match (§8.4)")
		}
	}

	bo := scan.MatchBodyBrace(toks, mi)
	scrut := strings.TrimSpace(src[toks[mi].End:toks[bo].Start])
	bc := scan.MatchBrace(toks, bo)
	arms := parseOptionArms(toks, bo+1, bc)

	var some, none *optionArm
	for i := range arms {
		switch arms[i].variant {
		case "Some":
			some = &arms[i]
		case "None":
			none = &arms[i]
		}
	}
	if some == nil || none == nil {
		return scan.Replacement{}, 0, fmt.Errorf("Option match must have both Option.Some and Option.None arms")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "if %s := %s; %s != nil {\n", optBase, scrut, optBase)
	if some.binding != "" && bodyUses(toks, some.bodyLo, some.bodyHi, some.binding) {
		fmt.Fprintf(&b, "%s := *%s\n", some.binding, optBase)
	}
	b.WriteString(bodySrc(src, toks, some.bodyLo, some.bodyHi))
	b.WriteString("\n} else {\n")
	b.WriteString(bodySrc(src, toks, none.bodyLo, none.bodyHi))
	b.WriteString("\n}")
	return scan.Replacement{Start: toks[mi].Start, End: toks[bc].End, Text: b.String()}, bc + 1, nil
}

// parseOptionArms splits arm-block tokens [lo, hi) into arms, delimited by `=>`.
func parseOptionArms(toks []scan.Token, lo, hi int) []optionArm {
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

	arms := make([]optionArm, len(arrows))
	for i, eq := range arrows {
		patStart := lo
		if i > 0 {
			patStart = patternStart(toks, arrows[i])
		}
		a := optionArm{variant: toks[patStart+2].Text}
		if patStart+3 < eq && toks[patStart+3].Text == "(" {
			a.binding = toks[patStart+4].Text
		}
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

// bodySrc returns the trimmed source spanned by arm-body tokens [lo, hi).
func bodySrc(src string, toks []scan.Token, lo, hi int) string {
	if lo >= hi {
		return ""
	}
	return strings.TrimSpace(src[toks[lo].Start:toks[hi-1].End])
}

// bodyUses reports whether token text name appears in [lo, hi).
func bodyUses(toks []scan.Token, lo, hi int, name string) bool {
	for j := lo; j < hi; j++ {
		if toks[j].Text == name {
			return true
		}
	}
	return false
}

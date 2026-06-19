package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Match lowers a pattern-matching `match` on an enum to a Go type-switch over the
// §8.1 encoding (spec §8.2). A proven-exhaustive match (per the reference's no-check
// discipline) lowers to a panicking default; an explicit `_` arm becomes a real
// default. Positions handled: statement, `return match`, and `var name T = match`;
// the untyped `name := match` value form needs the checker's inferred type and is
// deferred with a located error.
//
// Only matches whose scrutinee is an enum (first arm qualifier names a known enum)
// are claimed; Result and Option matches belong to their own passes. This runs
// before the enums pass so the enum patterns here are consumed before that pass's
// construction rewrite could mistake them for value constructions.
func Match(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	var reps []scan.Replacement
	for i := 0; i < len(toks); {
		if toks[i].Text == "match" && t.Enums[scan.MatchQualifier(toks, i)] != nil {
			rep, next, err := lowerMatch(src, toks, t, i)
			if err != nil {
				return "", err
			}
			reps = append(reps, rep)
			i = next
			continue
		}
		i++
	}
	return scan.Splice(src, 0, len(src), reps), nil
}

// matchPos is where a match sits: a statement, in return position, or the
// initializer of an explicitly-typed `var`.
type matchPos int

const (
	posStmt matchPos = iota
	posReturn
	posVar
)

// matchArm is one arm of an enum match.
type matchArm struct {
	rest    bool   // the `_` rest arm
	enum    string // for non-rest arms
	variant string
	binding string // "" if no payload binding
	bodyLo  int
	bodyHi  int
}

// lowerMatch lowers the match whose `match` keyword is at toks[mi].
func lowerMatch(src string, toks []scan.Token, t *analyze.Tables, mi int) (scan.Replacement, int, error) {
	pos, name, typ, repStart, err := classifyPosition(src, toks, mi)
	if err != nil {
		return scan.Replacement{}, 0, err
	}

	bo := scan.MatchBodyBrace(toks, mi)
	scrut := strings.TrimSpace(src[toks[mi].End:toks[bo].Start])
	bc := scan.MatchBrace(toks, bo)
	arms := parseMatchArms(toks, bo+1, bc)

	bodies := make([]string, len(arms))
	usesBinding := false
	for i, a := range arms {
		var used bool
		bodies[i], used = rewriteMatchBody(src, toks, a, t)
		usesBinding = usesBinding || used
	}

	var b strings.Builder
	if pos == posVar {
		fmt.Fprintf(&b, "var %s %s\n", name, typ)
	}
	if usesBinding {
		fmt.Fprintf(&b, "switch %s := %s.(type) {\n", valName, scrut)
	} else {
		fmt.Fprintf(&b, "switch %s.(type) {\n", scrut)
	}

	enumName := ""
	restBody := ""
	hasRest := false
	for i, a := range arms {
		if a.rest {
			hasRest = true
			restBody = bodies[i]
			continue
		}
		enumName = a.enum
		fmt.Fprintf(&b, "case %s_%s:\n", a.enum, a.variant)
		b.WriteString(armStatement(pos, name, bodies[i]))
	}

	b.WriteString("default:\n")
	if hasRest {
		b.WriteString(armStatement(pos, name, restBody))
	} else {
		fmt.Fprintf(&b, "panic(%q)\n", fmt.Sprintf("unreachable: non-exhaustive %s (compiler invariant violated)", enumName))
	}
	b.WriteString("}")

	return scan.Replacement{Start: repStart, End: toks[bc].End, Text: b.String()}, bc + 1, nil
}

// classifyPosition inspects the tokens before `match` to decide its position and, for
// posVar, recover the declared name and type. repStart is where the generated
// replacement begins (the `return`/`var` keyword, or `match` itself).
func classifyPosition(src string, toks []scan.Token, mi int) (pos matchPos, name, typ string, repStart int, err error) {
	if mi == 0 {
		return posStmt, "", "", toks[mi].Start, nil
	}
	prev := toks[mi-1].Text
	switch {
	case prev == "return":
		return posReturn, "", "", toks[mi-1].Start, nil
	case prev == "=" && mi-2 >= 0 && toks[mi-2].Text == ":":
		return 0, "", "", 0, fmt.Errorf("value-position `name := match` needs the checker's inferred result type (deferred); use `var name T = match ...` or `return match ...`")
	case prev == "=":
		return classifyVar(src, toks, mi)
	default:
		return posStmt, "", "", toks[mi].Start, nil
	}
}

// classifyVar handles `var NAME TYPE = match ...` by walking back to the `var`.
func classifyVar(src string, toks []scan.Token, mi int) (matchPos, string, string, int, error) {
	eq := mi - 1 // the "=" token
	k := eq - 1
	for depth := 0; k >= 0; k-- {
		switch toks[k].Text {
		case ")", "]", "}":
			depth++
		case "(", "[", "{":
			depth--
		}
		if depth == 0 && toks[k].Text == "var" {
			break
		}
	}
	if k < 0 || toks[k].Text != "var" {
		return 0, "", "", 0, fmt.Errorf("match in `= match` position must be a `var NAME TYPE = match ...` declaration")
	}
	name := toks[k+1].Text
	typ := strings.TrimSpace(src[toks[k+2].Start:toks[eq].Start])
	return posVar, name, typ, toks[k].Start, nil
}

// armStatement wraps a lowered arm body for the match position.
func armStatement(pos matchPos, name, body string) string {
	switch pos {
	case posReturn:
		return "return " + body + "\n"
	case posVar:
		return name + " = " + body + "\n"
	default:
		return body + "\n"
	}
}

// parseMatchArms splits the arm-block tokens [lo, hi) into arms, delimited by `=>`.
func parseMatchArms(toks []scan.Token, lo, hi int) []matchArm {
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

	arms := make([]matchArm, len(arrows))
	for i, eq := range arrows {
		patStart := lo
		if i > 0 {
			patStart = patternStart(toks, arrows[i])
		}
		a := parseMatchPattern(toks, patStart, eq)
		a.bodyLo = eq + 2 // skip "=" ">"
		if i+1 < len(arrows) {
			a.bodyHi = patternStart(toks, arrows[i+1])
		} else {
			a.bodyHi = hi
		}
		arms[i] = a
	}
	return arms
}

// parseMatchPattern reads `Enum.Variant[(binding)]` or the `_` rest arm.
func parseMatchPattern(toks []scan.Token, start, eqIdx int) matchArm {
	if toks[start].Text == "_" {
		return matchArm{rest: true}
	}
	a := matchArm{enum: toks[start].Text, variant: toks[start+2].Text}
	if start+3 < eqIdx && toks[start+3].Text == "(" {
		a.binding = toks[start+4].Text
	}
	return a
}

// rewriteMatchBody rewrites the arm body: the payload binding becomes the guard
// variable and field accesses on it are exported (since -> Since). The bool reports
// whether the binding was referenced (so the caller knows if the guard is needed).
func rewriteMatchBody(src string, toks []scan.Token, a matchArm, t *analyze.Tables) (string, bool) {
	lo, hi := a.bodyLo, a.bodyHi
	if lo >= hi {
		return "", false
	}
	used := false
	var fields map[string]bool
	if a.enum != "" {
		if e, ok := t.Enums[a.enum]; ok {
			fields = e.FieldSet[a.variant]
		}
	}

	var reps []scan.Replacement
	for j := lo; j < hi; {
		if a.binding != "" && toks[j].Text == a.binding {
			used = true
			if j+2 < hi && toks[j+1].Text == "." && fields[toks[j+2].Text] {
				reps = append(reps, scan.Replacement{Start: toks[j].Start, End: toks[j].End, Text: valName})
				f := toks[j+2]
				reps = append(reps, scan.Replacement{Start: f.Start, End: f.End, Text: exported(f.Text)})
				j += 3
				continue
			}
			reps = append(reps, scan.Replacement{Start: toks[j].Start, End: toks[j].End, Text: valName})
		}
		j++
	}
	return scan.Splice(src, toks[lo].Start, toks[hi-1].End, reps), used
}

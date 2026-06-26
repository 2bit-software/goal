package check

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// checkExhaustive enforces feature 02 (match): a `match` over an enum must cover
// every variant, or supply an explicit `_` rest-arm. A match missing a variant and
// lacking `_` is an Error — the very case the lowering would otherwise turn into a
// silent `panic("unreachable: …")` default (spec §8.2).
//
// Reuse, not reinvention:
//   - The match pass (internal/pass/match.go) locates a `match` block by scanning for
//     the `match` token, then scan.MatchBodyBrace/scan.MatchBrace bound the arm block
//     and the `=>`-delimited arms are split at depth 0; the patternStart locator (shared
//     by every qualified match in internal/pass/result.go) finds each arm's first token.
//     This check mirrors that arm walk, but reads the covered variant set instead of
//     lowering.
//   - The match pass claims a match iff its first arm qualifier names a known enum
//     (t.Enums[scan.MatchQualifier(...)] != nil); this check resolves the enum the same
//     way — from the arm qualifiers, never from the scrutinee.
//   - Variant sets: analyze.Tables.Enums[…].VSet / .Variants (variants per enum).
//   - Must run pre-lowering: the type switch erases which variants were covered.
//
// Position-independence: exhaustiveness reads the *arm* qualifiers (`Status.Pending`),
// which are present in every match position — statement, `return match`, `var x T =
// match`, and the untyped `x := match` the lowering defers. So this check fires on all
// of them; it does not need the scrutinee's declared type.
//
// Defer-boundary: when a match's arms are enum-qualified but the named enum is not
// declared in this file (an out-of-package enum whose variant set is unknown),
// completeness is unprovable — emit a located Warning naming the unresolved enum rather
// than assume exhaustiveness. A match with no enum-qualified arm (a Result/Option match,
// or a value-position match whose first qualifier is not a known enum) is not this
// guarantee's concern and is skipped silently — those belong to their own features.
func checkExhaustive(src string, t *analyze.Tables) ([]Diagnostic, error) {
	toks := scan.Lex(src)
	var diags []Diagnostic
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "match" {
			continue
		}
		bo := scan.MatchBodyBrace(toks, i)
		if bo < 0 {
			continue
		}
		bc := scan.MatchBrace(toks, bo)
		diags = append(diags, checkOneMatch(toks, t, i, bo, bc)...)
		i = bc // skip past this match's body so a nested `match` token isn't re-scanned out of context
	}
	return diags, nil
}

// checkOneMatch checks the single match whose `match` keyword is toks[mi], arm block
// opens at toks[bo] == "{" and closes at toks[bc] == "}". The enum is resolved from the
// arm qualifiers, not the scrutinee, so the result is position-independent.
func checkOneMatch(toks []scan.Token, t *analyze.Tables, mi, bo, bc int) []Diagnostic {
	arrows := matchArrows(toks, bo+1, bc)
	if len(arrows) == 0 {
		return nil // not an arm block we recognize — leave it alone
	}

	enumName := "" // the enum named by the qualified arms
	covered := map[string]bool{}
	hasRest := false
	for k, arrow := range arrows {
		start := bo + 1
		if k > 0 {
			start = patternStart(toks, arrows[k])
		}
		qual, variant, rest := armPattern(toks, start, arrow)
		switch {
		case rest:
			hasRest = true
		case qual == "":
			// pattern this check cannot read — ignore the arm
		default:
			if enumName == "" {
				enumName = qual
			}
			if qual == enumName {
				covered[variant] = true
			}
		}
	}

	// No enum-qualified arm: a value-position match or some other construct — not ours.
	if enumName == "" {
		return nil
	}

	// `Result`/`Option` are builtin sum types, not user enums declared in any file. Their
	// match exhaustiveness is owned by their own features: the result/option lowering
	// passes reject a match missing an Ok/Err (or Some/None) arm outright. Checking them
	// here only ever fires the unresolved-enum deferral on correct, complete matches —
	// noise, never a catch. Skip them silently (the defer-boundary above intends this; the
	// builtin arms are qualified, so they reach here rather than the enumName=="" guard).
	if enumName == "Result" || enumName == "Option" {
		return nil
	}

	enum, ok := t.Enums[enumName]
	if !ok {
		// Arms name an enum not declared in this file: its full variant set is unknown,
		// so completeness is unprovable. Defer with a located Warning rather than risk a
		// false "exhaustive".
		return []Diagnostic{{
			Pos:      toks[mi].Start,
			Severity: Warning,
			Feature:  "02-match",
			Code:     "unresolved-match-enum",
			Message: fmt.Sprintf("cannot verify exhaustiveness of `match` on `%s`: enum `%s` is not declared in this file — exhaustiveness deferred",
				enumName, enumName),
		}}
	}

	// An explicit `_` rest-arm is a deliberate opt-out of exhaustiveness (spec §8.2):
	// complete by construction, no diagnostic.
	if hasRest {
		return nil
	}

	missing := missingVariants(enum, covered)
	if len(missing) == 0 {
		return nil
	}
	return []Diagnostic{{
		Pos:      toks[mi].Start,
		Severity: Error,
		Feature:  "02-match",
		Code:     "non-exhaustive-match",
		Message: fmt.Sprintf("non-exhaustive `match` on enum `%s`: missing variant%s %s — handle %s, or add a `_` rest-arm to dismiss the rest",
			enum.Name, plural(len(missing)), quoteVariants(enum.Name, missing), pronoun(len(missing))),
	}}
}

// matchArrows returns the indices of the `=` token of each `=>` arm arrow at depth 0
// between lo and hi (a scrutinee literal's or arm body's brackets are skipped). Mirrors
// parseMatchArms in the match pass.
func matchArrows(toks []scan.Token, lo, hi int) []int {
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
	return arrows
}

// patternStart finds where the arm pattern ending just before the arrow at eqIdx begins,
// from the token immediately before the arrow. Lifted from internal/pass/result.go (the
// locator shared by every qualified match): `Qual.Variant`, `Qual.Variant(binding)`, and
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

// armPattern reads the arm pattern in toks[start:arrowIdx]: `_` (rest=true), or
// `Enum.Variant[(binding)]` yielding the qualifier and variant. A pattern this check
// cannot read (not `_` and not `IDENT . IDENT`) returns qual=="" so the caller skips it.
func armPattern(toks []scan.Token, start, arrowIdx int) (qual, variant string, rest bool) {
	if start < 0 || start >= arrowIdx {
		return "", "", false
	}
	if toks[start].Text == "_" {
		return "", "", true
	}
	if start+2 < arrowIdx && toks[start+1].Text == "." && scan.IsIdent(toks[start].Text) {
		return toks[start].Text, toks[start+2].Text, false
	}
	return "", "", false
}

// missingVariants returns the enum's declared variants not in the covered set, in
// declaration order (the order an agent reads them in the enum decl).
func missingVariants(enum *analyze.Enum, covered map[string]bool) []string {
	var missing []string
	for _, v := range enum.Variants {
		if !covered[v.Name] {
			missing = append(missing, v.Name)
		}
	}
	return missing
}

// quoteVariants renders missing variants as backtick-quoted `Enum.Variant`,
// comma-separated, in the order given (declaration order). The qualifier echoes the
// match's arm form so the message names exactly the patterns the agent must add.
func quoteVariants(enumName string, variants []string) string {
	qualified := make([]string, len(variants))
	for i, v := range variants {
		qualified[i] = "`" + enumName + "." + v + "`"
	}
	return strings.Join(qualified, ", ")
}

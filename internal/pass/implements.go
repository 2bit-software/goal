package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Implements lowers `implements X for T` to the §8.5 compile-time assertion
// `var _ X = T{}` (or `var _ X = (*T)(nil)` when T has a pointer-receiver method).
// The check is the checker's job and is erased; only the free, self-verifying
// assertion remains (feature 07).
//
// `implements X for T` is shared surface with feature 01: when X is a sealed
// interface the enums pass emits a marker method instead. This pass therefore claims
// only the non-sealed form, so the two never both rewrite the same declaration.
func Implements(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	pointerRecv := scanPointerReceivers(toks)

	var reps []scan.Replacement
	for i := range toks {
		if toks[i].Text != "implements" {
			continue
		}
		j := i + 1
		for j < len(toks) && toks[j].Text != "for" {
			j++
		}
		if j >= len(toks) {
			continue
		}
		iface := strings.TrimSpace(src[toks[i].End:toks[j].Start])
		if t.Sealed[iface] {
			continue // sealed-interface marker form is the enums pass's job
		}
		lineEnd := scan.NextNewline(src, toks[j].End)
		typ := strings.TrimSpace(src[toks[j].End:lineEnd])
		base := scan.BaseType(typ)

		var assertion string
		if pointerRecv[base] {
			assertion = fmt.Sprintf("var _ %s = (*%s)(nil)", iface, base)
		} else {
			assertion = fmt.Sprintf("var _ %s = %s{}", iface, typ)
		}
		reps = append(reps, scan.Replacement{Start: toks[i].Start, End: lineEnd, Text: assertion})
	}
	return scan.Splice(src, 0, len(src), reps), nil
}

// scanPointerReceivers returns the set of type names with at least one pointer-
// receiver method (`func (x *T) ...`), so the assertion can address T correctly.
func scanPointerReceivers(toks []scan.Token) map[string]bool {
	set := map[string]bool{}
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "func" || toks[i+1].Text != "(" {
			continue
		}
		rc := scan.MatchParen(toks, i+1)
		star := false
		typeName := ""
		for k := i + 2; k < rc; k++ {
			switch {
			case toks[k].Text == "*":
				star = true
			case scan.IsIdent(toks[k].Text):
				typeName = toks[k].Text // last identifier in the receiver = the type
			}
		}
		if star && typeName != "" {
			set[typeName] = true
		}
	}
	return set
}

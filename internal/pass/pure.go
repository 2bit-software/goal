package pass

import (
	"goal/internal/analyze"
	"goal/internal/scan"
)

// Pure erases the `pure func` modifier (spec §4.2, §8.5): the checker verifies a
// pure body is effect-free, then the guarantee is erased — `pure func` lowers to a
// plain `func`, nothing else. `pure` anywhere other than directly before `func` is an
// ordinary identifier and is left untouched.
func Pure(src string, _ *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	var reps []scan.Replacement
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text == "pure" && toks[i+1].Text == "func" {
			reps = append(reps, scan.Replacement{Start: toks[i].Start, End: toks[i+1].Start, Text: ""})
		}
	}
	return scan.Splice(src, 0, len(src), reps), nil
}

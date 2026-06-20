package pass

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Implements lowers the inline `implements` clause on a struct declaration —
// `type T struct implements X, Y { … }` — into the Go that proves the satisfaction.
// Each interface in the comma-separated list becomes either:
//
//   - the §8.5 compile-time assertion `var _ X = T{}` (or `var _ X = (*T)(nil)` when T
//     has a pointer-receiver method) for an ordinary interface (feature 07), or
//   - the unexported marker method `func (T) isX() {}` when X is a sealed interface
//     (feature 01).
//
// The clause is stripped from the declaration (leaving plain Go) and the generated
// declarations are emitted immediately after the struct's closing brace — the same place
// the standalone `implements X for T` form used to lower to. A single clause may mix
// sealed and ordinary interfaces, so this one pass now handles both forms; the enums pass
// no longer touches `implements`.
//
// Only struct types carry the clause today. Extending it to any concrete type (as Go
// allows, e.g. `type Celsius float64 implements Stringer`) is future work.
func Implements(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	pointerRecv := scanPointerReceivers(toks)

	var reps []scan.Replacement
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "type" || !scan.IsIdent(toks[i+1].Text) || toks[i+2].Text != "struct" {
			continue
		}
		name := toks[i+1].Text

		// The struct body's "{" is the first brace after `struct`; the interface names
		// in the clause (type names, possibly qualified or `[T]`-generic) carry none.
		open := -1
		for k := i + 3; k < len(toks); k++ {
			if toks[k].Text == "{" {
				open = k
				break
			}
		}
		if open < 0 {
			continue
		}
		imp := -1
		for k := i + 3; k < open; k++ {
			if toks[k].Text == "implements" {
				imp = k
				break
			}
		}
		if imp < 0 {
			continue // a plain struct with no implements clause
		}
		ifaces := splitInterfaces(src[toks[imp].End:toks[open].Start])
		if len(ifaces) == 0 {
			continue
		}
		closeIdx := scan.MatchBrace(toks, open)

		var b strings.Builder
		for _, iface := range ifaces {
			switch {
			case t.Sealed[iface]:
				b.WriteString(genMarker(name, iface)) // sealed-interface marker (feature 01)
			case pointerRecv[name]:
				fmt.Fprintf(&b, "var _ %s = (*%s)(nil)", iface, name)
			default:
				fmt.Fprintf(&b, "var _ %s = %s{}", iface, name)
			}
			b.WriteByte('\n')
		}

		// Collapse the clause `struct implements X, Y {` -> `struct {`, then emit the
		// generated declarations after the struct's closing brace.
		reps = append(reps, scan.Replacement{Start: toks[i+2].End, End: toks[open].Start, Text: " "})
		reps = append(reps, scan.Replacement{Start: toks[closeIdx].End, End: toks[closeIdx].End, Text: "\n\n" + b.String()})
	}
	return scan.Splice(src, 0, len(src), reps), nil
}

// splitInterfaces splits a clause's comma-separated interface list into trimmed names,
// dropping empties. Qualified names (`io.Writer`) survive intact — they carry no comma.
func splitInterfaces(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
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

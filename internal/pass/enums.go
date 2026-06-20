package pass

import (
	"fmt"
	"strings"
	"unicode"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Enums lowers closed sum types to the sealed-interface + per-variant-struct +
// unexported-marker Go encoding (spec §8.1): an `enum` declaration becomes its
// encoding, a `sealed interface` becomes a marker interface, and a variant construction
// `Enum.V(field: x)` becomes `Enum(Enum_V{Field: x})`.
//
// This pass runs LATE — after the match pass — because a variant construction and an
// enum match pattern share the surface form `Enum.Variant(...)`. The match pass
// consumes the patterns inside `match` blocks first; whatever `Enum.Variant(...)`
// remains is a genuine construction this pass rewrites. The marker method that admits a
// type into a sealed interface comes from its `implements` clause, lowered by the
// implements pass (which calls genMarker); this pass only emits the sealed interface
// declaration itself.
func Enums(src string, t *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	var reps []scan.Replacement

	// Pass A: declarations (enum encoding, sealed interface).
	for i := 0; i < len(toks); {
		switch toks[i].Text {
		case "enum":
			e := t.Enums[toks[i+1].Text]
			closeIdx := scan.MatchBrace(toks, i+2) // i+1 name, i+2 "{"
			if e != nil {
				reps = append(reps, scan.Replacement{Start: toks[i].Start, End: toks[closeIdx].End, Text: genEnum(e)})
			}
			i = closeIdx + 1
		case "sealed":
			// sealed interface NAME { } -> a marker interface.
			name := toks[i+2].Text
			end := scan.MatchBrace(toks, i+3) // i+3 "{"
			reps = append(reps, scan.Replacement{Start: toks[i].Start, End: toks[end].End, Text: genInterface(name)})
			i = end + 1
		default:
			i++
		}
	}

	// Pass B: rewrite variant constructions `Enum.V[(...)]` now that all enums are
	// known. Match patterns have already been consumed by the match pass.
	for j := 0; j+2 < len(toks); {
		e, ok := t.Enums[toks[j].Text]
		if ok && toks[j+1].Text == "." && e.VSet[toks[j+2].Text] {
			vname := toks[j+2].Text
			if j+3 < len(toks) && toks[j+3].Text == "(" {
				closeIdx := scan.MatchParen(toks, j+3)
				args := parseArgs(src, toks, j+4, closeIdx)
				reps = append(reps, scan.Replacement{Start: toks[j].Start, End: toks[closeIdx].End, Text: construct(e.Name, vname, args)})
				j = closeIdx + 1
				continue
			}
			reps = append(reps, scan.Replacement{Start: toks[j].Start, End: toks[j+2].End, Text: construct(e.Name, vname, nil)})
			j += 3
			continue
		}
		j++
	}

	return scan.Splice(src, 0, len(src), reps), nil
}

// kv is a labelled construction argument `label: expr`.
type kv struct {
	label string
	expr  string
}

// parseArgs parses `label: expr, label: expr` between the open paren (start = first
// arg token) and closeIdx (the matching ")"), capturing expressions verbatim and
// honoring nesting so calls like now() survive.
func parseArgs(src string, toks []scan.Token, start, closeIdx int) []kv {
	var args []kv
	k := start
	for k < closeIdx {
		label := toks[k].Text
		k += 2 // skip label and ":"
		exprStart := toks[k].Start
		exprEnd := toks[k].End
		depth := 0
		for k < closeIdx {
			t := toks[k]
			if depth == 0 && t.Text == "," {
				break
			}
			switch t.Text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			exprEnd = t.End
			k++
		}
		args = append(args, kv{label: label, expr: strings.TrimSpace(src[exprStart:exprEnd])})
		if k < closeIdx && toks[k].Text == "," {
			k++
		}
	}
	return args
}

// genEnum emits the §8.1 encoding for an enum: a marker interface, one struct per
// variant, and a marker method per variant.
func genEnum(e *analyze.Enum) string {
	marker := "is" + e.Name
	var b strings.Builder
	fmt.Fprintf(&b, "type %s interface{ %s() }\n\n", e.Name, marker)
	for _, v := range e.Variants {
		if len(v.Fields) == 0 {
			fmt.Fprintf(&b, "type %s_%s struct{}\n", e.Name, v.Name)
			continue
		}
		fmt.Fprintf(&b, "type %s_%s struct {\n", e.Name, v.Name)
		for _, f := range v.Fields {
			fmt.Fprintf(&b, "\t%s %s\n", exported(f.Name), f.Type)
		}
		b.WriteString("}\n")
	}
	b.WriteString("\n")
	for _, v := range e.Variants {
		fmt.Fprintf(&b, "func (%s_%s) %s() {}\n", e.Name, v.Name, marker)
	}
	return b.String()
}

// genInterface emits a sealed interface as a marker interface.
func genInterface(name string) string {
	return fmt.Sprintf("type %s interface{ is%s() }", name, name)
}

// genMarker emits the marker method that admits typ into the sealed interface iface.
func genMarker(typ, iface string) string {
	return fmt.Sprintf("func (%s) is%s() {}", typ, iface)
}

// construct emits a variant construction: Enum(Enum_V{Field: expr, ...}).
func construct(enum, variant string, args []kv) string {
	if len(args) == 0 {
		return fmt.Sprintf("%s(%s_%s{})", enum, enum, variant)
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprintf("%s: %s", exported(a.label), a.expr)
	}
	return fmt.Sprintf("%s(%s_%s{%s})", enum, enum, variant, strings.Join(parts, ", "))
}

// exported capitalizes the first rune so a goal field/label maps to an exported Go
// field name.
func exported(name string) string {
	if name == "" {
		return name
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

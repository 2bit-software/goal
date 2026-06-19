package pass

import (
	"fmt"
	"strconv"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Assert lowers the runtime assertion `assert <cond> [, <fmt> [, <args>...]]` to a
// runtime-preserved `if !(cond) { panic(...) }` (spec §4.3, §8.6). Unlike the erased
// static guarantees, the check survives into the generated Go. The panic message
// always includes the source expression text (the located-feedback rule); an
// optional printf-style message is appended via fmt.Sprintf, and `import "fmt"` is
// injected after the package clause when needed.
//
// `assert` is the keyword only as the first token on its line; conditions are
// single-line and statement-positioned.
func Assert(src string, _ *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	var reps []scan.Replacement
	needsFmt := false
	for i := range toks {
		if toks[i].Text != "assert" || !scan.IsLineStart(src, toks[i].Start) {
			continue
		}
		lineEnd := scan.NextNewline(src, toks[i].End)

		// Split at the first top-level comma: left is the condition, right (if any)
		// is the printf-style message `"fmt", args...`.
		commaStart := firstTopLevelComma(toks, i+1, lineEnd)
		var cond, msg string
		if commaStart < 0 {
			cond = trimStmt(src[toks[i].End:lineEnd])
		} else {
			cond = trimStmt(src[toks[i].End:commaStart])
			msg = trimStmt(src[commaStart+1 : lineEnd])
		}

		var block string
		if msg == "" {
			block = fmt.Sprintf("if !(%s) { panic(%s) }", cond, strconv.Quote("assertion failed: "+cond))
		} else {
			needsFmt = true
			block = fmt.Sprintf("if !(%s) { panic(%s + fmt.Sprintf(%s)) }",
				cond, strconv.Quote("assertion failed: "+cond+": "), msg)
		}
		reps = append(reps, scan.Replacement{Start: toks[i].Start, End: lineEnd, Text: block})
	}

	if needsFmt && !importsFmt(toks) {
		if pos := packageLineEnd(src, toks); pos >= 0 {
			reps = append(reps, scan.Replacement{Start: pos, End: pos, Text: "\n\nimport \"fmt\""})
		}
	}
	return scan.Splice(src, 0, len(src), reps), nil
}

// firstTopLevelComma returns the byte offset of the first comma at bracket depth 0
// between token index `from` and byte offset `lineEnd`, or -1 if none.
func firstTopLevelComma(toks []scan.Token, from, lineEnd int) int {
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

// trimStmt trims surrounding whitespace and a trailing statement-ending ";".
func trimStmt(s string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), ";"))
}

// importsFmt reports whether the source already imports the fmt package.
func importsFmt(toks []scan.Token) bool {
	for i := range toks {
		if toks[i].Text == `"fmt"` {
			return true
		}
	}
	return false
}

// packageLineEnd returns the offset of the newline ending the `package <name>`
// clause, where an injected import can be inserted, or -1 if not found.
func packageLineEnd(src string, toks []scan.Token) int {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text == "package" {
			return scan.NextNewline(src, toks[i+1].End)
		}
	}
	return -1
}

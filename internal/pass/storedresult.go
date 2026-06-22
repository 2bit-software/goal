package pass

import (
	"fmt"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// StoredResultGuard refuses, with a located error, an OPEN-E `Result[T, error]` used as a
// first-class *value* in the two positions where it is unambiguously stored and provably
// miscompiles today: a slice/array/map element (`[]Result[...]`, `map[K]Result[...]`) and a
// struct or enum field. Per spec §8.7 an open-E Result lowers to a native `(T, error)`
// tuple meant to be CONSUMED at the use site — returned, then `?`/`match`-ed — not held as
// data; stored, the tuple has no type name to spell, so `Result[int, error]` is undefined
// in the lowered Go. Rather than emit that silent breakage, this guard turns it into an
// honest, located diagnostic naming the §8.7 rule. It runs before the Result/Option/closed
// passes, so a rejected program never reaches them.
//
// Deliberately NOT guarded (only the always-broken open-E value case is):
//   - a CLOSED-E `Result[T, E]` (E not `error`), which lowers to the Ok/Err sum interface
//     and IS first-class — a closed-E Result field is a supported, lowerable pattern (the
//     must-use depth check relies on it). (A separate, narrower gap — constructing a
//     closed-E Result *in* a stored position — is left to the full L4b work, not pre-judged
//     here, to avoid a false rejection of the common declared/assigned field.)
//   - `Option[T]`, which lowers to `*T` and IS first-class — stored Options are fine.
//   - a Result *parameter* matched in-body, which the lexical checker treats as valid
//     surface (testdata/check/02-match/result_match_ignored.goal): whether that is
//     "immediate" or "stored" is the open §8.7 question, left to L4b rather than pre-judged.
//   - a bare local `var x Result[...]` (rare; not in a clearly-stored position).
func StoredResultGuard(src string, _ *analyze.Tables) (string, error) {
	toks := scan.Lex(src)
	declSpans := typeBodySpans(toks)
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "Result" || toks[i+1].Text != "[" {
			continue
		}
		if !isOpenEResult(toks, i+1) {
			continue // closed-E Result is first-class (Ok/Err sum) — not guarded
		}
		where := ""
		switch {
		case i > 0 && toks[i-1].Text == "]":
			where = "a slice/array/map element"
		case inBraceSpan(i, declSpans):
			where = "a struct/enum field"
		default:
			continue // a direct return type, a parameter, or a local — not guarded here
		}
		line, col := lineCol(src, toks[i].Start)
		return "", fmt.Errorf("open-E `Result[T, error]` stored as %s at %d:%d: it is not a first-class value (§8.7) — it lowers to an `(T, error)` tuple consumed at the use site (return it, then `?` or `match`), not stored. Consume it there, or store the unwrapped value and `error` separately", where, line, col)
	}
	return src, nil
}

// isOpenEResult reports whether the `Result[...]` whose `[` is at token openIdx is the
// open-E form — its error parameter E is exactly `error` (vs. a closed-E sum whose E is an
// enum/struct). It reads the last top-level type argument between the brackets.
func isOpenEResult(toks []scan.Token, openIdx int) bool {
	close := scan.MatchBracket(toks, openIdx)
	last := openIdx + 1 // start of the current (eventually last) top-level arg
	depth := 0
	for k := openIdx + 1; k < close; k++ {
		switch toks[k].Text {
		case "[", "(", "{":
			depth++
		case "]", ")", "}":
			depth--
		case ",":
			if depth == 0 {
				last = k + 1
			}
		}
	}
	// The error parameter is the single token before the closing bracket when it is a bare
	// `error`; a qualified or compound E spans more tokens and is therefore closed-E.
	return close-last == 1 && toks[last].Text == "error"
}

// braceSpan is a half-open token-index range (open, close) of a type-declaration body.
type braceSpan struct{ open, close int }

// typeBodySpans returns the brace-body token spans of every `struct { … }` and
// `enum NAME { … }` declaration, so a `Result[…]` inside one is recognized as a field
// type. (A nested struct's brace sits inside the outer span; the outer span suffices.)
func typeBodySpans(toks []scan.Token) []braceSpan {
	var spans []braceSpan
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "struct" && toks[i].Text != "enum" {
			continue
		}
		open := -1
		for k := i + 1; k < len(toks) && k <= i+3; k++ {
			if toks[k].Text == "{" {
				open = k
				break
			}
		}
		if open < 0 {
			continue
		}
		spans = append(spans, braceSpan{open: open, close: scan.MatchBrace(toks, open)})
	}
	return spans
}

// inBraceSpan reports whether token index idx falls strictly inside any span.
func inBraceSpan(idx int, spans []braceSpan) bool {
	for _, s := range spans {
		if idx > s.open && idx < s.close {
			return true
		}
	}
	return false
}

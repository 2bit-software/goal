// Package textedit holds the lexer-independent text utilities the transpiler
// passes need: the splice model (Replacement + Splice), small identifier and
// line-scanning helpers, and the type helpers (BaseType, ZeroLit) that reason
// over type expressions as plain strings.
//
// Nothing here depends on the lexer or the token stream — these are the pure
// survivors relocated out of the former internal/scan and internal/analyze so
// those lexer-based packages could be deleted without losing the utilities.
package textedit

import (
	"sort"
	"strings"
	"unicode"
)

// Replacement is a byte span [Start, End) of the source to splice over with Text.
type Replacement struct {
	Start, End int
	Text       string
}

// Splice rebuilds src[lo:hi] with each replacement span swapped for its text.
// Replacements are sorted by start; any that overlaps an earlier one is skipped
// defensively rather than producing corrupt output.
func Splice(src string, lo, hi int, reps []Replacement) string {
	sort.Slice(reps, func(a, b int) bool { return reps[a].Start < reps[b].Start })
	var b strings.Builder
	prev := lo
	for _, r := range reps {
		if r.Start < prev {
			continue
		}
		b.WriteString(src[prev:r.Start])
		b.WriteString(r.Text)
		prev = r.End
	}
	b.WriteString(src[prev:hi])
	return b.String()
}

// IsLineStart reports whether everything between the previous newline and byte
// offset p is whitespace — i.e. the token at p begins a statement (so a keyword like
// `assert` is the statement keyword, not an identifier used mid-expression).
func IsLineStart(src string, p int) bool {
	for k := p - 1; k >= 0; k-- {
		switch src[k] {
		case '\n':
			return true
		case ' ', '\t':
			continue
		default:
			return false
		}
	}
	return true
}

// NextNewline returns the offset of the next '\n' at or after p, or len(src).
func NextNewline(src string, p int) int {
	if nl := strings.IndexByte(src[p:], '\n'); nl >= 0 {
		return p + nl
	}
	return len(src)
}

// LeadIdent returns the leading identifier of s (letters, digits, underscore from
// the start), e.g. the callee name of a call expression `parse(x)` -> "parse".
func LeadIdent(s string) string {
	end := 0
	for end < len(s) {
		r := rune(s[end])
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			end++
			continue
		}
		break
	}
	return s[:end]
}

// IsIdent reports whether s begins like a Go identifier (letter or underscore).
func IsIdent(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return unicode.IsLetter(r) || r == '_'
}

// SplitAssign parses `lhs := rhs` into its trimmed halves. ok is false when there is
// no `:=`, in which case rhs is the whole trimmed string and name is empty.
func SplitAssign(s string) (name, rhs string, ok bool) {
	if lhs, after, found := strings.Cut(s, ":="); found {
		return strings.TrimSpace(lhs), strings.TrimSpace(after), true
	}
	return "", strings.TrimSpace(s), false
}

// IsStmtKeyword reports whether s is a Go/goal keyword that can lead a statement, so a line
// beginning with it is not a bare expression statement.
func IsStmtKeyword(s string) bool {
	switch s {
	case "return", "go", "defer", "if", "else", "for", "switch", "select", "case",
		"default", "var", "const", "type", "func", "range", "break", "continue",
		"goto", "fallthrough", "match", "assert", "enum", "import", "package":
		return true
	}
	return false
}

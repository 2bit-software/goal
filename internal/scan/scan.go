// Package scan holds the shared low-level machinery every transpiler pass needs:
// the lexer (text/scanner with byte-offset recovery), the splice model
// (Replacement + Splice), balanced-delimiter matching, and the structural helpers
// (function scanning, parameter/brace location) that the references each duplicated.
//
// Passes splice bytes, so byte offsets shift between passes. Nothing in this package
// caches offsets across edits: each pass re-lexes the current source and rebuilds the
// spans it needs. Analysis that must survive splicing is keyed by name, not offset
// (see package analyze).
package scan

import (
	"sort"
	"strings"
	"text/scanner"
	"unicode"
)

// Token is a lexical token with its byte span [Start, End) in the source it was
// lexed from.
type Token struct {
	Text  string
	Start int
	End   int
}

// Lex tokenizes src with Go's text/scanner, recovering each token's byte offset.
// Comments are scanned and skipped (so passes that care about comments read them
// from the raw source, not the token stream).
func Lex(src string) []Token {
	var s scanner.Scanner
	s.Init(strings.NewReader(src))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanRawStrings |
		scanner.ScanInts | scanner.ScanFloats | scanner.ScanChars |
		scanner.ScanComments | scanner.SkipComments
	s.Whitespace = 1<<'\t' | 1<<'\n' | 1<<'\r' | 1<<' '
	var toks []Token
	for tk := s.Scan(); tk != scanner.EOF; tk = s.Scan() {
		txt := s.TokenText()
		start := s.Position.Offset
		toks = append(toks, Token{Text: txt, Start: start, End: start + len(txt)})
	}
	return toks
}

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

// MatchPair returns the index of the close delimiter matching the open delimiter
// at openIdx. The generalized form (from feature 12) underlies the named wrappers.
func MatchPair(toks []Token, openIdx int, open, close string) int {
	depth := 0
	for k := openIdx; k < len(toks); k++ {
		switch toks[k].Text {
		case open:
			depth++
		case close:
			depth--
		}
		if depth == 0 {
			return k
		}
	}
	return len(toks) - 1
}

// MatchParen returns the index of the ")" matching the "(" at openIdx.
func MatchParen(toks []Token, openIdx int) int { return MatchPair(toks, openIdx, "(", ")") }

// MatchBracket returns the index of the "]" matching the "[" at openIdx.
func MatchBracket(toks []Token, openIdx int) int { return MatchPair(toks, openIdx, "[", "]") }

// MatchBrace returns the index of the "}" matching the "{" at openIdx.
func MatchBrace(toks []Token, openIdx int) int { return MatchPair(toks, openIdx, "{", "}") }

// MatchBodyBrace returns the index of the "{" opening a `match`'s arm block: the
// first "{" at paren/bracket depth 0 after the `match` token at mi. The scrutinee's
// own braces (composite literals) sit inside parens, so they are not mistaken for
// the arm block. It returns -1 if none.
func MatchBodyBrace(toks []Token, mi int) int {
	depth := 0
	for k := mi + 1; k < len(toks); k++ {
		switch toks[k].Text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case "{":
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// MatchQualifier reports the leading qualifier of a match's first arm pattern
// ("Result", "Option", an enum type name, …) so a pass can claim only the matches it
// lowers. It returns "" when the qualifier cannot be read.
func MatchQualifier(toks []Token, mi int) string {
	bo := MatchBodyBrace(toks, mi)
	if bo < 0 || bo+1 >= len(toks) {
		return ""
	}
	return toks[bo+1].Text
}

// FirstBodyBrace returns the index of a function body's opening "{": the first "{"
// at paren/bracket depth 0 after the `func` token at fi. It returns -1 if none.
func FirstBodyBrace(toks []Token, fi int) int {
	depth := 0
	for k := fi + 1; k < len(toks); k++ {
		switch toks[k].Text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case "{":
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// ParamsClose returns the index of the ")" closing a function's parameter list: the
// first ")" at depth 0 scanning back from the body brace, skipping a balanced return
// type such as Result[T, error]. It returns -1 if none.
func ParamsClose(toks []Token, body int) int {
	depth := 0
	for k := body - 1; k >= 0; k-- {
		t := toks[k].Text
		if depth == 0 && t == ")" {
			return k
		}
		switch t {
		case ")", "]":
			depth++
		case "(", "[":
			depth--
		}
	}
	return -1
}

// TopLevelComma returns the index of the "," at depth 0 strictly between openIdx and
// closeIdx, or -1 if there is none.
func TopLevelComma(toks []Token, openIdx, closeIdx int) int {
	depth := 0
	for k := openIdx + 1; k < closeIdx; k++ {
		switch toks[k].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
		if depth == 0 && toks[k].Text == "," {
			return k
		}
	}
	return -1
}

// BaseType strips a leading "*" and any "pkg." qualifier, yielding the bare type
// name (used to look up a local type or receiver type).
func BaseType(t string) string {
	t = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), "*"))
	if i := strings.LastIndexByte(t, '.'); i >= 0 {
		t = t[i+1:]
	}
	return t
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

// Func is the structural skeleton of one top-level function or method, located by
// token index in whatever source was lexed. Spans are recomputed per pass.
type Func struct {
	Name        string // declared name ("" if it could not be read)
	NameTok     int    // token index of the name identifier
	ParamsClose int    // token index of the ")" closing the parameter list, or -1
	BodyOpen    int    // token index of the body "{"
	BodyClose   int    // token index of the body "}"
}

// ScanFuncs locates every `func` declaration in toks, reading each one's name and
// body span. A receiver (`func (r T) name(...)`) is skipped so Name is the method
// name. Functions whose body brace cannot be found are omitted.
func ScanFuncs(toks []Token) []Func {
	var funcs []Func
	for i := range toks {
		if toks[i].Text != "func" {
			continue
		}
		nameTok := i + 1
		if nameTok < len(toks) && toks[nameTok].Text == "(" {
			nameTok = MatchParen(toks, nameTok) + 1 // skip receiver
		}
		bo := FirstBodyBrace(toks, i)
		if bo < 0 {
			continue
		}
		f := Func{
			NameTok:     nameTok,
			ParamsClose: ParamsClose(toks, bo),
			BodyOpen:    bo,
			BodyClose:   MatchBrace(toks, bo),
		}
		if nameTok < len(toks) && IsIdent(toks[nameTok].Text) {
			f.Name = toks[nameTok].Text
		}
		funcs = append(funcs, f)
	}
	return funcs
}

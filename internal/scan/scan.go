// Package scan holds the shared low-level lexer machinery every transpiler pass
// needs: the lexer (text/scanner with byte-offset recovery), balanced-delimiter
// matching, and the structural helpers (function scanning, parameter/brace
// location) that the references each duplicated. The lexer-independent text and
// type utilities live in package textedit.
//
// Passes splice bytes, so byte offsets shift between passes. Nothing in this package
// caches offsets across edits: each pass re-lexes the current source and rebuilds the
// spans it needs. Analysis that must survive splicing is keyed by name, not offset
// (see package analyze).
package scan

import (
	"strings"
	"text/scanner"

	"goal/internal/textedit"
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

// CalleeKey returns the analyze-table lookup key for the function expr calls: a
// package-qualified call "os.MkdirAll(p)" yields "os.MkdirAll", a plain call "doThing(x)"
// yields "doThing", and a generic instantiation "f[T](x)" yields the base "f". It returns ""
// when expr is not a direct call to a named function — a value-led or parenthesized
// expression, or a chained call whose OUTERMOST call is a method ("exec.Command(x).Output()").
// A chain is deliberately unresolved rather than mis-attributed to the head of the chain
// ("exec.Command"): the `?` lowering then keeps the safe two-value form for it. Lexing (so the
// parens of a string argument never miscount) keeps this robust for real call expressions.
func CalleeKey(expr string) string {
	toks := Lex(expr)
	if len(toks) == 0 || !textedit.IsIdent(toks[0].Text) {
		return ""
	}
	key := toks[0].Text
	i := 1
	if i+1 < len(toks) && toks[i].Text == "." && textedit.IsIdent(toks[i+1].Text) {
		key += "." + toks[i+1].Text // package-qualified callee pkg.Func
		i += 2
	}
	if i < len(toks) && toks[i].Text == "[" {
		i = MatchBracket(toks, i) + 1 // generic instantiation f[T](x) -> base f
	}
	if i >= len(toks) || toks[i].Text != "(" {
		return "" // not a direct call (a bare selector or value)
	}
	if MatchParen(toks, i) != len(toks)-1 {
		return "" // a token follows the call's `)` — a chained method call; unresolvable here
	}
	return key
}

// IsBareQuestionStmt reports whether the `?` token at index qIdx is the whole of a
// standalone expression statement `expr?` — the binding-free discard form, where the call's
// only output is the error and there is no `name :=` / `_ :=` to its left. It requires the
// `?` to be the final token of its line and the line not to begin with a statement keyword,
// so a `?` used mid-expression (`f(g()?)`) or after a keyword (`return f()?`) is not mistaken
// for a bare statement. lineStart is the byte offset beginning the `?`'s line.
func IsBareQuestionStmt(src string, toks []Token, qIdx, lineStart int) bool {
	if qIdx < 0 || qIdx >= len(toks) || toks[qIdx].Text != "?" {
		return false
	}
	// `?` must terminate the statement: nothing follows it on the same line.
	if qIdx+1 < len(toks) && toks[qIdx+1].Start < textedit.NextNewline(src, toks[qIdx].End) {
		return false
	}
	// Walk back to the first token of the line and reject a leading statement keyword.
	first := qIdx
	for first > 0 && toks[first-1].Start >= lineStart {
		first--
	}
	return !textedit.IsStmtKeyword(toks[first].Text)
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
		if nameTok < len(toks) && textedit.IsIdent(toks[nameTok].Text) {
			f.Name = toks[nameTok].Text
		}
		funcs = append(funcs, f)
	}
	return funcs
}

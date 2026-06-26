package lsp

import (
	"encoding/json"

	"goal/internal/check"
	"goal/internal/scan"
)

// documentSymbols answers a textDocument/documentSymbol request with the open file's outline:
// one entry per top-level declaration (enum, struct, interface, sealed interface, type alias,
// function, method, and `from`/`derive func`). It is best-effort — a declaration it cannot
// read is skipped rather than failing — and always returns a non-nil slice so it marshals as
// `[]`, never `null`.
func (s *Server) documentSymbols(raw json.RawMessage) []DocumentSymbol {
	var p DocumentSymbolParams
	if !s.decode(raw, &p, "documentSymbol") {
		return []DocumentSymbol{}
	}
	text, _, ok := s.buffer(p.TextDocument.URI)
	if !ok {
		return []DocumentSymbol{}
	}
	return collectSymbols(text)
}

// decl is a top-level declaration located during the first pass: its symbol kind, optional
// detail, the token of its leading keyword, and the token of its name.
type decl struct {
	kind    int
	detail  string
	kwTok   int
	nameTok int
}

// collectSymbols extracts the top-level declarations of src as outline symbols. The range of
// each declaration is bounded by the next declaration's keyword so a bodyless `from`/`derive
// func` or `type X = …` alias never absorbs the declaration that follows it.
func collectSymbols(src string) []DocumentSymbol {
	toks := scan.Lex(src)
	decls := scanDecls(src, toks)

	out := []DocumentSymbol{}
	for k, d := range decls {
		if d.nameTok < 0 || d.nameTok >= len(toks) || !scan.IsIdent(toks[d.nameTok].Text) {
			continue // unreadable name — skip, don't guess
		}
		limit := len(toks)
		if k+1 < len(decls) {
			limit = decls[k+1].kwTok
		}
		startOff := toks[d.kwTok].Start
		endOff := declEnd(src, toks, d.kwTok, limit)
		name := toks[d.nameTok]
		out = append(out, DocumentSymbol{
			Name:           name.Text,
			Detail:         d.detail,
			Kind:           d.kind,
			Range:          rangeOf(src, startOff, endOff),
			SelectionRange: rangeOf(src, name.Start, name.End),
		})
	}
	return out
}

// scanDecls walks toks once, recording each top-level declaration in source order. It tracks
// bracket nesting and only acts on a keyword seen at depth zero; a declaration's own body
// raises the depth, so members are naturally skipped. A `type X = …` alias's whole line is
// stepped over so a `func(...)` type in its right-hand side is not mistaken for a function.
func scanDecls(srcText string, toks []scan.Token) []decl {
	var decls []decl
	depth := 0
	for i := 0; i < len(toks); i++ {
		switch toks[i].Text {
		case "{", "(", "[":
			depth++
			continue
		case "}", ")", "]":
			depth--
			continue
		}
		if depth != 0 {
			continue
		}
		switch {
		case toks[i].Text == "enum":
			decls = append(decls, decl{kind: symEnum, kwTok: i, nameTok: i + 1})
		case toks[i].Text == "sealed" && i+1 < len(toks) && toks[i+1].Text == "interface":
			decls = append(decls, decl{kind: symInterface, detail: "sealed interface", kwTok: i, nameTok: i + 2})
		case toks[i].Text == "type":
			d := decl{kind: symClass, kwTok: i, nameTok: i + 1}
			alias := false
			if i+2 < len(toks) {
				switch toks[i+2].Text {
				case "struct":
					d.kind = symStruct
				case "interface":
					d.kind = symInterface
				case "=":
					alias = true
				}
			}
			decls = append(decls, d)
			if alias {
				i = skipLine(srcText, toks, i) // step past the alias's RHS so its func(...) type isn't a decl
			}
		case toks[i].Text == "func" && (i == 0 || toks[i-1].Text != "="):
			d := decl{kind: symFunction, kwTok: i, nameTok: i + 1}
			if i > 0 && (toks[i-1].Text == "from" || toks[i-1].Text == "derive") {
				d.kwTok = i - 1
				d.detail = toks[i-1].Text + " func"
			} else if i+1 < len(toks) && toks[i+1].Text == "(" {
				d.kind = symMethod // func (recv T) name(...)
				d.nameTok = scan.MatchParen(toks, i+1) + 1
			}
			decls = append(decls, d)
		}
	}
	return decls
}

// skipLine returns the token index to resume from after stepping over the rest of the line
// that token i begins on, so a single-line construct (a `type X = …` alias) is consumed whole.
// The parentheses on such a line are balanced, so bracket depth is unaffected by the skip.
func skipLine(srcText string, toks []scan.Token, i int) int {
	lineEnd := scan.NextNewline(srcText, toks[i].Start)
	j := i + 1
	for j < len(toks) && toks[j].Start < lineEnd {
		j++
	}
	return j - 1 // the loop's i++ advances to j
}

// declEnd returns the byte offset where the declaration starting at keyword token kwTok ends:
// the close of its body brace when one appears before limit (the next declaration's keyword),
// or the end of the keyword's line for a bodyless declaration.
func declEnd(srcText string, toks []scan.Token, kwTok, limit int) int {
	for b := kwTok + 1; b < limit && b < len(toks); b++ {
		if toks[b].Text == "{" {
			if close := scan.MatchBrace(toks, b); close >= 0 && close < len(toks) {
				return toks[close].End
			}
			break
		}
	}
	return scan.NextNewline(srcText, toks[kwTok].Start)
}

// rangeOf converts a byte span into a 0-based protocol range.
func rangeOf(srcText string, startOff, endOff int) Range {
	s := check.OffsetToPosition(srcText, startOff)
	e := check.OffsetToPosition(srcText, endOff)
	return Range{
		Start: Position{Line: s.Line - 1, Character: s.Col - 1},
		End:   Position{Line: e.Line - 1, Character: e.Col - 1},
	}
}

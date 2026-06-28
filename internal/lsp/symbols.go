package lsp

import (
	"encoding/json"

	"goal/internal/ast"
	"goal/internal/parser"
	"goal/internal/token"
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

// collectSymbols extracts the top-level declarations of src as outline symbols by parsing the
// file and walking its declarations. Each declaration carries its own source positions, so the
// full range is the declaration's span and the selection range is its name — a bodyless
// `from`/`derive func` or `type X = …` alias therefore cannot absorb the declaration that
// follows it (its End() stops at its own last token). Source that does not parse yields an
// empty (non-nil) slice rather than an error, matching the LSP's best-effort contract.
func collectSymbols(src string) []DocumentSymbol {
	out := []DocumentSymbol{}
	file, err := parser.ParseFile(src)
	if err != nil || file == nil {
		return out
	}
	for _, d := range file.Decls {
		out = append(out, symbolsFor(src, d)...)
	}
	return out
}

// symbolsFor maps one top-level declaration to its outline symbols. A type declaration may
// hold several specs (a grouped `type ( … )`) and so yields one symbol per spec; every other
// declaration yields a single symbol. A declaration whose name is missing is skipped.
func symbolsFor(src string, d ast.Decl) []DocumentSymbol {
	switch decl := d.(type) {
	case *ast.EnumDecl:
		return single(src, decl.Name, symEnum, "", decl.Pos(), decl.End())
	case *ast.SealedInterfaceDecl:
		return single(src, decl.Name, symInterface, "sealed interface", decl.Pos(), decl.End())
	case *ast.FuncDecl:
		kind := symFunction
		if decl.Recv != nil {
			kind = symMethod // func (recv T) name(...)
		}
		detail := ""
		switch decl.Mod {
		case ast.FuncFrom:
			detail = "from func"
		case ast.FuncDerive:
			detail = "derive func"
		}
		return single(src, decl.Name, kind, detail, decl.Pos(), decl.End())
	case *ast.GenDecl:
		if decl.Tok != token.TYPE {
			return nil // import/const/var declarations are not part of the outline
		}
		var out []DocumentSymbol
		for _, sp := range decl.Specs {
			ts, ok := sp.(*ast.TypeSpec)
			if !ok {
				continue
			}
			start := ts.Pos()
			if len(decl.Specs) == 1 {
				start = decl.Pos() // keep the `type` keyword in range for a single-spec decl
			}
			out = append(out, single(src, ts.Name, typeSpecKind(ts), "", start, ts.End())...)
		}
		return out
	}
	return nil
}

// typeSpecKind classifies a type declaration's outline kind: a struct, an interface, or — for
// an alias or any other underlying type — a plain class symbol.
func typeSpecKind(ts *ast.TypeSpec) int {
	if ts.Assign != (token.Pos{}) {
		return symClass // type X = … alias
	}
	switch ts.Type.(type) {
	case *ast.StructType:
		return symStruct
	case *ast.InterfaceType:
		return symInterface
	default:
		return symClass
	}
}

// single builds the one-element symbol slice for a declaration whose name is name, spanning
// [start,end) with the name as its selection range. A nil or unnamed declaration yields no
// symbol so the outline never reports an unreadable entry.
func single(src string, name *ast.Ident, kind int, detail string, start, end token.Pos) []DocumentSymbol {
	if name == nil || name.Name == "" {
		return nil
	}
	return []DocumentSymbol{{
		Name:           name.Name,
		Detail:         detail,
		Kind:           kind,
		Range:          rangeOf(src, start.Offset, end.Offset),
		SelectionRange: rangeOf(src, name.Pos().Offset, name.End().Offset),
	}}
}

// rangeOf converts a byte span into a 0-based protocol range.
func rangeOf(srcText string, startOff, endOff int) Range {
	s := token.OffsetToPosition(srcText, startOff)
	e := token.OffsetToPosition(srcText, endOff)
	return Range{
		Start: Position{Line: s.Line - 1, Character: s.Col - 1},
		End:   Position{Line: e.Line - 1, Character: e.Col - 1},
	}
}

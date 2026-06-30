package lsp

import (
	"encoding/json"
	"strings"

	"goal/internal/fix"
	"goal/internal/token"
)

// fixAllKind is the code-action kind for the whole-file idiomatize rewrite. It sits under the
// `source.fixAll` umbrella so a client's `editor.codeActionsOnSave` can trigger it on save.
const fixAllKind = "source.fixAll.goal"

// codeActions answers a textDocument/codeAction request. When the document's buffer contains
// plain-Go patterns that `fix.File` rewrites, it offers a single "Idiomatize file" action
// whose edit replaces the whole document with the idiomatic source; otherwise it offers none.
// The result is always a non-nil slice so it marshals as `[]`, never `null`.
func (s *Server) codeActions(raw json.RawMessage) []CodeAction {
	none := []CodeAction{}
	var p CodeActionParams
	if !s.decode(raw, &p, "codeAction") {
		return none
	}
	if !wantsKind(p.Context.Only, fixAllKind) {
		return none
	}
	text, version, ok := s.buffer(p.TextDocument.URI)
	if !ok {
		return none
	}
	// fix.File is a total, conservative lexical rewrite — it never panics on partial source
	// and returns the input unchanged when there is nothing to idiomatize.
	out, _, _ := fix.File(text)
	if out == text {
		return none
	}
	end := token.OffsetToPosition(text, len(text))
	return []CodeAction{{
		Title: "Idiomatize file (goal fix)",
		Kind:  fixAllKind,
		Edit: &WorkspaceEdit{DocumentChanges: []TextDocumentEdit{{
			TextDocument: versionedTextDocumentIdentifier{URI: p.TextDocument.URI, Version: version},
			Edits: []TextEdit{{
				Range: Range{
					Start: Position{Line: 0, Character: 0},
					End:   Position{Line: end.Line - 1, Character: end.Col - 1},
				},
				NewText: out,
			}},
		}}},
	}}
}

// wantsKind reports whether the client's code-action kind filter admits kind. An empty filter
// admits everything; otherwise a requested kind matches when it equals kind or is one of its
// dot-separated ancestors (so a request for "source" admits "source.fixAll.goal").
func wantsKind(only []string, kind string) bool {
	if len(only) == 0 {
		return true
	}
	for _, o := range only {
		if o == kind || strings.HasPrefix(kind, o+".") {
			return true
		}
	}
	return false
}

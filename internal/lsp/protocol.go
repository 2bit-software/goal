// Package lsp is a minimal Language Server for goal. It speaks the Language
// Server Protocol over stdio and, for this milestone, surfaces goal's static
// check violations to an editor as diagnostics. It depends only on the standard
// library and the existing check package.
package lsp

// Position is a zero-based line and character offset, as the protocol requires.
// goal's own positions are one-based, so they are decremented on the way out.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is a half-open span; End is exclusive.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Diagnostic is one editor-visible finding. Severity uses the protocol's scale
// (1=Error, 2=Warning, 3=Information, 4=Hint).
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// PublishDiagnosticsParams carries a file's complete current diagnostic set to
// the editor; an empty slice clears prior findings.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     int          `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// InitializeResult tells the client what the server can do.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

// ServerCapabilities advertises full-document sync; that is all a
// diagnostics-only server needs.
type ServerCapabilities struct {
	TextDocumentSync int `json:"textDocumentSync"`
}

// ServerInfo identifies the server in client logs.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// textDocumentItem is the full text and identity of a newly opened document.
type textDocumentItem struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
	Text    string `json:"text"`
}

// versionedTextDocumentIdentifier names a document and the edit revision it is at.
type versionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// textDocumentIdentifier names a document.
type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

// contentChange is one edit. Under full sync, Text is the whole new document.
type contentChange struct {
	Text string `json:"text"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type didChangeParams struct {
	TextDocument   versionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []contentChange                 `json:"contentChanges"`
}

type didCloseParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

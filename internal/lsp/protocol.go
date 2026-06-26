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

// ServerCapabilities advertises what the server can do: full-document sync, an idiomatize
// fix-all code action, and document-symbol (outline) support.
type ServerCapabilities struct {
	TextDocumentSync       int                `json:"textDocumentSync"`
	CodeActionProvider     *CodeActionOptions `json:"codeActionProvider,omitempty"`
	DocumentSymbolProvider bool               `json:"documentSymbolProvider,omitempty"`
}

// CodeActionOptions declares which code-action kinds the server offers.
type CodeActionOptions struct {
	CodeActionKinds []string `json:"codeActionKinds,omitempty"`
}

// CodeActionParams is a textDocument/codeAction request: the document, the range the action
// was requested for, and a context carrying the client's kind filter and current diagnostics.
type CodeActionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionContext carries the client's requested kinds (Only) and the diagnostics at the
// requested range.
type CodeActionContext struct {
	Only        []string     `json:"only,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

// CodeAction is one offered action; Edit applies directly when the client runs it.
type CodeAction struct {
	Title string         `json:"title"`
	Kind  string         `json:"kind,omitempty"`
	Edit  *WorkspaceEdit `json:"edit,omitempty"`
}

// WorkspaceEdit groups document edits. The version-pinned DocumentChanges form lets the
// client reject the edit if the buffer changed since it was computed.
type WorkspaceEdit struct {
	DocumentChanges []TextDocumentEdit `json:"documentChanges"`
}

// TextDocumentEdit edits one versioned document.
type TextDocumentEdit struct {
	TextDocument versionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                      `json:"edits"`
}

// TextEdit replaces Range with NewText.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// DocumentSymbolParams is a textDocument/documentSymbol request for one document's outline.
type DocumentSymbolParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

// DocumentSymbol is one outline entry: Range covers the whole declaration, SelectionRange the
// name to reveal, and Children any nested symbols.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolKind values from the Language Server Protocol.
const (
	symClass      = 5
	symMethod     = 6
	symField      = 8
	symEnum       = 10
	symInterface  = 11
	symFunction   = 12
	symEnumMember = 22
	symStruct     = 23
)

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

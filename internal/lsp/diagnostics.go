package lsp

import (
	"strings"

	"goal/internal/check"
)

// compile runs goal's static checks over text and publishes the findings for
// uri, unless a newer revision of the document has since arrived.
func (s *Server) compile(uri, text string, version int) {
	diags, err := check.Analyze(text)
	if err != nil {
		// An error here is an internal checker bug, not a rejected program.
		s.logf("analyze %s: %v", uri, err)
		return
	}

	out := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		out = append(out, toLSP(text, d))
	}

	if s.superseded(uri, version) {
		return
	}
	s.publish(uri, version, out)
}

// superseded reports whether a revision newer than version is already stored,
// meaning this analysis is stale and should be dropped.
func (s *Server) superseded(uri string, version int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.docs[uri]
	return cur != nil && cur.version > version
}

func (s *Server) publish(uri string, version int, diags []Diagnostic) {
	s.notify("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Version:     version,
		Diagnostics: diags,
	})
}

// toLSP converts a goal check finding (1-based byte-offset position) into a
// protocol diagnostic (0-based range). With no token length available, the
// range spans from the finding to the end of its line.
func toLSP(text string, d check.Diagnostic) Diagnostic {
	p := check.OffsetToPosition(text, d.Pos)
	line := p.Line - 1
	startChar := p.Col - 1
	endChar := lineLength(text, p.Line)
	if endChar <= startChar {
		endChar = startChar + 1
	}

	severity := 1 // Error
	if d.Severity == check.Warning {
		severity = 2 // Warning
	}

	return Diagnostic{
		Range: Range{
			Start: Position{Line: line, Character: startChar},
			End:   Position{Line: line, Character: endChar},
		},
		Severity: severity,
		Code:     d.Code,
		Source:   "goal",
		Message:  d.Message,
	}
}

// lineLength returns the character count of the given 1-based line, excluding
// the trailing carriage return. ASCII source is assumed for character counting.
func lineLength(src string, line1 int) int {
	idx := line1 - 1
	if idx < 0 {
		return 0
	}
	lines := strings.Split(src, "\n")
	if idx >= len(lines) {
		return 0
	}
	return len(strings.TrimRight(lines[idx], "\r"))
}

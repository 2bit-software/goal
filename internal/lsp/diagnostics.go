package lsp

import (
	"path/filepath"
	"sort"
	"strings"

	"goal/internal/check"
	"goal/internal/project"
)

// compile runs goal's static checks for the open document uri and publishes the findings,
// resolving types declared in sibling files of the same package directory and in imported
// Go packages so cross-file and cross-package references are proven rather than deferred.
// When the document cannot be mapped to a package directory on disk, it falls back to
// analyzing the buffer alone. Diagnostics for a newer revision supersede this one.
func (s *Server) compile(uri, text string, version int) {
	path, ok := uriToPath(uri)
	if !ok {
		s.compileSingle(uri, text, version)
		return
	}
	dir := filepath.Dir(path)

	// Serialize package analysis: when several files' debounced compiles overlap, the last
	// to run re-reads the freshest buffers (below) and publishes last, so the final state
	// is correct even though it publishes siblings' diagnostics too.
	s.analysisMu.Lock()
	defer s.analysisMu.Unlock()

	disk, err := s.files(dir)
	if err != nil {
		s.logf("read dir %s: %v", dir, err)
		s.compileSingle(uri, text, version)
		return
	}

	// Re-snapshot the open buffers now that analysisMu is held, and pin the analyzed file to
	// the revision that triggered this compile.
	open := s.openFilesInDir(dir)
	open[uri] = openFile{uri: uri, path: path, text: text, version: version}

	views := mergePackageView(disk, open)
	srcs := make([]string, len(views))
	for i := range views {
		srcs[i] = views[i].src
	}

	if conflictingPackageNames(srcs) {
		s.logf("package name conflict in %s: single-file fallback", dir)
		s.compileSingle(uri, text, version)
		return
	}

	perFile, ferrs, err := check.AnalyzePackageInDirWith(srcs, dir, s.resolve)
	if err != nil {
		// An error here is an internal checker bug, not a rejected program.
		s.logf("analyze package %s: %v", dir, err)
		return
	}
	for _, fe := range ferrs {
		s.logf("foreign resolve %s: %v", dir, fe) // non-fatal: the type stays deferred
	}

	// Publish refreshed diagnostics for every open file in the package — fixing one file
	// clears the stale cross-file diagnostics of its open siblings without touching them.
	for i := range views {
		of := views[i].open
		if of == nil {
			continue // not open in the editor; nothing to show
		}
		if s.superseded(of.uri, of.version) {
			continue
		}
		out := make([]Diagnostic, 0, len(perFile[i]))
		for _, d := range perFile[i] {
			out = append(out, toLSP(of.text, d))
		}
		s.publish(of.uri, of.version, out)
	}
}

// compileSingle analyzes the buffer in isolation and publishes its findings — the fallback
// when no package directory resolves (a non-file URI, an unreadable directory, or a package
// whose files disagree on their package name).
func (s *Server) compileSingle(uri, text string, version int) {
	diags, err := check.Analyze(text)
	if err != nil {
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

// openFile is an open document resolved to its filesystem path.
type openFile struct {
	uri     string
	path    string
	text    string
	version int
}

// openFilesInDir returns the open documents whose file lives directly in dir, keyed by URI.
func (s *Server) openFilesInDir(dir string) map[string]openFile {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]openFile{}
	for uri, d := range s.docs {
		p, ok := uriToPath(uri)
		if !ok || filepath.Dir(p) != dir {
			continue
		}
		out[uri] = openFile{uri: uri, path: p, text: d.text, version: d.version}
	}
	return out
}

// packageEntry is one source of the package view: its contents and, when the source is an
// open document, that document so its diagnostics can be published back to its URI.
type packageEntry struct {
	src  string
	open *openFile
}

// mergePackageView builds the package's sources, path-sorted for a deterministic table
// merge, overlaying open buffers onto the on-disk files (an open buffer's unsaved text wins)
// and including any open file absent from disk (a new, never-saved file).
func mergePackageView(disk []fileSrc, open map[string]openFile) []packageEntry {
	byPath := map[string]*packageEntry{}
	paths := make([]string, 0, len(disk)+len(open))
	for _, f := range disk {
		byPath[f.path] = &packageEntry{src: f.src}
		paths = append(paths, f.path)
	}
	for _, of := range open {
		o := of
		if e, found := byPath[o.path]; found {
			e.src = o.text // buffer wins over disk
			e.open = &o
			continue
		}
		byPath[o.path] = &packageEntry{src: o.text, open: &o}
		paths = append(paths, o.path)
	}
	sort.Strings(paths)
	views := make([]packageEntry, len(paths))
	for i, p := range paths {
		views[i] = *byPath[p]
	}
	return views
}

// conflictingPackageNames reports whether the sources declare more than one package name —
// the one-package-per-directory rule project.Discover enforces. Sources with no package
// clause are ignored.
func conflictingPackageNames(srcs []string) bool {
	name := ""
	for _, src := range srcs {
		got := project.PackageClause(src)
		if got == "" {
			continue
		}
		if name == "" {
			name = got
			continue
		}
		if got != name {
			return true
		}
	}
	return false
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

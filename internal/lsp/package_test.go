package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"goal/internal/analyze"
	"goal/internal/check"
)

// fakeFiles is an in-memory dirReader: dir → its .goal files.
func fakeFiles(byDir map[string][]fileSrc) dirReader {
	return func(dir string) ([]fileSrc, error) {
		fs, ok := byDir[dir]
		if !ok {
			return nil, fmt.Errorf("no such dir %q", dir)
		}
		return fs, nil
	}
}

// fakeResolver maps known import paths to fixture directories; an unknown import errors so
// the test never reaches the real go toolchain.
func fakeResolver(m map[string]string) analyze.DirResolver {
	return func(importPath, _ string) (string, error) {
		if d, ok := m[importPath]; ok {
			return d, nil
		}
		return "", fmt.Errorf("unresolved import %q", importPath)
	}
}

// testServer is a synchronous server (no debounce) wired to in-memory IO, so a compile's
// published diagnostics are in out before the call returns.
func testServer(out *bytes.Buffer, files dirReader, resolve analyze.DirResolver) *Server {
	s := NewServerWithIO(out, files, resolve)
	s.debounce = 0
	return s
}

func (s *Server) testOpen(t *testing.T, uri, text string, version int) {
	t.Helper()
	raw, _ := json.Marshal(didOpenParams{TextDocument: textDocumentItem{URI: uri, Version: version, Text: text}})
	s.didOpen(raw)
}

func (s *Server) testChange(t *testing.T, uri, text string, version int) {
	t.Helper()
	raw, _ := json.Marshal(didChangeParams{
		TextDocument:   versionedTextDocumentIdentifier{URI: uri, Version: version},
		ContentChanges: []contentChange{{Text: text}},
	})
	s.didChange(raw)
}

func (s *Server) testClose(t *testing.T, uri string) {
	t.Helper()
	raw, _ := json.Marshal(didCloseParams{TextDocument: textDocumentIdentifier{URI: uri}})
	s.didClose(raw)
}

// latestDiagnostics returns, per URI, the diagnostics of the most recent publish for it.
func latestDiagnostics(t *testing.T, out *bytes.Buffer) map[string][]Diagnostic {
	t.Helper()
	byURI := map[string][]Diagnostic{}
	r := bufio.NewReader(bytes.NewReader(out.Bytes()))
	for {
		m, err := readMessage(r)
		if err != nil {
			break
		}
		if m.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p PublishDiagnosticsParams
		if err := json.Unmarshal(m.Params, &p); err != nil {
			t.Fatalf("unmarshal diagnostics: %v", err)
		}
		byURI[p.URI] = p.Diagnostics
	}
	return byURI
}

func hasCode(diags []Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// A derive func whose target struct lives in a sibling file resolves once the package is
// analyzed together — the in-file-only deferral is gone.
func TestCrossFileDeriveResolves(t *testing.T) {
	const aURI, bURI = "file:///pkg/a.goal", "file:///pkg/b.goal"
	aSrc := `package p

type Src struct {
	ID string
}

type Spec struct {
	ID string
}
`
	bSrc := `package p

derive func mk(s Src) Spec
`
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(map[string][]fileSrc{
		"/pkg": {{path: "/pkg/a.goal", src: aSrc}, {path: "/pkg/b.goal", src: bSrc}},
	}), fakeResolver(nil))
	s.testOpen(t, aURI, aSrc, 1)
	s.testOpen(t, bURI, bSrc, 1)

	diags := latestDiagnostics(t, &out)
	if hasCode(diags[bURI], "unresolved-derive-type") {
		t.Errorf("Spec should resolve cross-file, got deferral: %+v", diags[bURI])
	}
	if hasCode(diags[bURI], "unsourced-field") {
		t.Errorf("derive is total, unexpected unsourced-field: %+v", diags[bURI])
	}
}

// With the cross-file types resolved, a genuinely unsourced target field is a real error,
// not a deferral — completeness is now enforced across files.
func TestCrossFileUnsourcedFieldErrors(t *testing.T) {
	const aURI, bURI = "file:///pkg/a.goal", "file:///pkg/b.goal"
	aSrc := `package p

type Src struct {
	ID string
}

type Spec struct {
	ID    string
	Extra string
}
`
	bSrc := `package p

derive func mk(s Src) Spec
`
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(map[string][]fileSrc{
		"/pkg": {{path: "/pkg/a.goal", src: aSrc}, {path: "/pkg/b.goal", src: bSrc}},
	}), fakeResolver(nil))
	s.testOpen(t, aURI, aSrc, 1)
	s.testOpen(t, bURI, bSrc, 1)

	diags := latestDiagnostics(t, &out)
	if !hasCode(diags[bURI], "unsourced-field") {
		t.Errorf("expected unsourced-field for Spec.Extra, got: %+v", diags[bURI])
	}
}

// A derive whose source is an imported Go struct resolves through the injected resolver —
// the non-goal reference no longer defers.
func TestForeignDeriveResolves(t *testing.T) {
	extDir, err := filepath.Abs("../check/testdata/extpkg")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	const uri = "file:///pkg/a.goal"
	src := `package p

import ext "example.com/extpkg"

type Local struct {
	ID    string
	Count int
}

derive func mk(o *ext.Outer) Local
`
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(map[string][]fileSrc{
		"/pkg": {{path: "/pkg/a.goal", src: src}},
	}), fakeResolver(map[string]string{"example.com/extpkg": extDir}))
	s.testOpen(t, uri, src, 1)

	diags := latestDiagnostics(t, &out)
	if hasCode(diags[uri], "unresolved-derive-type") {
		t.Errorf("foreign type should resolve, got deferral: %+v", diags[uri])
	}
	if len(diags[uri]) != 0 {
		t.Errorf("expected no diagnostics for a total foreign derive, got: %+v", diags[uri])
	}
}

// Editing one open file refreshes the diagnostics of its open siblings without the sibling
// being touched: adding an unsourced field to A's struct makes B's derive error.
func TestEditRefreshesOpenSibling(t *testing.T) {
	const aURI, bURI = "file:///pkg/a.goal", "file:///pkg/b.goal"
	aClean := `package p

type Src struct {
	ID string
}

type Spec struct {
	ID string
}
`
	aBroken := `package p

type Src struct {
	ID string
}

type Spec struct {
	ID    string
	Extra string
}
`
	bSrc := `package p

derive func mk(s Src) Spec
`
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(map[string][]fileSrc{
		"/pkg": {{path: "/pkg/a.goal", src: aClean}, {path: "/pkg/b.goal", src: bSrc}},
	}), fakeResolver(nil))
	s.testOpen(t, aURI, aClean, 1)
	s.testOpen(t, bURI, bSrc, 1)
	if d := latestDiagnostics(t, &out); hasCode(d[bURI], "unsourced-field") {
		t.Fatalf("B should be clean before the edit: %+v", d[bURI])
	}

	out.Reset()
	s.testChange(t, aURI, aBroken, 2) // unsaved edit to A only

	diags := latestDiagnostics(t, &out)
	if !hasCode(diags[bURI], "unsourced-field") {
		t.Errorf("editing A should surface an unsourced-field on B, got: %+v", diags[bURI])
	}
}

// Each file's diagnostics publish to its own URI only — a clean file is not tagged with its
// sibling's error.
func TestPerFileAttribution(t *testing.T) {
	const aURI, bURI = "file:///pkg/a.goal", "file:///pkg/b.goal"
	aSrc := `package p

type Src struct {
	ID string
}

type Spec struct {
	ID    string
	Extra string
}
`
	bSrc := `package p

derive func mk(s Src) Spec
`
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(map[string][]fileSrc{
		"/pkg": {{path: "/pkg/a.goal", src: aSrc}, {path: "/pkg/b.goal", src: bSrc}},
	}), fakeResolver(nil))
	s.testOpen(t, aURI, aSrc, 1)
	s.testOpen(t, bURI, bSrc, 1)

	diags := latestDiagnostics(t, &out)
	if hasCode(diags[aURI], "unsourced-field") {
		t.Errorf("the error belongs to B, not A: %+v", diags[aURI])
	}
	if !hasCode(diags[bURI], "unsourced-field") {
		t.Errorf("expected the error on B: %+v", diags[bURI])
	}
}

// A non-file URI has no package directory, so the server analyzes the buffer alone and still
// publishes its findings.
func TestNonFileURIFallsBackToSingleFile(t *testing.T) {
	const uri = "untitled:Untitled-1"
	var out bytes.Buffer
	// The dirReader would error for any dir; reaching it would fail the test.
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.testOpen(t, uri, nonExhaustiveSrc, 1)

	diags := latestDiagnostics(t, &out)
	if len(diags[uri]) == 0 {
		t.Errorf("expected single-file diagnostics for a non-file URI, got none")
	}
}

// A directory whose files disagree on their package name cannot be analyzed as a unit, so
// the server falls back to single-file analysis and still publishes the open file's findings.
func TestPackageConflictFallsBackToSingleFile(t *testing.T) {
	const bURI = "file:///pkg/b.goal"
	aSrc := `package q

type Spec struct {
	ID string
}
`
	bSrc := nonExhaustiveSrc // declares `package p`
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(map[string][]fileSrc{
		"/pkg": {{path: "/pkg/a.goal", src: aSrc}, {path: "/pkg/b.goal", src: bSrc}},
	}), fakeResolver(nil))
	s.testOpen(t, bURI, bSrc, 1)

	diags := latestDiagnostics(t, &out)
	if len(diags[bURI]) == 0 {
		t.Errorf("expected single-file fallback to still publish B's diagnostics")
	}
}

// An open buffer in a different directory is not pulled into the analyzed file's package view.
func TestOpenFilesInDirExcludesOtherDirs(t *testing.T) {
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert("file:///pkg/a.goal", "package p", 1)
	s.upsert("file:///other/c.goal", "package p", 1)

	open := s.openFilesInDir("/pkg")
	if _, ok := open["file:///pkg/a.goal"]; !ok {
		t.Errorf("expected /pkg/a.goal in the /pkg view")
	}
	if _, ok := open["file:///other/c.goal"]; ok {
		t.Errorf("a buffer in /other must not appear in the /pkg view")
	}
}

// Closing one file re-analyzes its still-open siblings, so their diagnostics do not go stale.
func TestCloseRefreshesOpenSibling(t *testing.T) {
	const aURI, bURI = "file:///pkg/a.goal", "file:///pkg/b.goal"
	aSrc := `package p

type Src struct {
	ID string
}

type Spec struct {
	ID string
}
`
	bSrc := `package p

derive func mk(s Src) Spec
`
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(map[string][]fileSrc{
		"/pkg": {{path: "/pkg/a.goal", src: aSrc}, {path: "/pkg/b.goal", src: bSrc}},
	}), fakeResolver(nil))
	s.testOpen(t, aURI, aSrc, 1)
	s.testOpen(t, bURI, bSrc, 1)

	out.Reset()
	s.testClose(t, aURI)

	// Closing A clears A and re-publishes B (still resolvable from A's on-disk text).
	diags := latestDiagnostics(t, &out)
	if _, ok := diags[bURI]; !ok {
		t.Errorf("closing A should re-publish its open sibling B")
	}
}

// The editor's package path and the CLI's AnalyzePackageInDir produce the same diagnostics
// for a saved, conflict-free package.
func TestParityWithCLI(t *testing.T) {
	const aURI, bURI = "file:///pkg/a.goal", "file:///pkg/b.goal"
	aSrc := `package p

type Src struct {
	ID string
}

type Spec struct {
	ID    string
	Extra string
}
`
	bSrc := `package p

derive func mk(s Src) Spec
`
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(map[string][]fileSrc{
		"/pkg": {{path: "/pkg/a.goal", src: aSrc}, {path: "/pkg/b.goal", src: bSrc}},
	}), fakeResolver(nil))
	s.testOpen(t, aURI, aSrc, 1)
	s.testOpen(t, bURI, bSrc, 1)
	editor := latestDiagnostics(t, &out)

	// CLI path over the same sources, path-sorted (a before b).
	perFile, err := check.AnalyzePackageInDir([]string{aSrc, bSrc}, "/pkg")
	if err != nil {
		t.Fatalf("AnalyzePackageInDir: %v", err)
	}
	if len(perFile[1]) != len(editor[bURI]) {
		t.Fatalf("B diagnostic count: editor %d, cli %d", len(editor[bURI]), len(perFile[1]))
	}
	for i, d := range perFile[1] {
		if editor[bURI][i].Code != d.Code {
			t.Errorf("B diag %d: editor %q, cli %q", i, editor[bURI][i].Code, d.Code)
		}
	}
}

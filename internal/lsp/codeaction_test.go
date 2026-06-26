package lsp

import (
	"bytes"
	"encoding/json"
	"testing"

	"goal/internal/fix"
)

// tupleSrc is a plain-Go (T, error) helper with manual propagation — exactly what the
// idiomatize fixer rewrites into Result + `?`.
const tupleSrc = "package app\n\n" +
	"import \"os\"\n\n" +
	"func load(p string) ([]byte, error) {\n" +
	"\tf, err := os.ReadFile(p)\n" +
	"\tif err != nil {\n" +
	"\t\treturn nil, err\n" +
	"\t}\n" +
	"\treturn f, nil\n" +
	"}\n"

func codeActionParams(uri string, only ...string) json.RawMessage {
	raw, _ := json.Marshal(CodeActionParams{
		TextDocument: textDocumentIdentifier{URI: uri},
		Context:      CodeActionContext{Only: only},
	})
	return raw
}

// A buffer with a fixable plain-Go pattern yields one source.fixAll.goal action whose
// version-pinned edit replaces the whole document with the idiomatic rewrite.
func TestCodeActionOffersIdiomatize(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, tupleSrc, 7)

	actions := s.codeActions(codeActionParams(uri))
	if len(actions) != 1 {
		t.Fatalf("want 1 action, got %d", len(actions))
	}
	a := actions[0]
	if a.Kind != "source.fixAll.goal" || a.Title == "" {
		t.Fatalf("unexpected action metadata: %+v", a)
	}
	if a.Edit == nil || len(a.Edit.DocumentChanges) != 1 || len(a.Edit.DocumentChanges[0].Edits) != 1 {
		t.Fatalf("expected a single full-document edit, got %+v", a.Edit)
	}
	tde := a.Edit.DocumentChanges[0]
	if tde.TextDocument.Version != 7 {
		t.Errorf("edit version = %d, want pinned 7", tde.TextDocument.Version)
	}
	want, _, _ := fix.File(tupleSrc)
	if tde.Edits[0].NewText != want {
		t.Errorf("edit text does not match fix.File output")
	}
}

// A file with nothing to idiomatize offers no action.
func TestCodeActionNoOpReturnsNone(t *testing.T) {
	const uri = "file:///pkg/clean.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, "package app\n", 1)
	if got := s.codeActions(codeActionParams(uri)); len(got) != 0 {
		t.Errorf("clean file should offer no action, got %d", len(got))
	}
}

// The action honors the client's kind filter: surfaced for an empty filter or a matching
// ancestor kind, withheld for an unrelated kind.
func TestCodeActionHonorsOnlyFilter(t *testing.T) {
	const uri = "file:///pkg/a.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, tupleSrc, 1)

	if len(s.codeActions(codeActionParams(uri, "source"))) != 1 {
		t.Error(`only=["source"] should surface the fix-all action`)
	}
	if len(s.codeActions(codeActionParams(uri, "source.fixAll"))) != 1 {
		t.Error(`only=["source.fixAll"] should surface the action`)
	}
	if len(s.codeActions(codeActionParams(uri, "quickfix"))) != 0 {
		t.Error(`only=["quickfix"] should not surface the action`)
	}
}

// A request for a document the server does not hold returns an empty result, not an error.
func TestCodeActionUnknownURI(t *testing.T) {
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	if got := s.codeActions(codeActionParams("file:///pkg/missing.goal")); len(got) != 0 {
		t.Errorf("unknown URI should yield no action, got %d", len(got))
	}
}

// A fixable pattern inside otherwise broken source never panics.
func TestCodeActionBrokenSourceNoPanic(t *testing.T) {
	const uri = "file:///pkg/broken.goal"
	var out bytes.Buffer
	s := testServer(&out, fakeFiles(nil), fakeResolver(nil))
	s.upsert(uri, tupleSrc+"\nfunc oops( {\n", 1)
	_ = s.codeActions(codeActionParams(uri)) // must not panic
}

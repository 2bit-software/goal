package lsp

import "testing"

// A file: URI decodes to its filesystem path (percent-escapes included); any other URI is
// rejected so the caller falls back to single-file analysis.
func TestURIToPath(t *testing.T) {
	cases := []struct {
		uri      string
		wantPath string
		wantOK   bool
	}{
		{"file:///a/b.goal", "/a/b.goal", true},
		{"file:///a/p%20q/x.goal", "/a/p q/x.goal", true},
		{"file:///a/./b/../c.goal", "/a/c.goal", true},
		{"untitled:Untitled-1", "", false},
		{"http://example.com/x.goal", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := uriToPath(c.uri)
		if ok != c.wantOK || got != c.wantPath {
			t.Errorf("uriToPath(%q) = (%q, %v), want (%q, %v)", c.uri, got, ok, c.wantPath, c.wantOK)
		}
	}
}

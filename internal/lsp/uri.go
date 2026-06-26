package lsp

import (
	"net/url"
	"path/filepath"
)

// uriToPath converts a document URI to a local filesystem path. It reports ok=false for any
// URI that is not a usable file: path — a different scheme, a parse failure, or an empty
// path — which the caller treats as "no package directory" and analyzes the buffer alone.
// Percent-encoded characters (e.g. spaces in a macOS path) are decoded.
func uriToPath(uri string) (path string, ok bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return "", false
	}
	if u.Path == "" {
		return "", false
	}
	return filepath.Clean(u.Path), true
}

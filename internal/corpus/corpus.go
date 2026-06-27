// Package corpus defines a runner-independent model of the golden test corpus.
//
// A [Case] describes a single golden case (a transpile pair, a checker case, or
// a doctest sidecar) without reference to where its source files physically
// live. A [Manifest] is a collection of cases, and [Load] reads a manifest from
// a JSON file on disk. This decouples the golden suite from package layout so
// any runner — the current pipeline, the future AST backend, the self-hosted
// compiler, or the interpreter — can be judged against the same yardstick.
//
// This package is Phase 0.1 of REWRITE-ARCHITECTURE.md: it provides only the
// data model and loader. Generating the manifest over the existing corpus and
// building the runners that consume it are later stories.
package corpus

import (
	"encoding/json"
	"fmt"
	"os"
)

// Kind classifies what a [Case] exercises.
type Kind string

const (
	// KindTranspile compares generated Go against an expected golden.
	KindTranspile Kind = "transpile"
	// KindCheck exercises the checker against inline // want markers.
	KindCheck Kind = "check"
	// KindDoctest compares an emitted _test.go doctest sidecar.
	KindDoctest Kind = "doctest"
)

// Mode describes whether a [Case] is a single file or a multi-file package.
type Mode string

const (
	// ModeFile is a single-source case.
	ModeFile Mode = "file"
	// ModePackage is a multi-file package case.
	ModePackage Mode = "package"
)

// Normalize describes how a [Case]'s expected output is compared.
type Normalize string

const (
	// NormalizeNone compares expected output byte-for-byte.
	NormalizeNone Normalize = "none"
	// NormalizeGofmt gofmt-normalizes both sides before comparing.
	NormalizeGofmt Normalize = "gofmt"
)

// Case is a single golden case in the corpus. It is described independently of
// where its source files live, so runners can consume it regardless of package
// layout.
type Case struct {
	// ID uniquely identifies the case within a manifest.
	ID string `json:"id"`
	// Kind classifies what the case exercises.
	Kind Kind `json:"kind"`
	// Input is the goal source (or a path to it) under test.
	Input string `json:"input"`
	// Expected is the expected result (golden output, want markers, or sidecar).
	Expected string `json:"expected"`
	// Mode is whether the case is a single file or a package.
	Mode Mode `json:"mode"`
	// Normalize is how Expected is compared against the produced output.
	Normalize Normalize `json:"normalize"`
}

// Manifest is a collection of golden [Case]s. It is a struct (rather than a bare
// slice) so it can carry corpus-level metadata in later stories without breaking
// callers.
type Manifest struct {
	// Cases are the golden cases indexed by this manifest.
	Cases []Case `json:"cases"`
}

// Load reads and decodes a JSON manifest from path. It returns a descriptive
// error if the file cannot be read or its contents are not valid JSON.
func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("corpus: reading manifest %q: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("corpus: parsing manifest %q: %w", path, err)
	}
	return m, nil
}

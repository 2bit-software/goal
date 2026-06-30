package corpus

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"

	"goal/internal/compiler/pipeline"
)

// Transpiler lowers goal source to its transpile output. It is the seam that
// lets the corpus judge any front-end — the AST backend, a future self-hosted
// compiler — by the same yardstick. backend.Transpile satisfies it via
// [TranspilerFunc].
type Transpiler interface {
	Transpile(src string) (pipeline.Output, error)
}

// TranspilerFunc adapts a plain transpile function to the [Transpiler]
// interface, so a free function such as backend.Transpile can be passed where a
// Transpiler is expected.
type TranspilerFunc func(src string) (pipeline.Output, error)

// Transpile calls the wrapped function.
func (f TranspilerFunc) Transpile(src string) (pipeline.Output, error) { return f(src) }

// RunTranspile executes one [KindTranspile] Case against tp. It reads the case's
// Input and Expected files relative to root, transpiles the input, and compares
// the produced Go against the golden after gofmt-normalizing BOTH sides — so
// equivalent Go differing only in formatting still matches.
//
// A case passes when the normalized golden equals the normalized main output
// (Output.Go) or, when present, the normalized doctest sidecar (Output.Test).
// The sidecar fallback exists because the manifest classifies the feature-11
// doctest examples as transpile cases whose golden is the emitted _test.go
// rather than the main Go output; honoring both keeps that classification intact
// without a dedicated doctest runner (a later story).
//
// It returns a descriptive, case-identified error on any read failure, a
// transpile failure, a gofmt failure, or an output mismatch; it returns nil when
// the case passes.
func RunTranspile(root string, c Case, tp Transpiler) error {
	if c.Kind != KindTranspile {
		return fmt.Errorf("corpus: RunTranspile: case %q is kind %q, not %q", c.ID, c.Kind, KindTranspile)
	}

	srcPath := filepath.Join(root, filepath.FromSlash(c.Input))
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("corpus: case %q: reading input: %w", c.ID, err)
	}

	out, err := tp.Transpile(string(src))
	if err != nil {
		return fmt.Errorf("corpus: case %q: transpile: %w", c.ID, err)
	}

	wantRaw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.Expected)))
	if err != nil {
		return fmt.Errorf("corpus: case %q: reading expected: %w", c.ID, err)
	}
	want, err := gofmtNormalize(string(wantRaw))
	if err != nil {
		return fmt.Errorf("corpus: case %q: gofmt expected: %w", c.ID, err)
	}

	gotGo, err := gofmtNormalize(out.Go)
	if err != nil {
		return fmt.Errorf("corpus: case %q: gofmt output: %w", c.ID, err)
	}
	if gotGo == want {
		return nil
	}

	if out.Test != "" {
		gotTest, err := gofmtNormalize(out.Test)
		if err != nil {
			return fmt.Errorf("corpus: case %q: gofmt doctest sidecar: %w", c.ID, err)
		}
		if gotTest == want {
			return nil
		}
	}

	return fmt.Errorf("corpus: case %q: output mismatch\n--- got ---\n%s\n--- want ---\n%s", c.ID, gotGo, want)
}

// gofmtNormalize returns src formatted by gofmt, so two equivalent Go sources
// that differ only in formatting normalize to the same text.
func gofmtNormalize(src string) (string, error) {
	out, err := format.Source([]byte(src))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

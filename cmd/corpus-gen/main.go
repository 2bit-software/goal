// Command corpus-gen generates corpus/manifest.json by indexing the existing
// golden corpus (feature examples, top-level testdata, and testdata/check).
//
// It moves no source file. Run it from the repository root, or via
// `go generate ./internal/corpus`.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"goal/internal/corpus"
)

func main() {
	root := flag.String("root", ".", "repository root to index")
	out := flag.String("out", filepath.Join("corpus", "manifest.json"), "output manifest path")
	flag.Parse()

	if err := run(*root, *out); err != nil {
		fmt.Fprintln(os.Stderr, "corpus-gen:", err)
		os.Exit(1)
	}
}

func run(root, out string) error {
	m, err := corpus.Generate(root)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	data = append(data, '\n')

	outPath := out
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(root, out)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", outPath, err)
	}
	fmt.Printf("wrote %s (%d cases)\n", outPath, len(m.Cases))
	return nil
}

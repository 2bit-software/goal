package corpus

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunDoctest executes one [KindDoctest] Case against tp. It reads the case's
// Input relative to root, transpiles it, and compares the emitted doctest
// sidecar (Output.Test) against the golden at Expected after gofmt-normalizing
// BOTH sides — so a sidecar differing from the golden only in formatting still
// matches.
//
// Unlike [RunTranspile] (which compares the main Go output and falls back to the
// sidecar), this runner asserts the sidecar specifically: the doctest tier
// exists precisely to judge the emitted _test.go that the doctest examples
// golden.
//
// It returns a descriptive, case-identified error on a wrong-kind case, a read
// failure, a transpile failure, a gofmt failure, an empty sidecar, or an output
// mismatch; it returns nil when the case passes.
func RunDoctest(root string, c Case, tp Transpiler) error {
	if c.Kind != KindDoctest {
		return fmt.Errorf("corpus: RunDoctest: case %q is kind %q, not %q", c.ID, c.Kind, KindDoctest)
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
	if out.Test == "" {
		return fmt.Errorf("corpus: case %q: transpile produced no doctest sidecar", c.ID)
	}

	wantRaw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.Expected)))
	if err != nil {
		return fmt.Errorf("corpus: case %q: reading expected: %w", c.ID, err)
	}
	want, err := gofmtNormalize(string(wantRaw))
	if err != nil {
		return fmt.Errorf("corpus: case %q: gofmt expected: %w", c.ID, err)
	}

	got, err := gofmtNormalize(out.Test)
	if err != nil {
		return fmt.Errorf("corpus: case %q: gofmt doctest sidecar: %w", c.ID, err)
	}
	if got == want {
		return nil
	}

	return fmt.Errorf("corpus: case %q: doctest sidecar mismatch\n--- got ---\n%s\n--- want ---\n%s", c.ID, got, want)
}

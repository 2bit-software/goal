package corpus

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RunCompile executes one [KindTranspile] Case behaviorally: rather than
// comparing the generated Go to a golden (that is RunTranspile's job), it proves
// the generated Go actually compiles. It transpiles the case Input via tp,
// writes the resulting Output.Go into a freshly created, isolated temp module
// (a minimal go.mod plus one .go file), and runs `go build` then `go vet`
// against that module. The source tree is never touched.
//
// This is the behavioral conformance tier: it judges output by behavior, not by
// Go spelling, so it is implementation-independent across front-ends.
//
// It returns a descriptive, case-identified error on any read, transpile, temp-
// module, build, or vet failure — including the go tool's combined output so the
// failing case and reason are obvious — and nil when the generated Go builds and
// vets cleanly.
func RunCompile(root string, c Case, tp Transpiler) error {
	if c.Kind != KindTranspile {
		return fmt.Errorf("corpus: RunCompile: case %q is kind %q, not %q", c.ID, c.Kind, KindTranspile)
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

	tmp, err := os.MkdirTemp("", "goal-corpus-compile-*")
	if err != nil {
		return fmt.Errorf("corpus: case %q: temp module: %w", c.ID, err)
	}
	defer os.RemoveAll(tmp)

	// A minimal module is enough: the generated Go is zero-dependency (stdlib
	// imports only), so go build/vet resolve everything offline. The module path
	// and the single file name are arbitrary; the package clause comes from the
	// generated source itself.
	const goMod = "module goalcorpus\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		return fmt.Errorf("corpus: case %q: writing go.mod: %w", c.ID, err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "case.go"), []byte(out.Go), 0o644); err != nil {
		return fmt.Errorf("corpus: case %q: writing generated Go: %w", c.ID, err)
	}

	for _, verb := range []string{"build", "vet"} {
		cmd := exec.Command("go", verb, "./...")
		cmd.Dir = tmp
		if combined, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("corpus: case %q: go %s failed: %w\n--- generated Go ---\n%s\n--- go %s output ---\n%s",
				c.ID, verb, err, out.Go, verb, combined)
		}
	}

	return nil
}

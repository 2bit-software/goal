package corpus

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RunDoctestExec executes one [KindDoctest] Case behaviorally: it proves the
// doctest sidecar actually passes when run, not merely that it compiles. It
// transpiles the case Input via tp, writes BOTH the generated package
// (Output.Go) and the generated doctest sidecar (Output.Test) into a freshly
// created, isolated temp module, and runs `go test ./...` against that module.
// The source tree is never touched.
//
// Where [RunDoctest] compares the emitted sidecar text to a golden and
// [RunCompile] only builds + vets the main output, this runner is the strict
// behavioral tier for doctests: a doctest example whose expected value is wrong
// at runtime turns the suite red here.
//
// It returns a descriptive, case-identified error on a wrong-kind case, a read
// failure, a transpile failure, an empty sidecar, a temp-module write failure,
// or a `go test` failure — including the go tool's combined output and both
// generated sources so the failing case and reason are obvious — and nil when
// the doctest passes.
func RunDoctestExec(root string, c Case, tp Transpiler) error {
	if c.Kind != KindDoctest {
		return fmt.Errorf("corpus: RunDoctestExec: case %q is kind %q, not %q", c.ID, c.Kind, KindDoctest)
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

	tmp, err := os.MkdirTemp("", "goal-corpus-doctest-*")
	if err != nil {
		return fmt.Errorf("corpus: case %q: temp module: %w", c.ID, err)
	}
	defer os.RemoveAll(tmp)

	// A minimal module suffices: the generated Go is zero-dependency (stdlib
	// imports only), so `go test` resolves everything offline. The module path
	// and file names are arbitrary; the package clause comes from the generated
	// source, and the sidecar shares that package so the test sees the package's
	// unexported identifiers.
	const goMod = "module goalcorpus\n\ngo 1.26\n"
	files := map[string]string{
		"go.mod":       goMod,
		"case.go":      out.Go,
		"case_test.go": out.Test,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(body), 0o644); err != nil {
			return fmt.Errorf("corpus: case %q: writing %s: %w", c.ID, name, err)
		}
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	if combined, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("corpus: case %q: go test failed: %w\n--- generated Go ---\n%s\n--- generated test ---\n%s\n--- go test output ---\n%s",
			c.ID, err, out.Go, out.Test, combined)
	}

	return nil
}

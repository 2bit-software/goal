package corpus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goal/internal/compiler/interp"
)

// RunInterp executes one [KindDoctest] Case behaviorally through the goscript
// interpreter. It reads the case's Input goal source and runs every `///  >>>`
// doctest example through internal/interp.RunDoctests, comparing the evaluated
// runtime result against the documented expected value — the SAME observable
// behavior the Go behavioral doctest tier ([RunDoctestExec]) judges, but proven
// by interpretation rather than by transpiling and running `go test`.
//
// It is the interpreter's entry into the implementation-independent behavioral
// conformance tier (REWRITE-ARCHITECTURE.md §6): the corpus is the one yardstick
// the Go backend and the interpreter are both measured against. Unlike
// RunDoctestExec it spawns no toolchain — it evaluates the examples in-process —
// so it runs in a host with no Go toolchain.
//
// It returns a descriptive, case-identified error on a wrong-kind case, a read
// failure, a parse/eval failure, a case that yields no doctest examples, or any
// example whose evaluated result does not match its expected value; it returns
// nil when every example matches.
func RunInterp(root string, c Case) error {
	if c.Kind != KindDoctest {
		return fmt.Errorf("corpus: RunInterp: case %q is kind %q, not %q", c.ID, c.Kind, KindDoctest)
	}

	srcPath := filepath.Join(root, filepath.FromSlash(c.Input))
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("corpus: case %q: reading input: %w", c.ID, err)
	}

	failures, ran, err := interp.RunDoctests(string(src))
	if err != nil {
		return fmt.Errorf("corpus: case %q: %w", c.ID, err)
	}
	if ran == 0 {
		return fmt.Errorf("corpus: case %q: produced no doctest examples", c.ID)
	}
	if len(failures) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "corpus: case %q: %d doctest example(s) failed under interpretation:", c.ID, len(failures))
		for _, f := range failures {
			fmt.Fprintf(&b, "\n  %s", f.String())
		}
		return fmt.Errorf("%s", b.String())
	}

	return nil
}

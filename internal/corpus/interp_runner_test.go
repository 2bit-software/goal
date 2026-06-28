package corpus

// TestInterpRunner is the interpreter's behavioral doctest tier: every doctest
// case in the committed manifest is run through RunInterp, which evaluates each
// `///  >>>` example through the goscript interpreter and asserts the runtime
// result matches the documented expected value. It proves the interpreter against
// the SAME implementation-independent yardstick as the Go backend's doctest tier
// (RunDoctestExec), but by interpretation in-process (no Go toolchain).
//
// It fails loudly if the manifest yields no doctest cases, so an empty or
// mis-generated manifest cannot masquerade as green.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInterpRunner(t *testing.T) {
	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindDoctest {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunInterp(repoRoot, c); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no doctest cases", manifestPath)
	}
}

func TestInterpRunnerWrongKind(t *testing.T) {
	c := Case{ID: "x", Kind: KindTranspile, Input: "whatever.goal"}
	err := RunInterp(repoRoot, c)
	if err == nil {
		t.Fatalf("RunInterp on a %s case: expected an error", KindTranspile)
	}
}

func TestInterpRunnerMutatedExpectedFails(t *testing.T) {
	// A doctest whose expected value is wrong must make RunInterp fail loudly — no
	// silent pass. Write a temp .goal fixture rather than mutating the corpus.
	dir := t.TempDir()
	const src = `package mathx

/// >>> add(2, 3)
/// 6
func add(a int, b int) int {
	return a + b
}
`
	if err := os.WriteFile(filepath.Join(dir, "add.goal"), []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	c := Case{ID: "mutated", Kind: KindDoctest, Mode: ModeFile, Input: "add.goal"}
	err := RunInterp(dir, c)
	if err == nil {
		t.Fatalf("RunInterp on a mutated-expected doctest: expected an error")
	}
}

func TestInterpRunnerNoExamplesFails(t *testing.T) {
	// A doctest-kind case whose source carries no `>>>` examples is a loud failure,
	// not a vacuous green.
	dir := t.TempDir()
	const src = `package p

/// just prose, no doctest
func f() int {
	return 0
}
`
	if err := os.WriteFile(filepath.Join(dir, "none.goal"), []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	c := Case{ID: "none", Kind: KindDoctest, Mode: ModeFile, Input: "none.goal"}
	if err := RunInterp(dir, c); err == nil {
		t.Fatalf("RunInterp on a doctest case with no examples: expected an error")
	}
}

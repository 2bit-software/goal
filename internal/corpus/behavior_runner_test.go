package corpus

import (
	"testing"

	"goal/internal/pipeline"
)

// TestCompileRunner is the behavioral conformance tier: every transpile case in
// the committed manifest is transpiled through pipeline.Transpile and its
// generated Go must build and vet cleanly in an isolated temp module. It proves
// conformance by behavior (the Go compiles) rather than by exact spelling.
//
// It spawns the go toolchain per case, so it is skipped under -short; the full
// `go test ./... -count=1` gate exercises it.
func TestCompileRunner(t *testing.T) {
	if testing.Short() {
		t.Skip("behavioral tier spawns the go toolchain per case")
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	tp := TranspilerFunc(pipeline.Transpile)
	seen := 0
	for _, c := range m.Cases {
		if c.Kind != KindTranspile {
			continue
		}
		seen++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunCompile(repoRoot, c, tp); err != nil {
				t.Errorf("%v", err)
			}
		})
	}
	if seen == 0 {
		t.Fatalf("manifest %q contains no transpile cases", manifestPath)
	}
}

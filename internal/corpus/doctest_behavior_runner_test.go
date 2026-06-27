package corpus

import (
	"testing"

	"goal/internal/pipeline"
)

// TestDoctestExecRunner is the behavioral doctest tier: every doctest case in
// the committed manifest is transpiled through pipeline.Transpile, and its
// generated package plus generated _test.go sidecar must pass `go test` in an
// isolated temp module. It proves doctest behavior by executing it, not merely
// by compiling or text-matching the sidecar.
//
// It spawns the go toolchain per case, so it is skipped under -short; the full
// `go test ./... -count=1` gate exercises it. It fails loudly if the manifest
// yields no doctest cases, so an empty or mis-generated manifest cannot
// masquerade as green.
func TestDoctestExecRunner(t *testing.T) {
	if testing.Short() {
		t.Skip("behavioral doctest tier spawns the go toolchain per case")
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	tp := TranspilerFunc(pipeline.Transpile)
	ran := 0
	for _, c := range m.Cases {
		if c.Kind != KindDoctest {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunDoctestExec(repoRoot, c, tp); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no doctest cases", manifestPath)
	}
}

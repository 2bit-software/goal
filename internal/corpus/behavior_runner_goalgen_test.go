package corpus

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"goal/internal/compiler/pipeline"
)

// goalGenTranspiler drives a goal-built compiler binary (cmd/goal-gen, compiled
// from the goal-SOURCED goal/internal/compiler/* packages) as the corpus
// Transpiler: it shells out `goal-gen transpile -`, feeding the goal source on
// stdin and capturing the lowered Go on stdout. This is the seam that lets the
// behavioral tier judge the goal-written front-end exactly as it judges the legacy
// Go one — same cases, same build+vet oracle, different compiler.
type goalGenTranspiler struct{ bin string }

// Transpile satisfies the corpus Transpiler interface by running the goal-built
// binary on src and returning its stdout as the generated Go. Any non-zero exit
// (including the binary's own "goal-gen: ..." diagnostics) is surfaced with stderr.
func (g goalGenTranspiler) Transpile(src string) (pipeline.Output, error) {
	cmd := exec.Command(g.bin, "transpile", "-")
	cmd.Stdin = strings.NewReader(src)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return pipeline.Output{}, fmt.Errorf("goal-gen transpile failed: %w\n--- stderr ---\n%s", err, stderr.String())
	}
	return pipeline.Output{Go: stdout.String()}, nil
}

// TestCompileRunnerGoalGen is the goal-built half of the behavioral conformance
// tier (US-005). It mirrors TestCompileRunner but, instead of the in-process legacy
// backend.Transpile, it transpiles every file-mode transpile case through the
// goal-built compiler binary named by the GOAL_GEN_BIN environment variable, then
// builds and vets the result in an isolated temp module. Passing it proves the
// goal-sourced compiler is correct end to end over the corpus.
//
// It is skipped unless GOAL_GEN_BIN is set, so the normal `go test ./...` gate does
// not require a pre-built binary; `task corpus-goalgen` builds cmd/goal-gen and runs
// this test with GOAL_GEN_BIN pointed at it.
func TestCompileRunnerGoalGen(t *testing.T) {
	bin := os.Getenv("GOAL_GEN_BIN")
	if bin == "" {
		t.Skip("set GOAL_GEN_BIN to a goal-built compiler binary (see `task corpus-goalgen`)")
	}
	if testing.Short() {
		t.Skip("behavioral tier spawns the go toolchain per case")
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	tp := goalGenTranspiler{bin: bin}
	seen := 0
	for _, c := range m.Cases {
		if c.Kind != KindTranspile || c.Mode != ModeFile {
			continue // package-mode cases run through the package runner
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

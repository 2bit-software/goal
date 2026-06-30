package interp

// These tests are the LEGACY-ONLY portion of the gate suite: they cannot run
// against the goal-built interp via the internal/selfhost behavioral gate, so
// they live apart from gate_test.go (which the gate DOES run).
//
//   - TestRunDoesNotBlockOnWarning switches on sema.Warning / sema.Error, which
//     the ported internal/compiler/sema models as a sealed-interface `enum`
//     (Severity_Error / Severity_Warning), not the legacy comparable consts — so
//     the rewritten copy would not compile against the ported sema.
//   - TestInterpHasNoGoTypesOrTypecheckDep shells out `go list -deps
//     goal/internal/interp`, a structural dependency-envelope assertion pinned to
//     the real module path (not the temp self-host module).
//
// Both run under `task check` against the legacy package; gate_test.go's two
// run-refusal tests are the gate-behavior coverage carried into the goal-built
// interp.

import (
	"os/exec"
	"strings"
	"testing"

	"goal/internal/parser"
	"goal/internal/sema"
)

// warningProgram matches an enum NOT declared in this file: sema cannot prove
// exhaustiveness, so it emits a WARNING (a located deferral), never an Error. The
// gate must NOT refuse on a warning — only violated guarantees block.
const warningProgram = `package main

func main() {
	s := External.A
	match s {
		External.A => return
	}
}
`

func TestRunDoesNotBlockOnWarning(t *testing.T) {
	// Sanity-check the fixture actually produces a Warning and no Error, so this
	// test exercises the warning path rather than a clean program.
	file, err := parser.ParseFile(warningProgram)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := sema.Resolve(file)
	sawWarning, sawError := false, false
	for _, d := range sema.Check(file, info) {
		switch d.Severity {
		case sema.Warning:
			sawWarning = true
		case sema.Error:
			sawError = true
		}
	}
	if !sawWarning {
		t.Fatalf("fixture produced no Warning; cannot exercise the warning path")
	}
	if sawError {
		t.Fatalf("fixture unexpectedly produced an Error; not a pure-warning case")
	}
	// The gate must not refuse on a warning. main reads `External.A`, which the
	// interpreter cannot resolve, so Run may still fail at eval time — but the
	// failure must NOT be the pre-run gate refusal.
	runErr := New(file, info).Run()
	if runErr != nil && strings.Contains(runErr.Error(), "refused before run") {
		t.Fatalf("gate refused a warning-only program: %v", runErr)
	}
}

// TestInterpHasNoGoTypesOrTypecheckDep pins the native-only dependency envelope:
// the interpreter validates exclusively through internal/sema, so its transitive
// dependencies must include neither the Go depth checker (go/types) nor the
// lowered-Go typecheck package.
func TestInterpHasNoGoTypesOrTypecheckDep(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "goal/internal/interp").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, out)
	}
	for _, dep := range strings.Fields(string(out)) {
		if dep == "go/types" {
			t.Errorf("internal/interp transitively depends on go/types — the interpreter must validate via native sema only")
		}
		if dep == "goal/internal/typecheck" {
			t.Errorf("internal/interp transitively depends on goal/internal/typecheck — the interpreter must validate via native sema only")
		}
	}
}

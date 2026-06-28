package parser

// AST snapshot tests (US-025): the parser's structural output for each goal
// construct is pinned to a checked-in s-expression golden. ast.Sexpr renders a
// parsed file to a deterministic, position-free textual form; this suite parses
// one representative input per goal construct, renders it, and compares against
// the golden under testdata/snapshots/. Run with -update-snapshots to
// regenerate the goldens after an intentional grammar change.

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"goal/internal/ast"
)

var updateSnapshots = flag.Bool("update-snapshots", false, "regenerate AST snapshot goldens")

// snapshotCases lists one representative input per goal-specific construct.
// Inputs live under features/NN/examples (cwd = internal/parser, so ../../).
var snapshotCases = []struct {
	name  string // golden file stem
	input string // path to the .goal input, relative to internal/parser
}{
	{"enum", "../../features/01-enums/examples/status.goal"},
	{"match", "../../features/02-match/examples/status_rest.goal"},
	{"question-prop", "../../features/05-question-prop/examples/qprop_bare.goal"},
	{"error-e", "../../features/06-error-e/examples/qclosed_match.goal"},
	{"implements", "../../features/07-implements/examples/value_recv.goal"},
	{"defaults", "../../features/08-no-zero-value/examples/defaults_primitives.goal"},
	{"assert", "../../features/10-assert/examples/message.goal"},
	{"doctests", "../../features/11-doctests/examples/add.goal"},
	{"derive-convert", "../../features/12-derive-convert/examples/from_storage.goal"},
}

func TestASTSnapshots(t *testing.T) {
	if len(snapshotCases) == 0 {
		t.Fatal("no snapshot cases declared")
	}
	for _, tc := range snapshotCases {
		t.Run(tc.name, func(t *testing.T) {
			src, err := os.ReadFile(tc.input)
			if err != nil {
				t.Fatalf("read input %s: %v", tc.input, err)
			}
			file, err := ParseFile(string(src))
			if err != nil {
				t.Fatalf("parse %s: %v", tc.input, err)
			}

			got := ast.Sexpr(file)

			// Determinism: rendering the same tree twice yields the same text.
			if again := ast.Sexpr(file); again != got {
				t.Fatalf("Sexpr is not deterministic for %s", tc.input)
			}

			golden := filepath.Join("testdata", "snapshots", tc.name+".sexpr")
			if *updateSnapshots {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatalf("mkdir snapshots: %v", err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden %s: %v", golden, err)
				}
				return
			}

			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden %s (run with -update-snapshots to create): %v", golden, err)
			}
			if got != string(want) {
				t.Errorf("AST snapshot mismatch for %s\n--- got ---\n%s\n--- want ---\n%s",
					tc.name, got, string(want))
			}
		})
	}
}

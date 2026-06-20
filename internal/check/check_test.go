package check

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// wantRe matches an inline expectation marker on a source line:
//
//	// want "substring the diagnostic message must contain"
//
// The marker's own line is the line the diagnostic is expected on. Multiple markers
// on one line are allowed. A `.goal` case with no markers asserts the program is
// CLEAN — any Error-severity diagnostic fails it.
var wantRe = regexp.MustCompile(`//\s*want\s+"([^"]*)"`)

// TestCases is the whole checker test harness: it walks testdata/check/**/*.goal,
// runs every registered check, and matches diagnostics against inline `// want`
// markers. Adding a case is dropping a `.goal` file under testdata/check/<feature>/
// — no edit to this file. The contract:
//
//   - Each `// want "sub"` marker must be satisfied by some diagnostic ON THAT LINE
//     whose message contains "sub".
//   - Every Error-severity diagnostic must be claimed by a marker on its line;
//     an unclaimed Error is an unexpected rejection and fails the case. (Warnings —
//     i.e. located deferrals — may go unclaimed, so negative cases stay easy to write.)
func TestCases(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "check")
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".goal") {
			files = append(files, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		t.Skip("no testdata/check yet — added per check by the loop in prompt.md")
	}
	if err != nil {
		t.Fatalf("walk testdata/check: %v", err)
	}
	for _, f := range files {
		t.Run(strings.TrimPrefix(f, root+string(filepath.Separator)), func(t *testing.T) {
			runCase(t, f)
		})
	}
}

func runCase(t *testing.T, path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	wants := parseWants(string(src)) // line -> expected substrings
	diags, err := Analyze(string(src))
	if err != nil {
		t.Fatalf("checker errored on %s: %v", path, err)
	}

	matched := map[int]map[int]bool{} // line -> index of want satisfied
	for _, d := range diags {
		line := OffsetToPosition(string(src), d.Pos).Line
		for i, sub := range wants[line] {
			if strings.Contains(d.Message, sub) {
				if matched[line] == nil {
					matched[line] = map[int]bool{}
				}
				matched[line][i] = true
			}
		}
		// An Error with no marker on its line is an unexpected rejection.
		if d.Severity == Error && len(wants[line]) == 0 {
			t.Errorf("%s: unexpected error at line %d: [%s] %s", path, line, d.Code, d.Message)
		}
	}
	for line, subs := range wants {
		for i, sub := range subs {
			if !matched[line][i] {
				t.Errorf("%s:%d: expected a diagnostic containing %q, none matched", path, line, sub)
			}
		}
	}
}

func parseWants(src string) map[int][]string {
	wants := map[int][]string{}
	for i, line := range strings.Split(src, "\n") {
		for _, m := range wantRe.FindAllStringSubmatch(line, -1) {
			wants[i+1] = append(wants[i+1], m[1])
		}
	}
	return wants
}

// TestRegistryRuns is a spine smoke test: every registered check runs without error
// on representative source, independent of whether any check is implemented yet.
func TestRegistryRuns(t *testing.T) {
	const src = `package p

enum Shape {
	Circle { r: float64 }
	Rect   { w: float64, h: float64 }
}
`
	if _, err := Analyze(src); err != nil {
		t.Fatalf("checker spine errored on valid source: %v", err)
	}
}

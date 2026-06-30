package corpus

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"goal/internal/sema"
)

// wantMarkerRe matches an inline expectation marker on a source line:
//
//	// want "substring the diagnostic message must contain"
//
// The marker's own line is the line the diagnostic is expected on. Multiple
// markers on one line are allowed. These semantics mirror the existing checker
// harness (internal/check/check_test.go) so a check case behaves identically
// whether driven by that harness or by this runner.
var wantMarkerRe = regexp.MustCompile(`//\s*want\s+"([^"]*)"`)

// RunCheck executes one [KindCheck] Case against the checker function ck. It reads
// the case's Input (a .goal file relative to root), runs the checker, and matches
// the produced diagnostics against the inline // want markers in the source. ck is
// passed directly (e.g. [SemaCheck]) — there is one checker front-end now that the
// legacy lexical checker is deleted, so no adapter interface is needed. The contract:
//
//   - Each `// want "sub"` marker must be satisfied by some diagnostic ON THAT
//     LINE whose message contains "sub"; an unsatisfied marker fails the case.
//   - Every Error-severity diagnostic must be claimed by a marker on its line;
//     an unclaimed Error is an unexpected rejection and fails the case.
//   - Warning-severity diagnostics (located deferrals) may go unclaimed, so
//     negative cases stay easy to write.
//
// It returns a descriptive, case-identified error on a read failure, a checker
// internal error, an unsatisfied marker, or an unclaimed Error; it returns nil
// when the case passes.
func RunCheck(root string, c Case, ck func(src string) ([]sema.Diagnostic, error)) error {
	if c.Kind != KindCheck {
		return fmt.Errorf("corpus: RunCheck: case %q is kind %q, not %q", c.ID, c.Kind, KindCheck)
	}

	srcPath := filepath.Join(root, filepath.FromSlash(c.Input))
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("corpus: case %q: reading input: %w", c.ID, err)
	}
	src := string(srcBytes)

	diags, err := ck(src)
	if err != nil {
		return fmt.Errorf("corpus: case %q: check: %w", c.ID, err)
	}

	wants := parseWantMarkers(src) // line -> expected substrings

	matched := map[int]map[int]bool{} // line -> set of satisfied want indices
	for _, d := range diags {
		line := d.Pos.Line
		for i, sub := range wants[line] {
			if strings.Contains(d.Message, sub) {
				if matched[line] == nil {
					matched[line] = map[int]bool{}
				}
				matched[line][i] = true
			}
		}
		// An Error with no marker on its line is an unexpected rejection.
		// compiler/sema models Severity as a sealed interface, so test the
		// concrete variant rather than comparing against a const.
		_, isError := d.Severity.(sema.Severity_Error)
		if isError && len(wants[line]) == 0 {
			return fmt.Errorf("corpus: case %q: unexpected error at line %d: [%s] %s",
				c.ID, line, d.Code, d.Message)
		}
	}

	for line, subs := range wants {
		for i, sub := range subs {
			if !matched[line][i] {
				return fmt.Errorf("corpus: case %q: line %d: expected a diagnostic containing %q, none matched",
					c.ID, line, sub)
			}
		}
	}
	return nil
}

// parseWantMarkers returns the inline // want substrings keyed by their 1-based
// source line.
func parseWantMarkers(src string) map[int][]string {
	wants := map[int][]string{}
	for i, line := range strings.Split(src, "\n") {
		for _, m := range wantMarkerRe.FindAllStringSubmatch(line, -1) {
			wants[i+1] = append(wants[i+1], m[1])
		}
	}
	return wants
}

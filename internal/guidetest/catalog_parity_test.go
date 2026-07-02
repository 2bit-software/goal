// Package guidetest holds Go-only tests for the guide package that cannot live
// beside guide.goal, because internal/guide is an emitted directory (its .go is
// generated from guide.goal / catalog.goal and verify-generated diffs it two-way,
// so an extra hand-written *_test.go there would trip the drift gate).
package guidetest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"goal/internal/guide"
)

// codeSourceDirs are the compiler source directories that emit stable diagnostic
// codes, relative to this test's package directory (internal/guidetest):
//
//   - lexer     — the 00-lex codes (unterminated-string, invalid-number-literal, …)
//   - sema      — the lexical-stage checker codes
//   - typecheck — the typed depth-stage codes
//   - backend   — the emitter, which emits question-arity-fallback as a Warning
//
// Scope note: diagnostic codes emitted by the CLI layer — [syntax] from the
// parse-error renderer and [go-build] from the build wrapper, both in cmd/goal —
// are intentionally NOT scanned here; this gate covers the checker/emitter
// catalog only.
var codeSourceDirs = []string{
	"../lexer",
	"../sema",
	"../typecheck",
	"../backend",
}

// nonCodeSkip lists kebab-case string literals in the scanned dirs that match the
// code shape but are NOT diagnostic codes, so the extractor must ignore them:
//
//   - "go-types"  — a Feature label (Feature: "go-types" in typecheck/gotypes);
//     every other Feature label starts with a digit ("00-lex", "05-question-prop")
//     and is excluded by the leading-letter rule, so this is the lone exception.
//   - "some-addr", "some-box" — Option lowering-kind tags in backend/lower.
//
// This is the escape hatch for future non-code kebab literals: add them here.
var nonCodeSkip = map[string]bool{
	"go-types":  true,
	"some-addr": true,
	"some-box":  true,
}

// codeLiteral matches the stable diagnostic-code shape: lowercase kebab-case with
// at least one hyphen, starting with a letter (so digit-leading feature strings
// like "00-lex" do not match).
var codeLiteral = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)+$`)

// TestDiagnosticCatalogMatchesSource proves the guide's diagnostic catalog stays
// in exact sync with the codes the checker and emitter actually emit: every
// emitted code is documented, and every documented code has an emitting source.
// It fails, naming the offending code, when either side drifts — so a new
// diagnostic code cannot ship undocumented, and a stale catalog entry cannot
// linger after its code is removed.
func TestDiagnosticCatalogMatchesSource(t *testing.T) {
	source := extractSourceCodes(t)
	catalog := guide.CatalogCodes()

	// Direction 1: emitted-in-source but missing from the catalog (undocumented).
	for code := range source {
		if !catalog[code] {
			t.Errorf("diagnostic code %q is emitted in source but not documented in the guide catalog (add a diagDoc entry in internal/guide/catalog.goal)", code)
		}
	}

	// Direction 2: documented in the catalog but no source emits it (stale entry).
	for code := range catalog {
		if !source[code] {
			t.Errorf("diagnostic code %q is documented in the guide catalog but no source in %v emits it (remove the stale diagDoc entry, or add the code to nonCodeSkip / codeSourceDirs if it moved)", code, codeSourceDirs)
		}
	}
}

// extractSourceCodes scans every non-test *.go file in codeSourceDirs and returns
// the set of kebab-case diagnostic-code string literals it finds, minus nonCodeSkip.
func extractSourceCodes(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, dir := range codeSourceDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		scanned := 0
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			scanned++
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read %s: %v", f, err)
			}
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, f, src, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", f, err)
			}
			ast.Inspect(node, func(n ast.Node) bool {
				bl, ok := n.(*ast.BasicLit)
				if !ok || bl.Kind != token.STRING {
					return true
				}
				v, err := strconv.Unquote(bl.Value)
				if err != nil {
					return true
				}
				if codeLiteral.MatchString(v) && !nonCodeSkip[v] {
					out[v] = true
				}
				return true
			})
		}
		// A wrong relative path would silently yield zero codes and let the gate
		// pass vacuously — fail loudly instead.
		if scanned == 0 {
			t.Fatalf("no non-test .go files found in %s (wrong relative path?)", dir)
		}
	}
	return out
}

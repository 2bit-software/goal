package corpus

// US-021: `?` applied to a package-qualified call (`pkg.Fn()?`) must be checked
// against the foreign function signature EnrichForeign collects (keyed `alias.Func`
// in info.FuncSignatures), so misuse of `?` on a cross-package call is caught
// instead of silently skipped. When the foreign sig is genuinely unavailable,
// checking stays permissive.
//
// Like staledep_test.go, this is a handwritten Go test in the Go-only corpus infra
// (NOT a testdata/check fixture, which the recursive walker would mis-index per
// file). It drives sema.AnalyzePackageInDirWith directly with a fake DirResolver
// (the shared resolverTo helper from staledep_test.go) pointed at the fixture dep.

import (
	"path/filepath"
	"testing"

	"goal/internal/sema"
)

// hasQuestionCalleeNoError reports whether any diagnostic across the analyzed files
// carries the question-callee-no-error code.
func hasQuestionCalleeNoError(perFile [][]sema.Diagnostic) bool {
	for _, diags := range perFile {
		for _, d := range diags {
			if d.Code == "question-callee-no-error" {
				return true
			}
		}
	}
	return false
}

// TestForeignQuestionSingleValueFlagged: `?` on a foreign function returning a
// single non-error value (dep.One() int) inside a Result[int, error] function must
// produce the existing question-callee-no-error diagnostic.
func TestForeignQuestionSingleValueFlagged(t *testing.T) {
	const importPath = "goal/internal/corpus/testdata/foreignq/dep"
	depDir, err := filepath.Abs(filepath.Join("testdata", "foreignq", "dep"))
	if err != nil {
		t.Fatal(err)
	}
	const consumerSrc = `package consumer

import "goal/internal/corpus/testdata/foreignq/dep"

func caller() Result[int, error] {
	x := dep.One()?
	return Result.Ok(x)
}
`
	perFile, _, err := sema.AnalyzePackageInDirWith([]string{consumerSrc}, filepath.Dir(depDir), resolverTo(importPath, depDir))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !hasQuestionCalleeNoError(perFile) {
		t.Fatalf("expected a question-callee-no-error diagnostic for `?` on foreign single-value dep.One(); got %+v", perFile)
	}
}

// TestForeignQuestionErrorCalleeClean: `?` on a foreign function returning
// (int, error) (dep.Get()) is well-formed and must produce no question diagnostic.
func TestForeignQuestionErrorCalleeClean(t *testing.T) {
	const importPath = "goal/internal/corpus/testdata/foreignq/dep"
	depDir, err := filepath.Abs(filepath.Join("testdata", "foreignq", "dep"))
	if err != nil {
		t.Fatal(err)
	}
	const consumerSrc = `package consumer

import "goal/internal/corpus/testdata/foreignq/dep"

func caller() Result[int, error] {
	x := dep.Get()?
	return Result.Ok(x)
}
`
	perFile, _, err := sema.AnalyzePackageInDirWith([]string{consumerSrc}, filepath.Dir(depDir), resolverTo(importPath, depDir))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if hasQuestionCalleeNoError(perFile) {
		t.Fatalf("expected NO question-callee-no-error diagnostic for `?` on foreign (int, error) dep.Get(); got %+v", perFile)
	}
}

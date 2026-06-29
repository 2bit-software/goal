package sema

import (
	"path/filepath"
	"testing"

	"goal/internal/ast"
	"goal/internal/parser"
)

// SEAM-CAP-3c: a sealed interface DEFINED in a sibling .goal package must be projected
// into a consumer's sealed-interface registry (info.Sealed + info.SealedImpls) by
// EnrichForeign's goal-source path, so a cross-package type-pattern `match` resolves and
// exhaustiveness-checks — exactly as a same-package sealed match does (CAP-3b). Before the
// fix, foreign enrichment projected enums only, so the consumer's match deferred with an
// `unresolved-match-sealed` Warning.

// sealedShapeResolver maps the fixture import path to the sibling-.goal package dir, so the
// goal-source enrichment path (goalForeignDecls) reads shape.goal straight from source.
func sealedShapeResolver(t *testing.T) DirResolver {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "sealedshape"))
	if err != nil {
		t.Fatal(err)
	}
	return func(importPath, fromDir string) (string, error) {
		if importPath != "example.com/shape" {
			t.Fatalf("unexpected import path %q", importPath)
		}
		return dir, nil
	}
}

// enrichSealedConsumer parses a consumer goal source that imports the sealed-shape fixture,
// resolves it, and runs EnrichForeign with the fixture resolver.
func enrichSealedConsumer(t *testing.T, src string) (*ast.File, *Info) {
	t.Helper()
	file, perr := parser.ParseFile(src)
	if perr != nil {
		t.Fatalf("parse goal source: %v", perr)
	}
	info := Resolve(file)
	if errs := EnrichForeign(info, file.Imports, ".", sealedShapeResolver(t)); len(errs) != 0 {
		t.Fatalf("EnrichForeign errors: %v", errs)
	}
	return file, info
}

// TestEnrichForeignProjectsSealedImplementors is the registry-level gate: after enrichment
// the consumer's info marks shape.Node sealed and registers its implementors qualified the
// way the consumer names them (`*shape.Lit`, `*shape.Neg`).
func TestEnrichForeignProjectsSealedImplementors(t *testing.T) {
	const src = `package consumer

import shape "example.com/shape"

func eval(n shape.Node) int {
	return match n {
		*shape.Lit(l) => l.Val
		*shape.Neg(g) => 0
	}
}
`
	_, info := enrichSealedConsumer(t, src)

	if !info.Sealed["shape.Node"] {
		t.Fatalf("shape.Node not marked sealed; Sealed=%v", info.Sealed)
	}
	impls := info.SealedImpls["shape.Node"]
	want := map[string]bool{"*shape.Lit": true, "*shape.Neg": true}
	if len(impls) != len(want) {
		t.Fatalf("SealedImpls[shape.Node] = %v, want the two qualified implementors", impls)
	}
	for _, got := range impls {
		if !want[got] {
			t.Errorf("unexpected implementor %q in %v", got, impls)
		}
	}
}

// TestCrossPackageSealedMatchExhaustive proves a complete cross-package sealed match is
// clean — no `unresolved-match-sealed` deferral and no `non-exhaustive-match` error.
func TestCrossPackageSealedMatchExhaustive(t *testing.T) {
	const src = `package consumer

import shape "example.com/shape"

func eval(n shape.Node) int {
	return match n {
		*shape.Lit(l) => l.Val
		*shape.Neg(g) => 0
	}
}
`
	file, info := enrichSealedConsumer(t, src)
	if d := CheckExhaustive(file, info); len(d) != 0 {
		t.Fatalf("exhaustive cross-package sealed match produced diagnostics: %+v", d)
	}
}

// TestCrossPackageSealedMatchNonExhaustiveIsError proves a cross-package sealed match that
// omits an implementor (and has no `_` rest) is a `non-exhaustive-match` Error naming the
// missing implementor — the registry resolves across the package boundary.
func TestCrossPackageSealedMatchNonExhaustiveIsError(t *testing.T) {
	const src = `package consumer

import shape "example.com/shape"

func eval(n shape.Node) int {
	return match n {
		*shape.Lit(l) => l.Val
	}
}
`
	file, info := enrichSealedConsumer(t, src)
	d := CheckExhaustive(file, info)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if SeverityLabel(d[0].Severity) != "error" {
		t.Errorf("want Error severity, got %v", d[0].Severity)
	}
	if d[0].Code != "non-exhaustive-match" {
		t.Errorf("want code non-exhaustive-match, got %q", d[0].Code)
	}
	if !contains(d[0].Message, "*shape.Neg") {
		t.Errorf("message should name the missing implementor `*shape.Neg`: %q", d[0].Message)
	}
}

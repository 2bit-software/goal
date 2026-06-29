package sema

import (
	"testing"

	"goal/internal/parser"
)

// SEAM-CAP-3d: a nested sealed-interface hierarchy. `sealed interface Expr` embeds
// `sealed interface Node`; concrete types declared `implements Expr` must register
// as implementors of BOTH Expr and Node (the embedding cascade), so exhaustiveness
// is enforced at both levels. `Comment` implements Node directly to keep the two
// levels' implementor sets distinct.
const nestedSealedHeader = `package p

sealed interface Node {}
sealed interface Expr { Node }

type Num struct implements Expr { Val int }
type Neg struct implements Expr { Inner Expr }
type Comment struct implements Node { Text string }
`

func hasImpl(impls []string, want string) bool {
	for _, s := range impls {
		if s == want {
			return true
		}
	}
	return false
}

// TestCascadeRegistersBothLevels proves the cascade folds an `implements Expr`
// type into Node's implementor set while leaving a direct-Node implementor in
// Node only.
func TestCascadeRegistersBothLevels(t *testing.T) {
	file, err := parser.ParseFile(nestedSealedHeader)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := Resolve(file)

	// Expr is the embedding level: only its direct implementors.
	for _, want := range []string{"*Num", "*Neg"} {
		if !hasImpl(info.SealedImpls["Expr"], want) {
			t.Errorf("SealedImpls[Expr] missing %q: %v", want, info.SealedImpls["Expr"])
		}
	}
	if hasImpl(info.SealedImpls["Expr"], "*Comment") {
		t.Errorf("SealedImpls[Expr] should NOT include the direct-Node *Comment: %v", info.SealedImpls["Expr"])
	}

	// Node is the embedded level: its direct implementor PLUS the cascaded ones.
	for _, want := range []string{"*Num", "*Neg", "*Comment"} {
		if !hasImpl(info.SealedImpls["Node"], want) {
			t.Errorf("SealedImpls[Node] missing cascaded %q: %v", want, info.SealedImpls["Node"])
		}
	}
}

// TestNestedMatchExhaustiveBothLevels: a match over the embedded interface (Node)
// and a match over the embedding interface (Expr) are each accepted when they
// cover their full implementor set.
func TestNestedMatchExhaustiveBothLevels(t *testing.T) {
	src := nestedSealedHeader + `
func evalExpr(e Expr) int {
	return match e {
		*Num(n) => n.Val
		*Neg(g) => -evalExpr(g.Inner)
	}
}

func evalNode(n Node) int {
	return match n {
		*Num(x) => x.Val
		*Neg(g) => -evalExpr(g.Inner)
		*Comment(c) => 0
	}
}
`
	if d := analyzeExhaustive(t, src); len(d) != 0 {
		t.Fatalf("exhaustive nested sealed matches produced diagnostics: %+v", d)
	}
}

// TestNestedMatchNonExhaustiveEmbeddingLevel: a match over Expr omitting *Neg errors.
func TestNestedMatchNonExhaustiveEmbeddingLevel(t *testing.T) {
	src := nestedSealedHeader + `
func evalExpr(e Expr) int {
	return match e {
		*Num(n) => n.Val
	}
}
`
	d := analyzeExhaustive(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if d[0].Code != "non-exhaustive-match" {
		t.Errorf("want code non-exhaustive-match, got %q", d[0].Code)
	}
	if !contains(d[0].Message, "*Neg") {
		t.Errorf("message should name the missing implementor `*Neg`: %q", d[0].Message)
	}
}

// TestNestedMatchNonExhaustiveEmbeddedLevel: a match over Node omitting the
// cascaded *Neg (and direct *Comment) errors — exhaustiveness sees the cascaded set.
func TestNestedMatchNonExhaustiveEmbeddedLevel(t *testing.T) {
	src := nestedSealedHeader + `
func evalNode(n Node) int {
	return match n {
		*Num(x) => x.Val
	}
}
`
	d := analyzeExhaustive(t, src)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(d), d)
	}
	if d[0].Code != "non-exhaustive-match" {
		t.Errorf("want code non-exhaustive-match, got %q", d[0].Code)
	}
	// The cascaded *Neg must appear among the missing implementors.
	if !contains(d[0].Message, "*Neg") {
		t.Errorf("message should name the cascaded missing implementor `*Neg`: %q", d[0].Message)
	}
}

package backendtest

import (
	"strings"
	"testing"

	"goal/internal/backend"
)

// TestExprTextPropagatesOptionHelper proves that an `Option.Some(...)` used as a
// derive-func override value — which the emitter renders through its exprText
// sub-emitter — still triggers the goalSome boxing-helper prelude (US-024). Before
// exprText propagated the sub-emitter's usedOptionHelper flag, the emitted Go called
// goalSome but never defined it, so the package failed to compile.
func TestExprTextPropagatesOptionHelper(t *testing.T) {
	src := `package conv

type Record struct {
	ID   string
	Name string
}

type View struct {
	ID   string
	Note Option[string]
}

derive func toView(src Record) View {
	return View{
		Note: Option.Some(src.Name),
		...derive(src),
	}
}
`
	got := mustTranspile(t, src)
	if !strings.Contains(got, "goalSome(src.Name)") {
		t.Errorf("expected the override to lower to a goalSome(...) call, got:\n%s", got)
	}
	if !strings.Contains(got, "func goalSome[") {
		t.Errorf("expected the goalSome boxing-helper prelude to be emitted, got:\n%s", got)
	}
}

// TestExprTextPropagatesError proves that an unsupported node inside a derive-func
// override surfaces as a located backend error instead of truncated Go (US-024).
// `src.Name?` parses to an *ast.UnwrapExpr, which the emitter's expression pass does
// not support; exprText must propagate the sub-emitter's error rather than dropping
// it and returning a partial rendering.
func TestExprTextPropagatesError(t *testing.T) {
	src := `package conv

type Src struct {
	Name string
}

type Dst struct {
	Name string
}

derive func toDst(src Src) Dst {
	return Dst{
		Name: src.Name?,
		...derive(src),
	}
}
`
	out, err := backend.Transpile(src)
	if err == nil {
		t.Fatalf("expected a transpile error for the unsupported override node, got none; output:\n%s", out.Go)
	}
	if !strings.Contains(err.Error(), "unsupported expression") {
		t.Errorf("error = %q, want it to name the unsupported expression", err.Error())
	}
}

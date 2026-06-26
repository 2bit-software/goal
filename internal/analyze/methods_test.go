package analyze

import (
	"testing"

	"goal/internal/scan"
)

// ResolveCallee resolves a `recv.Method()?` through the receiver parameter's declared type:
// an error-only method is arity 1, a (value, error) method arity 2.
func TestResolveCalleeInFileMethod(t *testing.T) {
	src := `package x

type Tx struct{}

func (tx *Tx) Commit() error { return nil }

func (tx *Tx) Load() (int, error) { return 0, nil }

func run(tx *Tx) Result[bool, error] {
	tx.Commit()?
	return Result.Ok(true)
}
`
	tables := Build(src)
	toks := scan.Lex(src)
	off := -1
	for _, tk := range toks {
		if tk.Text == "?" {
			off = tk.Start
			break
		}
	}
	if off < 0 {
		t.Fatal("no `?` in source")
	}

	commit, ok := ResolveCallee(tables, toks, off, "tx.Commit()")
	if !ok {
		t.Fatal("tx.Commit() not resolved")
	}
	if commit.Arity != 1 || !commit.EndsInError {
		t.Errorf("Commit = %+v, want arity 1 ending in error", commit)
	}

	load, ok := ResolveCallee(tables, toks, off, "tx.Load()")
	if !ok {
		t.Fatal("tx.Load() not resolved")
	}
	if load.Arity != 2 || !load.EndsInError {
		t.Errorf("Load = %+v, want arity 2 ending in error", load)
	}

	// A receiver whose type isn't a declared param/receiver stays unresolved.
	if _, ok := ResolveCallee(tables, toks, off, "other.Close()"); ok {
		t.Error("an undeclared receiver should not resolve")
	}
}

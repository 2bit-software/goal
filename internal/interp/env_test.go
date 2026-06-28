package interp

import (
	"errors"
	"testing"
)

func TestDefineAndLookupSameScope(t *testing.T) {
	env := NewEnv()
	env.Define("x", IntVal(42))

	got, err := env.Lookup("x")
	if err != nil {
		t.Fatalf("Lookup(x) returned error: %v", err)
	}
	if !got.Equal(IntVal(42)) {
		t.Fatalf("Lookup(x) = %s, want 42", got)
	}
}

func TestParentFallThrough(t *testing.T) {
	root := NewEnv()
	root.Define("outer", StrVal("hello"))
	child := root.NewChild()

	// A name defined only in the parent resolves from the child.
	got, err := child.Lookup("outer")
	if err != nil {
		t.Fatalf("Lookup(outer) from child returned error: %v", err)
	}
	if !got.Equal(StrVal("hello")) {
		t.Fatalf("Lookup(outer) = %s, want \"hello\"", got)
	}
}

func TestShadowing(t *testing.T) {
	root := NewEnv()
	root.Define("v", IntVal(1))
	child := root.NewChild()
	child.Define("v", IntVal(2))

	// Inner scope sees the shadowing value.
	inner, err := child.Lookup("v")
	if err != nil {
		t.Fatalf("child.Lookup(v) returned error: %v", err)
	}
	if !inner.Equal(IntVal(2)) {
		t.Fatalf("child.Lookup(v) = %s, want 2", inner)
	}

	// The outer binding is non-destructively preserved.
	outer, err := root.Lookup("v")
	if err != nil {
		t.Fatalf("root.Lookup(v) returned error: %v", err)
	}
	if !outer.Equal(IntVal(1)) {
		t.Fatalf("root.Lookup(v) = %s, want 1 (outer must be untouched)", outer)
	}
}

func TestLookupUndefinedReturnsNotFound(t *testing.T) {
	env := NewEnv().NewChild()

	got, err := env.Lookup("missing")
	if err == nil {
		t.Fatalf("Lookup(missing) returned no error, value %s", got)
	}
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("Lookup(missing) error = %T %v, want *NotFoundError", err, err)
	}
	if nf.Name != "missing" {
		t.Fatalf("NotFoundError.Name = %q, want \"missing\"", nf.Name)
	}
	if !got.Equal(Value{}) {
		t.Fatalf("Lookup(missing) value = %s, want zero Value", got)
	}
}

func TestDefineOverwriteSameScope(t *testing.T) {
	env := NewEnv()
	env.Define("x", IntVal(1))
	env.Define("x", IntVal(9))

	got, err := env.Lookup("x")
	if err != nil {
		t.Fatalf("Lookup(x) returned error: %v", err)
	}
	if !got.Equal(IntVal(9)) {
		t.Fatalf("Lookup(x) = %s, want 9 (re-Define must overwrite)", got)
	}
}

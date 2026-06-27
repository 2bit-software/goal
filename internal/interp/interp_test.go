package interp

// These tests prove the interpreter entry seam (US-004): a trivial goal program
// parsed by internal/parser and resolved by internal/sema runs through the
// interpreter with no error, demonstrating goscript consumes the SHARED AST +
// sema front-end rather than the Go backend's lowered output. A program with no
// `func main` is a loud, named refusal.

import (
	"errors"
	"strings"
	"testing"

	"goal/internal/parser"
	"goal/internal/sema"
)

func TestRunTrivialMain(t *testing.T) {
	const src = "package main\nfunc main() {}\n"

	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := sema.Resolve(file)

	ip := New(file, info)
	if err := ip.Run(); err != nil {
		t.Fatalf("Run trivial main: unexpected error: %v", err)
	}
}

func TestRunMissingMainErrors(t *testing.T) {
	const src = "package main\nfunc other() {}\n"

	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := sema.Resolve(file)

	err = New(file, info).Run()
	if err == nil {
		t.Fatalf("Run without main: want error, got nil")
	}
	if !errors.Is(err, ErrNoMain) {
		t.Fatalf("Run without main: want ErrNoMain, got %v", err)
	}
	if !strings.Contains(err.Error(), "main") {
		t.Fatalf("Run without main: error %q does not mention main", err.Error())
	}
}

// TestConstructFromSharedFrontEnd documents the seam: New takes the parsed
// *ast.File and *sema.Info — the same artifacts the Go back-end receives — and
// nothing Go-lowered.
func TestConstructFromSharedFrontEnd(t *testing.T) {
	const src = "package main\nfunc main() {}\n"

	file, err := parser.ParseFile(src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	info := sema.Resolve(file)

	ip := New(file, info)
	if ip == nil {
		t.Fatal("New returned nil")
	}
	if ip.file != file {
		t.Fatal("interpreter did not retain the parsed *ast.File")
	}
	if ip.info != info {
		t.Fatal("interpreter did not retain the *sema.Info")
	}
}

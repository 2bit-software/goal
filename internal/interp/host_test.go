package interp

// These tests prove US-011 "Add host-function bridge for stdlib": the
// interpreter resolves a curated set of imported stdlib calls (fmt.Sprintf,
// fmt.Sprint, fmt.Println, fmt.Errorf, errors.New) to native Go, and an
// unresolved imported call is a LOCATED, NAMED refusal rather than a silent
// nil. Programs are parsed + sema-resolved through the shared front-end and run
// through the interpreter (the standard direct-evalExpr testing seam).

import (
	"strings"
	"testing"
)

const hostProgram = `package main

import (
	"fmt"
	"errors"
	"strings"
)

func greet() string {
	return fmt.Sprintf("%s-%d", "x", 7)
}

func joined() string {
	return fmt.Sprint("a", "b", 1)
}

func plain() error {
	return errors.New("boom")
}

func wrapped() error {
	return fmt.Errorf("ctx: %w", errors.New("inner"))
}

func shout() string {
	return strings.ToUpper("hi")
}

func main() {}
`

func TestHostSprintf(t *testing.T) {
	ip := newInterp(t, hostProgram)
	got := evalFn(t, ip, "greet")
	if got.Kind != KindString || got.Str != "x-7" {
		t.Fatalf("fmt.Sprintf: want string \"x-7\", got %s %q", got.Kind, got.Str)
	}
}

func TestHostSprint(t *testing.T) {
	ip := newInterp(t, hostProgram)
	got := evalFn(t, ip, "joined")
	// fmt.Sprint inserts spaces only between non-string operands; "a","b" abut.
	if got.Kind != KindString || got.Str != "ab1" {
		t.Fatalf("fmt.Sprint: want string \"ab1\", got %s %q", got.Kind, got.Str)
	}
}

func TestHostErrorsNew(t *testing.T) {
	ip := newInterp(t, hostProgram)
	got := evalFn(t, ip, "plain")
	if !isErrorValue(got) {
		t.Fatalf("errors.New: want error value, got %s", got.Kind)
	}
	if msg := got.Struct.Fields["message"]; msg.Str != "boom" {
		t.Fatalf("errors.New: want message \"boom\", got %q", msg.Str)
	}
}

func TestHostErrorfWrapsError(t *testing.T) {
	ip := newInterp(t, hostProgram)
	got := evalFn(t, ip, "wrapped")
	if !isErrorValue(got) {
		t.Fatalf("fmt.Errorf: want error value, got %s", got.Kind)
	}
	msg := got.Struct.Fields["message"].Str
	// The %w-wrapped inner error must render through goArg's error bridge.
	if msg != "ctx: inner" {
		t.Fatalf("fmt.Errorf: want message \"ctx: inner\", got %q", msg)
	}
}

func TestUnregisteredImportedCallNamedError(t *testing.T) {
	ip := newInterp(t, hostProgram)
	_, err := ip.evalExpr(call("shout"), ip.root)
	if err == nil {
		t.Fatal("unregistered imported call: want error, got nil")
	}
	// The refusal must NAME the missing pkg.Symbol ...
	if !strings.Contains(err.Error(), "strings.ToUpper") {
		t.Fatalf("unregistered call error %q does not name strings.ToUpper", err.Error())
	}
	// ... and be LOCATED (a line:col position renders with a colon).
	if !strings.Contains(err.Error(), ":") {
		t.Fatalf("unregistered call error %q is not located", err.Error())
	}
}

// TestShadowedPackageNameNotRoutedToHost proves the host bridge yields to a
// local binding that shadows a package name: a selector call on a shadowing
// local routes to ordinary method dispatch, not the host registry.
func TestShadowedPackageNameNotRoutedToHost(t *testing.T) {
	const src = `package main

import "fmt"

type Box struct {
	v string
}

func (b Box) Sprintf() string {
	return b.v
}

func shadow() string {
	fmt := Box{v: "local"}
	return fmt.Sprintf()
}

func main() {}
`
	ip := newInterp(t, src)
	got := evalFn(t, ip, "shadow")
	if got.Kind != KindString || got.Str != "local" {
		t.Fatalf("shadowed package name: want method result \"local\", got %s %q", got.Kind, got.Str)
	}
}

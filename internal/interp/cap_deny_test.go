package interp

// These tests prove US-024 "Enforce capability denial": running the goscript
// interpreter under a cap.CapabilitySet that does NOT grant a capability causes
// the corresponding host effect to fail with a located, named CapabilityError
// instead of performing it — and, for the standard-output effect, nothing
// reaches the sink. The granted path is unchanged (regression).

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"goal/internal/cap"
	"goal/internal/parser"
	"goal/internal/sema"
	"goal/internal/token"
)

// TestPrintlnUnderDeniedStdoutIsRefused runs the printing program under a set
// that denies every capability and asserts the run is refused with a located,
// named CapabilityError naming Stdout, and that nothing is written to the sink.
func TestPrintlnUnderDeniedStdoutIsRefused(t *testing.T) {
	file, err := parser.ParseFile(printlnProgram)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var buf bytes.Buffer
	ip := New(file, sema.Resolve(file), WithCapabilities(cap.DenyAll()), WithStdout(&buf))
	runErr := ip.Run()
	if runErr == nil {
		t.Fatalf("Run under denied Stdout returned nil; want a capability-denied error")
	}
	var capErr CapabilityError
	if !errors.As(runErr, &capErr) {
		t.Fatalf("Run error = %v (%T); want CapabilityError", runErr, runErr)
	}
	if capErr.Cap != cap.Stdout {
		t.Fatalf("CapabilityError.Cap = %s; want Stdout", capErr.Cap)
	}
	// Named: the message identifies the denied capability.
	if !strings.Contains(runErr.Error(), "Stdout") {
		t.Fatalf("error %q does not name the Stdout capability", runErr.Error())
	}
	// Located: the message carries a non-empty source position.
	if pos := capErr.Pos.String(); pos == "" || !strings.Contains(runErr.Error(), pos) {
		t.Fatalf("error %q is not located (pos %q)", runErr.Error(), pos)
	}
	// Nothing written.
	if buf.Len() != 0 {
		t.Fatalf("denied Stdout wrote %q; want nothing written", buf.String())
	}
}

// TestEmitStdoutDeniedReturnsLocatedNamedErrorAndSkipsWrite drives the gate
// directly: under a denied set emitStdout must NOT invoke the write func and
// must return a located, named CapabilityError.
func TestEmitStdoutDeniedReturnsLocatedNamedErrorAndSkipsWrite(t *testing.T) {
	file, err := parser.ParseFile(printlnProgram)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var buf bytes.Buffer
	ip := New(file, sema.Resolve(file), WithCapabilities(cap.DenyAll()), WithStdout(&buf))
	wrote := false
	pos := token.Pos{Line: 7, Col: 2}
	emitErr := ip.emitStdout(pos, func(w io.Writer) error {
		wrote = true
		_, e := w.Write([]byte("x"))
		return e
	})
	if wrote {
		t.Fatalf("emitStdout under denied Stdout invoked the write func; want it skipped")
	}
	var capErr CapabilityError
	if !errors.As(emitErr, &capErr) {
		t.Fatalf("emitStdout error = %v (%T); want CapabilityError", emitErr, emitErr)
	}
	if capErr.Cap != cap.Stdout {
		t.Fatalf("CapabilityError.Cap = %s; want Stdout", capErr.Cap)
	}
	if !strings.Contains(emitErr.Error(), "7:2") {
		t.Fatalf("emitStdout error %q is not located at the supplied position", emitErr.Error())
	}
}

// TestPrintlnUnderGrantedStdoutStillPrints is the FR-5 regression: under full
// authority the program still prints and Run returns nil.
func TestPrintlnUnderGrantedStdoutStillPrints(t *testing.T) {
	file, err := parser.ParseFile(printlnProgram)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var buf bytes.Buffer
	ip := New(file, sema.Resolve(file), WithCapabilities(cap.GrantAll()), WithStdout(&buf))
	if err := ip.Run(); err != nil {
		t.Fatalf("Run under GrantAll: %v", err)
	}
	if got, want := buf.String(), "hello 42\n"; got != want {
		t.Fatalf("granted Stdout output = %q, want %q", got, want)
	}
}

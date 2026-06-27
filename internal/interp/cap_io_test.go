package interp

// These tests prove US-023 "Route runtime IO through cap": the goscript
// interpreter's host effects flow through a cap.CapabilitySet, the default Run
// path grants every capability (cap.GrantAll), and the standard-output effect
// targets a configurable sink so a host — or a test — can capture what a
// program prints. Denial-to-error is the next story (US-024); here the routing
// seam exists and the default grant performs the effect.

import (
	"bytes"
	"io"
	"testing"

	"goal/internal/cap"
	"goal/internal/parser"
	"goal/internal/sema"
)

const printlnProgram = `package main

import "fmt"

func main() {
	fmt.Println("hello", 42)
}
`

// TestPrintlnUnderGrantAllWritesToSink runs a printing program under the default
// full-grant authority with a captured sink and asserts the produced output.
func TestPrintlnUnderGrantAllWritesToSink(t *testing.T) {
	file, err := parser.ParseFile(printlnProgram)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var buf bytes.Buffer
	ip := New(file, sema.Resolve(file), WithStdout(&buf))
	if err := ip.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, want := buf.String(), "hello 42\n"; got != want {
		t.Fatalf("captured stdout = %q, want %q", got, want)
	}
}

// TestNewDefaultsGrantAllCapabilities proves the default run path grants every
// defined capability (FR-2): a freshly constructed interpreter holds Stdout,
// Stdin, FileRead, FileWrite, Net, Concurrency, Time, and Env.
func TestNewDefaultsGrantAllCapabilities(t *testing.T) {
	ip := newInterp(t, printlnProgram)
	all := []cap.Capability{
		cap.Stdout, cap.Stdin, cap.FileRead, cap.FileWrite,
		cap.Net, cap.Concurrency, cap.Time, cap.Env,
	}
	for _, c := range all {
		if !ip.caps.Has(c) {
			t.Fatalf("default interpreter does not grant %s; want GrantAll", c)
		}
	}
}

// TestEmitStdoutRoutesThroughConfiguredSink guards the capability-mediated gate
// independent of any shim: under a granted Stdout the write lands in the
// configured sink.
func TestEmitStdoutRoutesThroughConfiguredSink(t *testing.T) {
	file, err := parser.ParseFile(printlnProgram)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var buf bytes.Buffer
	ip := New(file, sema.Resolve(file), WithStdout(&buf))
	err = ip.emitStdout(func(w io.Writer) error {
		_, e := w.Write([]byte("sentinel"))
		return e
	})
	if err != nil {
		t.Fatalf("emitStdout: %v", err)
	}
	if got := buf.String(); got != "sentinel" {
		t.Fatalf("emitStdout sink = %q, want %q", got, "sentinel")
	}
}

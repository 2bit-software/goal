package sema

import "testing"

// TestResolveFuncSignatureModes asserts the error-propagation Mode resolved for
// each function signature in the representative source.
//
// This lives in its own internal-only test file (NOT in the self-host port-gate
// test list at internal/selfhost/port_test.go) because it references the Go-iota
// Mode constants (ModeNone/ModeResult/ModeResultClosed/ModeOption) directly. In the
// self-hosted tree (SEAM-003) Mode is a goal `enum`, whose transpiled Go has no
// such bare constants — so these assertions cannot be compiled against the enum
// form. The bootstrap reference compiler's own sema (this package) stays Go-iota,
// and keeping this coverage here lets the shared resolve_test.go remain Mode-symbol-
// free so it compiles against both representations. Reuses mustResolve/repr from
// resolve_test.go (same package).
func TestResolveFuncSignatureModes(t *testing.T) {
	s := mustResolve(t, repr)

	if got := s.FuncSignatures["area"]; got.Mode != ModeNone {
		t.Errorf("area mode = %v, want ModeNone", got.Mode)
	}
	if got := s.FuncSignatures["find"]; got.Mode != ModeResult {
		t.Errorf("find mode = %v, want ModeResult", got.Mode)
	}
	if got := s.FuncSignatures["lookup"]; got.Mode != ModeOption {
		t.Errorf("lookup mode = %v, want ModeOption", got.Mode)
	}
	if got := s.FuncSignatures["closed"]; got.Mode != ModeResultClosed {
		t.Errorf("closed mode = %v, want ModeResultClosed", got.Mode)
	}
}

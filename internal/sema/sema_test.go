package sema

import "testing"

// TestNewReturnsInfo pins the placeholder seam: New yields a usable, non-nil Info
// that the Backend.Emit signature (Emit(*ast.File, *sema.Info)) can be handed.
func TestNewReturnsInfo(t *testing.T) {
	if New() == nil {
		t.Fatal("sema.New() = nil, want non-nil *Info")
	}
}

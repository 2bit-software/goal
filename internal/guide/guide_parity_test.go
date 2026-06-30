package guide_test

// US-009 parity oracle. The guide surface was ported to goal source under
// internal/compiler/guide; its committed generated Go lives in the real module
// alongside every dependency (the root goal package's embedded docs, byexample,
// and the ported backend/sema), so parity is proven by rendering the guide from
// BOTH the legacy hand-written package and the goal-sourced one and asserting
// byte-identical output across a fixture set (the full document and every
// section). This is the AC-2 check; it can run here — unlike the throwaway
// `module goal` selfhost harness, which cannot resolve guide's root-goal and
// byexample imports.

import (
	"bytes"
	"testing"

	ported "goal/internal/compiler/guide"
	legacy "goal/internal/guide"
)

// fixtureCommands mirrors the same command set into each package's distinct
// Command type, so the toolchain section renders from identical inputs on both
// sides.
func legacyCommands() []legacy.Command {
	return []legacy.Command{
		{Name: "build", Summary: "transpile goal to Go", Usage: "goal build [--emit dir] <path>", Flags: []legacy.Flag{{Name: "--emit", Summary: "write generated Go to dir"}}},
		{Name: "check", Summary: "report located correctness feedback", Usage: "goal check <path>"},
		{Name: "run", Summary: "transpile and run", Usage: "goal run <path>"},
		{Name: "ai", Summary: "print this guide", Usage: "goal ai [section]"},
	}
}

func portedCommands() []ported.Command {
	return []ported.Command{
		{Name: "build", Summary: "transpile goal to Go", Usage: "goal build [--emit dir] <path>", Flags: []ported.Flag{{Name: "--emit", Summary: "write generated Go to dir"}}},
		{Name: "check", Summary: "report located correctness feedback", Usage: "goal check <path>"},
		{Name: "run", Summary: "transpile and run", Usage: "goal run <path>"},
		{Name: "ai", Summary: "print this guide", Usage: "goal ai [section]"},
	}
}

// TestGuideParitySectionKeys holds the two packages to the same section set and
// order — the precondition for the per-section parity sweep below.
func TestGuideParitySectionKeys(t *testing.T) {
	want := legacy.SectionKeys()
	got := ported.SectionKeys()
	if len(want) != len(got) {
		t.Fatalf("section count: legacy %d, ported %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("section %d: legacy %q, ported %q", i, want[i], got[i])
		}
	}
}

// TestGuideParityFullDocument renders the entire guide from both packages and
// asserts byte-identical output — the headline AC-2 fixture.
func TestGuideParityFullDocument(t *testing.T) {
	var lb, pb bytes.Buffer
	if err := legacy.Render(&lb, "", legacyCommands()); err != nil {
		t.Fatalf("legacy Render: %v", err)
	}
	if err := ported.Render(&pb, "", portedCommands()); err != nil {
		t.Fatalf("ported Render: %v", err)
	}
	if lb.String() != pb.String() {
		t.Errorf("full-document guide differs between legacy and goal-built guide\n--- legacy len=%d ---\n--- ported len=%d ---", lb.Len(), pb.Len())
	}
}

// TestGuideParityPerSection renders each section individually from both packages
// and asserts byte-identical output, widening the fixture set beyond the full
// document so a divergence is localized to its section.
func TestGuideParityPerSection(t *testing.T) {
	for _, key := range legacy.SectionKeys() {
		var lb, pb bytes.Buffer
		if err := legacy.Render(&lb, key, legacyCommands()); err != nil {
			t.Fatalf("legacy Render(%q): %v", key, err)
		}
		if err := ported.Render(&pb, key, portedCommands()); err != nil {
			t.Fatalf("ported Render(%q): %v", key, err)
		}
		if lb.String() != pb.String() {
			t.Errorf("section %q differs between legacy and goal-built guide (legacy len=%d, ported len=%d)", key, lb.Len(), pb.Len())
		}
	}
}

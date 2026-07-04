package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestAICommandEmitsGuide checks `goal ai` writes a non-empty Markdown guide to stdout
// and nothing to stderr, via both the subcommand and the `--ai` alias.
func TestAICommandEmitsGuide(t *testing.T) {
	for _, args := range [][]string{{"ai"}, {"--ai"}} {
		var out, errOut bytes.Buffer
		if err := run(args, &out, &errOut); err != nil {
			t.Fatalf("run %v: %v\nstderr: %s", args, err, errOut.String())
		}
		if !strings.HasPrefix(out.String(), "# goal — AI bootstrap guide") {
			t.Errorf("run %v: output does not start with the guide title:\n%.80s", args, out.String())
		}
		if errOut.Len() != 0 {
			t.Errorf("run %v: wrote to stderr: %s", args, errOut.String())
		}
	}
}

// TestAISectionSelectsOneSection checks `goal ai <section>` prints just that section,
// and that an unknown section name is a useful error.
func TestAISectionSelectsOneSection(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"ai", "toolchain"}, &out, &errOut); err != nil {
		t.Fatalf("run ai toolchain: %v", err)
	}
	if !strings.HasPrefix(out.String(), "## The toolchain") {
		t.Errorf("section output should start with the toolchain heading, got:\n%.80s", out.String())
	}
	if strings.Contains(out.String(), "## The features") {
		t.Error("section output leaked another section")
	}

	out.Reset()
	errOut.Reset()
	if err := run([]string{"ai", "bogus"}, &out, &errOut); err == nil {
		t.Fatal("expected an error for an unknown section")
	}
}

// TestToolchainSectionListsEveryCommand holds the rendered toolchain section to the
// real command registry, so a new subcommand cannot ship undocumented.
func TestToolchainSectionListsEveryCommand(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"ai", "toolchain"}, &out, io.Discard); err != nil {
		t.Fatalf("run ai toolchain: %v", err)
	}
	for _, c := range guideCommands {
		if !strings.Contains(out.String(), c.Usage) {
			t.Errorf("toolchain section omits command %q (usage %q)", c.Name, c.Usage)
		}
	}
}

// TestAIBaseIsSlim holds the base `goal ai` to its two-tier contract: it teaches the four
// core features in full, advertises `goal fix` and the on-demand tiers, and omits the bulk
// reference material that now lives behind `goal category` / `goal ai <section>`.
func TestAIBaseIsSlim(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"ai"}, &out, io.Discard); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	got := out.String()

	if n := strings.Count(got, "\n") + 1; n >= 500 {
		t.Errorf("slim base is %d lines; want under 500", n)
	}

	// The four core features are taught in full (heading + live result block).
	for _, want := range []string{"### Enums", "### Match", "### Doctests", "### implements", "_transpiles to:_"} {
		if !strings.Contains(got, want) {
			t.Errorf("slim base omits core-feature marker %q", want)
		}
	}

	// The fix pointer names the in-place invocation, and the tiers pointer names both
	// on-demand surfaces.
	if !strings.Contains(got, "goal fix") || !strings.Contains(got, "-inplace") {
		t.Error("slim base omits the `goal fix -inplace` pointer")
	}
	if !strings.Contains(got, "goal category") || !strings.Contains(got, "goal ai <section>") {
		t.Error("slim base omits the tiers pointer naming `goal category` and `goal ai <section>`")
	}

	// The non-core features and the relocated sections must not appear in the base.
	for _, absent := range []string{
		"#### Option", "#### derive-convert", "## Checker diagnostics",
		"## A complete starter program", "## Locked conventions", "## Deeper references",
	} {
		if strings.Contains(got, absent) {
			t.Errorf("slim base leaks relocated content %q", absent)
		}
	}
}

// TestAIFeaturesUnchanged pins `goal ai features` to a golden captured before the guide
// was re-tiered, so the shared per-feature renderer keeps producing the full features
// section byte-for-byte. Regenerate with: go run ./cmd/goal ai features > cmd/goal/testdata/ai-features.golden
func TestAIFeaturesUnchanged(t *testing.T) {
	golden, err := os.ReadFile("testdata/ai-features.golden")
	if err != nil {
		t.Fatalf("read features golden: %v", err)
	}
	var out bytes.Buffer
	if err := run([]string{"ai", "features"}, &out, io.Discard); err != nil {
		t.Fatalf("run ai features: %v", err)
	}
	if out.String() != string(golden) {
		t.Errorf("`goal ai features` drifted from testdata/ai-features.golden; if intended, regenerate with:\n" +
			"    go run ./cmd/goal ai features > cmd/goal/testdata/ai-features.golden")
	}
}

// TestBootstrapGoldenMatches asserts the committed AI-KNOWLEDGE-BOOTSTRAP.md equals what
// `goal ai` produces now. If this fails, the language changed and the committed copy is
// stale — regenerate it with: go run ./cmd/goal ai > AI-KNOWLEDGE-BOOTSTRAP.md
func TestBootstrapGoldenMatches(t *testing.T) {
	committed, err := os.ReadFile("../../AI-KNOWLEDGE-BOOTSTRAP.md")
	if err != nil {
		t.Fatalf("read committed bootstrap: %v", err)
	}
	var out bytes.Buffer
	if err := run([]string{"ai"}, &out, io.Discard); err != nil {
		t.Fatalf("run ai: %v", err)
	}
	if out.String() != string(committed) {
		t.Errorf("AI-KNOWLEDGE-BOOTSTRAP.md is stale; regenerate with:\n" +
			"    go run ./cmd/goal ai > AI-KNOWLEDGE-BOOTSTRAP.md")
	}
}

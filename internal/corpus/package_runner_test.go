package corpus

import (
	"testing"

	"goal/internal/backend"
)

// TestPackageRunner runs every Mode=package case in the committed manifest
// through the current package driver via the [PackageTranspiler] interface and
// asserts each transpiles, emits valid Go, and builds as one module (with its
// declared foreign imports wired in). It fails loudly if the manifest yields no
// package-mode cases, so a mis-generated manifest cannot masquerade as green.
//
// It spawns the go toolchain per case, so it is skipped under -short; the full
// `go test ./... -count=1` gate exercises it.
func TestPackageRunner(t *testing.T) {
	if testing.Short() {
		t.Skip("package tier spawns the go toolchain per case")
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", manifestPath, err)
	}

	pt := PackageTranspilerFunc(backend.TranspilePackage)
	ran := 0
	for _, c := range m.Cases {
		if c.Mode != ModePackage {
			continue
		}
		ran++
		c := c
		t.Run(c.ID, func(t *testing.T) {
			if err := RunPackage(repoRoot, c, pt); err != nil {
				t.Error(err)
			}
		})
	}

	if ran == 0 {
		t.Fatalf("manifest %q contains no package-mode cases", manifestPath)
	}
}

# Plan Buildability Audit — US-005

## Buildability check

- Dependency order valid: source file -> helper -> test. No forward references.
- Interface contract for BuildAndTest is concrete (signature given) and mirrors
  the existing BuildTranspiled/writePackage patterns in selfhost.go.
- File paths verified: selfhost/ and internal/selfhost/ exist;
  internal/token/token_test.go exists and is same-package (white-box).
- Integration point specifies exact reader (project.Discover), exact harness
  calls, and the relative paths from the test cwd (internal/selfhost).
- Offline-safe: token has no imports; `go test ./internal/token` in a `module
  goal` temp module needs no network.

No CRITICAL or MAJOR findings.

## Assumptions

- `go` toolchain is on PATH in the test environment (already required by the
  existing US-002 smoke gate, which shells out to `go build`).
- A throwaway `module goal` temp module resolves in-module imports correctly
  (already proven by the existing BuildTranspiled).

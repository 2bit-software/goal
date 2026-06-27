# Verify — Quality

Implementation: internal/interp/interp.go, internal/interp/host.go,
internal/interp/cap_io_test.go

## Checks

- Error handling: `emitStdout` returns the writer's error; `evalHostCall`
  propagates it. The nil-sink guard falls back to `os.Stdout`.
- Tests assert what they claim: the print test asserts the exact captured bytes
  (not just non-empty); the default-grant test iterates every capability; the
  gate test exercises `emitStdout` directly with a sentinel.
- No contradiction with spec: the default grant performs the effect; routing is
  through the capability set; the sink is configurable.
- Dependency hygiene: only `io`, `os`, and the dependency-free `goal/internal/cap`
  were added to interp.go; host.go swapped `os` for `io`. US-022 gate still green.
- Style: stdlib `testing` only, no testify; doc comments updated on the new
  fields, options, gate, and the host.go header.

## Findings
- No CRITICAL/MAJOR. MINOR: time/env effect sites do not exist yet, so the seam
  is proven only for stdout today — consistent with Out of Scope.

## Assumptions
- Variadic functional options are the construction mechanism for the optional
  sink/capabilities; existing 2-arg `New` callers are unaffected.

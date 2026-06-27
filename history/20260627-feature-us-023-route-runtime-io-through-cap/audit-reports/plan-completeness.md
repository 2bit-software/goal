# Plan Audit — Coverage

Plan: implementation-plan.md  Spec: business-spec.md

## Coverage map

- FR-1 (effects capability-mediated) → `emitStdout` gate + `fmt.Println` routed
  through it (interp.go, host.go). Covered.
- FR-2 (default authority full) → `New` defaults `caps = cap.GrantAll()`;
  `TestNewDefaultsGrantAllCapabilities`. Covered.
- FR-3 (configurable sink) → `stdout io.Writer` + `WithStdout`;
  `TestPrintlnUnderGrantAllWritesToSink`. Covered.
- AC "print under GrantAll captured through sink" → the print test. Covered.
- AC "existing behavior unchanged" → variadic options keep 2-arg `New`; full
  suite re-run. Covered.

## Findings

- No CRITICAL/MAJOR. No scope creep: every plan element traces to a requirement.
- MINOR: only `fmt.Println` exists as an effect today; time/env are seam-only and
  correctly deferred — matches Out of Scope.

## Assumptions

- The effectful shim is intercepted in `evalHostCall` rather than kept in the
  pure registry, because it needs the interpreter's sink + caps.

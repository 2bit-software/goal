# Audit: AI-Consumer Readiness — US-021

## Findings

No CRITICAL findings. No MAJOR findings.

- The conversion strategy is fully specified by reference to the established Go
  backend lowering (genConversion/resolveField) over the same sema facts
  (Structs, FromRegistry). An implementer has an exact, known-good reference.
- Data formats are defined: sema.Field{Name, Type}, ConvEntry{Name, Fallible},
  and the runtime Value/StructVal model.
- Acceptance criteria are concrete enough to write assertions: read the produced
  struct's fields and compare identity, bridged, and nested values.

- MINOR: The error-message text for unsourced/unconvertible fields is not pinned
  to an exact string; the spec only requires it be descriptive and name the
  derive + field. Acceptable — tests assert on substrings.

## Assumptions

- Derive decls are intercepted before the generic call path (they are not ordinary
  callables); a dedicated registry map holds them.
- Type-string splitters needed by the runtime conversion are mirrored locally in
  internal/interp to preserve the US-022 dependency envelope (no internal/backend
  import).

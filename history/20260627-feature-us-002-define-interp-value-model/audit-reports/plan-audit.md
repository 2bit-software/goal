# Plan Audit

## Findings

- Every FR traces to `internal/interp/value.go`: FR-1 (primitives) → Int/Float/
  Str/Bool/Nil kinds + constructors; FR-2 (composites) → Struct/Slice/Map/Func;
  FR-3 (universal tagged union) → Variant{TypeID,Tag,Fields} + VariantVal; FR-4
  (field read-back) → Field(name); FR-5 (equality) → Equal; FR-6 (rendering) →
  String. No requirement is unmapped.
- Dependency order is trivially valid (single leaf file + its test). No cycles.
- File paths verified: `internal/interp` does not exist yet (no conflict);
  sibling packages internal/cap, internal/sema confirm placement + style.
- No external dependencies introduced; tests use stdlib testing only.

- MINOR: v1 Map is string-keyed. Real goal maps allow non-string keys; deferred
  to a later eval story. Acceptable for this data-model story (the AC only needs
  "make of a map" round-trips, exercised later in US-009/US-010).
- MINOR: FuncVal is a name-only carrier; binding/calling is US-004+. The spec
  explicitly scopes callable wiring out, so this is intended.

No CRITICAL or MAJOR findings. The plan is implementable as written.

## Assumptions

- TypeID and Tag are strings; Fields and Map entries are string-keyed.
- A single int64/float64 represents all int/float widths in v1.
- Struct/Map/Func use small wrapper structs for distinct identity.
- Function values are name-only carriers in this story.

# Audit — Completeness (US-050)

The spec extends a well-established in-repo pattern (US-048 definition / US-049
hover), so the requirement surface is fully bounded by precedent. No CRITICAL or
MAJOR gaps.

## Findings

- **MINOR** — Rename new-name validation. Spec FR-3 returns an edit; AC adds that
  an invalid new name yields null. Resolution: validate the new name is a legal
  Go-style identifier (letter/`_` then letters/digits/`_`); reject empty/illegal
  with a null response. Captured in technical-requirements-research.md.
- **MINOR** — Cursor on the declaration name. Should rename/references work when
  the cursor sits on the declaration itself (not just a reference)? Resolution:
  yes — seed declaration-name occurrences (mirrors hover.go), so the declaration
  is a valid cursor target and is always part of the occurrence set.
- **MINOR** — Method calls `x.Foo()`. The existing definition graph keys a
  method call's `Sel` by name into the func index (no receiver typing). Rename
  inherits that single-document, name-keyed behavior; acceptable and consistent
  with US-048 scope.

## Edge cases covered

- Unknown URI, no symbol under cursor, unparseable source -> null (best-effort).
- Invalid rename target -> null.
- Variant tag shared by two enums -> kept distinct (keyed under enum).
- `includeDeclaration` false/true both exercised.

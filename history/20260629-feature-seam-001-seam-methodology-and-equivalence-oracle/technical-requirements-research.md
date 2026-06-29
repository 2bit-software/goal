# Technical Requirements / Research — SEAM-001

## Existing facts (from the repo)

- Verify gates (`prd.json` verifyCommands): `task check`, `task build`,
  `task fixpoint`. Taskfile targets confirmed via `task --list`.
- `task fixpoint` proves goal-c-1 and goal-c-2 emit byte-identical Go for the
  compiler's OWN source (stage1==stage2 self-consistency), NOT output==before.
  It therefore survives an emitted-Go change introduced by a seam edit.
- Golden regeneration mechanisms already in the tree:
  - Corpus exact-tier goldens (`.go.expected`): `internal/corpus/update_goldens_test.go`
    exposes `-update-goldens`; `TestUpdateGoldens` rewrites every exact-tier golden
    from the AST backend when the flag is set (no-op/skip otherwise).
  - Parser AST snapshots (`internal/parser/testdata/snapshots/`):
    `internal/parser/snapshot_test.go` exposes `-update-snapshots`.
  - go/ast-mirror unit tests live in `internal/ast` (ast_test.go etc.) — these
    are the byte-for-byte mirror tests the US-003 oracle pins.

## Approach

Append a new "## Seam methodology — equivalence oracle for cross-package idiom
changes" section to DECISIONS.md, consistent in tone/structure with the
existing US-005..US-013 audit sections (blockquote intro + bulleted decisions).
The section is referenced by the SEAM PRD description and SEAM-002..006 notes.

No code changes; the three verify gates stay green by construction.

# Implementation Plan — SEAM-001

## File Inventory

### New Files
None.

### Modified Files
| File | Change |
|------|--------|
| `DECISIONS.md` | Append a new "## Seam methodology — equivalence oracle for cross-package idiom changes" section after the US-013 section (current EOF, line ~2710). |
| `prd.json` | (done at finalize) set SEAM-001 `passes: true`. |
| `progress.txt` | (done at finalize) append a SEAM-001 progress entry. |

## Section content outline (DECISIONS.md)

Mirror the existing audit-section house style: a `>` blockquote intro, then
bulleted decision/procedure entries.

1. Blockquote intro: SEAM PRD; this section defines the relaxed gate for the
   seam stories (SEAM-002..006).
2. **The two gates contrasted** — per-package (US-005..US-013) required
   byte-identical emitted Go + fixed oracle signatures; seam stories ALLOW
   emitted-Go change. (FR-1)
3. **The crux** — `task fixpoint` proves goal-c-1 == goal-c-2 (stage1==stage2
   self-consistency), NOT output==before; it survives an emitted-Go change.
   (FR-1)
4. **The three equivalence proofs** for a seam edit: (a) fixpoint
   self-consistency, (b) corpus behavioral tier (compiled programs behave
   identically), (c) deliberately regenerated goldens reviewed for behavior
   preservation. (FR-1)
5. **Test classification table/list** — EXPECTED to change: ported go/ast-mirror
   unit tests (internal/ast), golden transpile-shape fixtures
   (.go.expected, parser snapshots). MUST stay byte-green: task fixpoint, corpus
   behavioral/interp(=behavioral/check) tiers, full task check after golden
   regen. (FR-2)
6. **Regenerate-and-review procedure** — concrete commands:
   - corpus exact goldens: `go test ./internal/corpus -run TestUpdateGoldens
     -update-goldens`
   - parser AST snapshots: `go test ./internal/parser -update-snapshots`
   - Reviewer checklist: diff is gofmt-stable; change is behavior-preserving;
     `task fixpoint` still byte-identical; corpus behavioral still green;
     never blindly accept a golden diff. (FR-3)

## Verification (maps to acceptance criteria)

- Read DECISIONS.md to confirm the section + its three required parts exist.
- `task check`, `task build`, `task fixpoint` all green (no source change, so
  unchanged from current green baseline).

## Dependency order

Single edit; no ordering concerns. Documentation only — nothing compiles
differently.

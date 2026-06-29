# Implementation Plan — go->goal upgrade skill

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `.claude/skills/go-to-goal-upgrade/SKILL.md` | The skill: frontmatter (name/description/compatibility) + the five-step pipeline, scope rules, idiom catalogue, carry-forward gotchas, report template. |
| `.claude/skills/go-to-goal-upgrade/scripts/scope-guard.sh` | Classify a target as single .go file / single package dir / multi-package module (refuse the last). |
| `.claude/skills/go-to-goal-upgrade/scripts/rename.sh` | Rename .go->.goal within scope; preserve package clause + build-tag comments; flag reserved-word (match/enum/assert) bare-identifier collisions. |
| `.claude/skills/go-to-goal-upgrade/references/idiom-catalogue.md` | The manual-idiom transforms (iota->enum, type-switch->sealed+match, method->free fn, fallible->Result/?) with convert-vs-refuse rules and goal syntax. |
| `.claude/skills/go-to-goal-upgrade/references/example-walkthrough.md` | The dogfood example: source, commands, before/after, build proof. |
| `.claude/skills/go-to-goal-upgrade/examples/before/` | The small self-contained Go source used for the dogfood (a COPY, purpose-made). |
| `.claude/skills/go-to-goal-upgrade/examples/after/` | The upgraded idiomatic goal output, build-verified. |

### Modified Files
| File | Changes |
|------|---------|
| `prd.json` | Set SEAM-007 `passes: true` (post-verify). |
| `progress.txt` | Append the SEAM-007 entry. |

No project source files (internal/, selfhost/, cmd/) are modified — the skill is
docs+assets, invisible to the Go toolchain and to project.Discover.

## Package Structure

```
.claude/skills/go-to-goal-upgrade/
  SKILL.md
  scripts/
    scope-guard.sh
    rename.sh
  references/
    idiom-catalogue.md
    example-walkthrough.md
  examples/
    before/   (Go source, COPY)
    after/    (idiomatic goal, build-verified)
```

## Dependency Graph

1. `references/idiom-catalogue.md` — the knowledge core (no deps).
2. `scripts/scope-guard.sh`, `scripts/rename.sh` — mechanical helpers (no deps).
3. `examples/before/` + dogfood run -> `examples/after/` (uses scripts + goal binary).
4. `references/example-walkthrough.md` — documents step 3.
5. `SKILL.md` — ties 1-4 together (references all).

## Interface Contracts

- `scope-guard.sh <target>` -> prints `FILE`, `PACKAGE`, or `MODULE:<reason>` and
  exits non-zero on MODULE (refuse).
- `rename.sh <scope-dir-or-file>` -> renames .go->.goal in place within scope,
  prints a per-file report of reserved-word collisions found.
- Verify: `goal build <scope-dir>` (package) or temp single-file package (file)
  must exit 0.

## Integration Points

The skill drives the existing `goal` CLI: `goal fix [-inplace]` (Step 2),
`goal build`/`goal check` (Step 4). No code-level integration; it orchestrates
the shipped binary. Reference docs link to `docs/by-example.md` and `goal ai`.

## Testing Strategy

The "test" is the dogfood: run the full pipeline on `examples/before/` producing
`examples/after/`, and prove `goal build` (or `goal check`) is green on the
output. No Go unit tests are added (the skill is not Go code). The project gates
(`task check`, `task build`, `task fixpoint`) are the regression guard and must
stay green (they are unaffected since the skill lives outside the build tree).

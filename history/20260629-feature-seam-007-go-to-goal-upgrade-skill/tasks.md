# Tasks — go->goal upgrade skill

- [ ] T1: Write `references/idiom-catalogue.md` — the four manual-idiom transforms
      with convert-vs-refuse rules, goal syntax, and carry-forward gotchas.
- [ ] T2: Write `scripts/scope-guard.sh` — classify FILE / PACKAGE / MODULE, refuse MODULE.
- [ ] T3: Write `scripts/rename.sh` — .go->.goal in scope, preserve package
      clause + build-tag comments, flag reserved-word collisions.
- [ ] T4: Create `examples/before/` — small purpose-made Go package exercising all
      four idioms (iota+method, type-switch over closed set, fallible (T,error),
      a goal-fix candidate).
- [ ] T5: Dogfood — run the pipeline on a COPY of before/ to produce
      `examples/after/`; prove `goal build`/`goal check` green.
- [ ] T6: Write `references/example-walkthrough.md` documenting the dogfood
      (commands, before/after, build proof, converted-vs-refused summary).
- [ ] T7: Write `SKILL.md` — frontmatter + five-step pipeline + scope rules +
      idiom catalogue link + carry-forward gotchas + report template.
- [ ] T8: Verify project gates: `task check`, `task build`, `task fixpoint` green.
- [ ] T9: Commit; set SEAM-007 passes:true; append progress.txt.

Dependency order: T1,T2,T3 -> T4 -> T5 -> T6 -> T7 -> T8 -> T9.

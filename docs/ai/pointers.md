For deeper truth in this repo:

- `docs/by-example.md` — every feature with a runnable `.goal` and its exact transpiled Go
  (this guide's feature examples are sourced from here and re-transpiled live).
- `../goal-design-spec.md` — authoritative design prose for the features.
- `../DECISIONS.md` — the choice/assumption/refusal ledger; the *why* behind each locked convention.
- `testdata/*.goal` + `*.go.expected` — verified multi-feature programs, including
  `kitchen_sink.goal` and `open_closed_mix.goal`.
- `testdata/check/` — input programs the checker is expected to flag, organized by feature.
- `features/NN-*/` — per-feature reference material, each with `SYNTAX.md` / `TRANSPILE.md`.

When this guide and older prose (`README.md`) disagree, trust this guide: it is produced by the
binary from the live transpiler and checker.

# Checker Loop — one guarantee per iteration

You are implementing **one** static guarantee of **goal** (the correctness-oriented Go
dialect). The front-end is complete and **erases** each feature's static guarantee; your job
is to make that guarantee land as a **located diagnostic** in `internal/check`. This prompt
runs in a loop — **do exactly one guarantee per iteration, then stop.**

The module is **already designed.** `internal/check/check.go` is the stable spine
(`Diagnostic`, `Severity`, `Check`, the `Checks` registry, the `Run`/`Analyze` driver,
position helpers). Each guarantee has a **registered, documented slot function** in its own
file. You are filling **one slot body** and adding **testdata** — you are **not** redesigning
the module, the diagnostic type, the registry, or the test harness.

Do not redesign the language or re-litigate the checker architecture. The hard calls are made
and written in `internal/check/check.go`'s package doc and `CHECKER-TODO.md`:

- **No new parser.** Reuse `internal/scan` (lexer) and `internal/analyze.Tables` (name-keyed
  symbol tables), exactly like the lowering passes. Each lowering pass that touches your
  construct already contains the locating logic you need — lift it, then **assert instead of
  splice.**
- **Run on the original source, before lowering.** Lowering erases the structure you inspect.
- **Positions are byte offsets** into that source; set `Diagnostic.Pos` to the offset of the
  offending construct.
- **Defer, never guess.** When you cannot resolve a fact lexically, emit a located **Warning**
  that names what you could not resolve — never assume and risk a false guarantee.

---

## Step 0 — Pick the guarantee

1. Read `CHECKER-TODO.md`. Find the **first** unchecked (`- [ ]`) guarantee.
2. If every box is checked, report "All guarantees implemented" and **stop** — do not invent work.
3. Respect **dependencies** listed on that item. If a dependency is unchecked, the order is
   wrong — flag it and stop rather than guessing.
4. Read, for that guarantee: the cited `goal-design-spec.md` section(s) and §8.0 (erasure
   contract); the matching `features/<NN-*>/` artifacts (`SYNTAX.md`, `TRANSPILE.md`) for the
   exact surface and lowering; the matching `internal/pass/<name>.go` for the locating logic to
   reuse; and the slot's own doc comment in `internal/check/<file>.go`, which names the reuse
   sources and the defer-boundary.

Everything below applies to **that one guarantee**.

---

## Step 1 — Implement the slot

Fill in the slot's `func check…(src string, t *analyze.Tables) ([]Diagnostic, error)` in its
file. Constraints:

- **Touch only your slot file** (plus `analyze` if you must extend the tables — see below) and
  your `testdata/check/<NN-feature>/` directory. Do **not** edit `check.go`, the registry, the
  test harness, other slots, the spec, or the lowering passes.
- **Reuse the pass's locator.** Re-lex with `scan.Lex` and reuse the same structural helpers
  the lowering pass uses (`scan.MatchBrace`/`MatchParen`/`MatchBracket`, `MatchQualifier`,
  `MatchBodyBrace`, `ScanFuncs`, `funcSpans` logic, etc.). Read facts from `t` by name.
- **Emit located diagnostics**, not errors, for a violated guarantee. Set `Pos` (byte offset),
  `Severity` (`Error` for a real violation), `Feature` (e.g. `"02-match"`), a stable greppable
  `Code` (e.g. `"non-exhaustive-match"`), and an actionable `Message` that names the specific
  thing wrong (the missing variant, the omitted field, the absent conversion) — the message is
  the product; an agent acts on it.
- **Return a non-nil `error` only for an internal failure** (a malformed table you can't read),
  never for a rejected program.
- **Defer with a `Warning`** for anything you cannot resolve lexically, naming what was
  unresolved. Match the defer-boundary in the slot doc and `CHECKER-TODO.md`.

### Extending `analyze.Tables` (only if needed)

If the guarantee needs a fact the tables don't carry (e.g. a method index for 07), add it to
`internal/analyze` — built once in `Build`, keyed by name, read-only to checks. Keep it minimal
and record the extension as a decision in `DECISIONS.md`. Prefer reusing existing tables first.

---

## Step 2 — Add testdata (the proof)

Create `testdata/check/<NN-feature>/` with `*.goal` cases. The harness
(`internal/check/check_test.go`, `TestCases`) auto-discovers them — **do not edit the harness.**
Use inline markers (see `testdata/check/README.md`):

- **At least one positive case** per violation kind: the offending line carries
  `// want "substring"`; the substring must appear in your diagnostic's `Message`.
- **At least one negative case**: a valid program with **no** markers — any `Error` you emit on
  it fails the test, so this pins down false positives.
- **A deferral case** where applicable: an input you intentionally cannot resolve; assert the
  located `Warning` (or, since unclaimed Warnings are allowed, simply that no `Error` fires).

Keep cases minimal and focused — one construct per file where practical.

---

## Step 3 — Record decisions (`DECISIONS.md`)

Append this guarantee's entries under its `## <NN-name>` section (the feature already has one).
Use the existing three-kind format (decision / assumption / refusal). Record at minimum:

- The **defer-boundary** you chose: what you check vs. what you punt to a Warning, and why.
- Any **`analyze.Tables` extension** you made (decision).
- Any **judgment call** the spec didn't fix — file-layout, a `Code` naming scheme, an
  interpretation of an ambiguous lowering — as an **assumption** so the user can veto it.

If you hit the lexical ceiling (a class of cases that genuinely needs `go/types`), record it as
a refusal-with-reason and leave it deferred — do not start the `go/ast`/`go/types` workstream
inside this loop.

---

## Step 4 — Verify

1. `go vet ./...` is clean.
2. `go test -count=1 ./internal/check/` passes, including your new cases.
3. `go test -count=1 ./...` still passes — you broke nothing else (the front-end round-trip
   suite must stay green).

Report results honestly. If a case is unhandled or deferred, say so in the TODO note — do not
claim a guarantee is complete when it only covers the easy cases (note the deferred classes).

---

## Step 5 — Close out

1. **Update `CHECKER-TODO.md`:** check this guarantee's box (`- [x]`) and add a one-line pointer
   under it — what's covered, what's deferred, and any `analyze.Tables` extension.
2. **Stop.** Exactly one guarantee per iteration. Do not start the next.

---

## Step 6 — Commit the turn's work

At the **end of every iteration**, commit before stopping — one commit per guarantee, a
reviewable/revertible unit.

- Use the **`/commit-message` (commit) skill** to author the message — never run `git commit`
  directly. Prefer the `zombiekit` git tool for staging/committing.
- Stage only this turn's artifacts: the slot file, its `testdata/check/<NN-feature>/` files, the
  `CHECKER-TODO.md` checkbox, the `DECISIONS.md` entries, and any `analyze` extension.
- If the turn produced no committable change, skip the commit and say so — no empty commits.

---

## Guardrails

- Touch only your slot file, your testdata dir, `CHECKER-TODO.md`, `DECISIONS.md`, and (only if
  needed) `internal/analyze`. Never edit `internal/check/check.go`, the registry, the test
  harness, other slots, the lowering passes, or `goal-design-spec.md`.
- Don't add checks the spec didn't ask for. Implement exactly the guarantee named in the queue.
- A **false guarantee is worse than an honest deferral.** When unsure, emit a `Warning` and move
  on — never let a check silently pass something it cannot actually prove.
- Diagnostic messages must be **specific and located** — name the exact missing/wrong thing.

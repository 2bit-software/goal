---
name: go-to-goal-upgrade
description: >-
  Upgrade an existing Go codebase to idiomatic goal — the Go Augmented Language —
  scoped to a SINGLE .go file or a SINGLE go.mod package directory. Runs the full
  idiomatization pipeline proven across the self-host effort: scope guard, rename
  (.go -> .goal), the mechanical `goal fix` autofixer, then the manual idiom
  transforms the autofixer does not do (iota const block -> enum, type-switch
  over a closed interface -> sealed interface + exhaustive match, method on a
  would-be-enum -> free label function, fallible (T,error) -> Result/?), and a
  verify+report pass. Use this whenever the user wants to "convert Go to goal",
  "upgrade this package to goal", "make this idiomatic goal", "port a .go file to
  goal", or "modernize Go with enums/match/Result". Scoped to one file or one
  package at a time — it refuses a whole multi-package module.
compatibility: Requires the `goal` binary on PATH (or build it with `task build`
  -> ./bin/goal); git and a POSIX bash for the helper scripts.
---

# go -> goal upgrade

Turn an existing Go file or package into idiomatic **goal**. goal is a strict Go
superset, so valid Go already parses; this skill is about reaching the deep
idioms — enums, exhaustive `match` over sealed interfaces, and `Result`/`?` —
that a literal port leaves on the table. It packages the entire pipeline proven
across SEAM-002..006 of the self-host idiomatization, including BOTH the
mechanical autofixer pass and the manual transforms the autofixer deliberately
will not perform.

The unit of work is **one .go file OR one go.mod package directory** — the same
per-package model the self-host used. The skill refuses a whole multi-package
module: run it once per package instead.

## Prerequisites

- `goal` on PATH, or build it: `task build` (produces `./bin/goal`). Use that
  path if `goal` is not installed globally.
- Reference reading: `references/idiom-catalogue.md` (the transforms and their
  refusal rules) and `references/example-walkthrough.md` (a full dogfood run).
  The canonical language reference is `docs/by-example.md` / `goal ai`.

## The pipeline

Run these five steps in order. Helper scripts live in `scripts/`.

### Step 0 — Scope guard

```bash
scripts/scope-guard.sh <target>
```
Prints `FILE <path>`, `PACKAGE <dir> <pkg>`, or `MODULE <reason>` (and exits 2).
If it reports `MODULE`, STOP: a multi-package module is out of scope. Tell the
user to point the skill at one package directory (or one file) at a time. Do not
proceed on a module.

### Step 1 — Rename (.go -> .goal)

```bash
scripts/rename.sh <target>
```
Renames each `<name>.go` -> `<name>.goal` (via `git mv` in a repo), keeps the
`package` clause unchanged, preserves build-tag comments verbatim, and REPORTS
any reserved-word identifier collisions. goal reserves three words beyond Go's
keywords: **`match`, `enum`, `assert`**. Go source that uses any of those as a
BARE identifier is rejected by the goal parser and must be renamed first (e.g.
`enum` -> `enumDecl`); `enumOf`/`enumName`/`.Enum` are fine. Fix any reported
collision before continuing. After renaming, `goal check <scope>` should be `ok`
(valid Go is valid goal).

### Step 2 — Autofix

```bash
goal fix -inplace <scope>     # or without -inplace to preview on stdout
```
The mechanical pass: it converts in-function `(T,error)` + manual `if err != nil`
propagation into `Result`/`?` where it can prove it safe. It writes fixed files
in place (`-inplace`) and prints a report to STDERR:
- `skipped: [result-sig] …` — it will NOT lift a signature it cannot prove (e.g.
  an exported function with callers it cannot see, or a non-propagating return).
  These are the MANUAL step's job (Step 3, idiom 4).
- `suggestion: [call-site] …` — advisory; manual error handling it noticed.

Report what `goal fix` changed vs skipped. The skips are expected and feed Step 3.

### Step 3 — Manual idioms

Apply the transforms `goal fix` does not do. Full rules and goal syntax are in
`references/idiom-catalogue.md`; the summary:

1. **iota const block -> `enum`** — when the set is closed tags with no
   numeric-identity / wire / ordering dependence. KEEP iota (documented) when the
   value is an array index, a wire/serialized value, or relied-on ordering.
2. **type-switch over a closed scrutinee -> `sealed interface` + exhaustive
   `match`** — seal the interface (`sealed interface I { … }`; embed for
   hierarchies: `sealed interface Expr { Node }`), mark implementors with
   `type T struct implements I { … }`, and rewrite `switch x.(type)` as `match x
   { *T(b) => … }`. A plain switch over a sealed type is a §9 compile error.
   REFUSE when the interface is genuinely open/extensible or the arms' concrete
   types are not all in scope.
3. **method on a would-be-enum -> free label function** — an enum lowers to an
   interface; Go forbids a method on it. Move `(e E) String() string` to a free
   `func ELabel(e E) string { return match e { … } }`; update `%s` callers.
4. **exported fallible `(T,error)` -> `Result`/`?`** — lift pure single-value
   propagation to `Result[T,error]` (`return Result.Ok(v)` / `Result.Err(e)`)
   and collapse call sites with `?`. Update cross-package callers in lockstep.
   REFUSE (documented semantic non-fit, not a scope problem): error
   ACCUMULATORS (`[]error`), MULTI-VALUE returns, comma-ok control flow, and any
   caller outside the chosen scope ("cross-package consumer not in scope").

**Carry-forward gotchas** (the skill must honor and surface these):
- enum/sealed **zero value is nil** — anywhere Go relied on the 0th iota being
  the default, set the field explicitly (e.g. `Mode.ModeNone`).
- enums **cannot carry methods** (idiom 3 exists because of this).
- value-position `match` lowers only as `x := match …`, `var x T = match …`, or
  `return match …` — restructure to one of those three shapes.
- cross-package enum/sealed idioms rely on the compiler's whole-program
  enum/sealed-fact propagation; within a single file/package scope a consumer in
  another package is a documented non-fit.

### Step 4 — Verify + report

```bash
goal check <scope>     # checker (idioms + exhaustiveness)
goal build <scope>     # transpile + go build — the real gate
```
Both must be green. Then emit a DECISIONS-style summary:

```
Scope: <FILE|PACKAGE> <name> — in scope.
Converted:
  - <type/func>: <idiom applied> (<why it fit>)
Refused / non-fit:
  - <type/func>: <reason: numeric identity | accumulator | multi-value |
    comma-ok | cross-package consumer not in scope>
Build: goal check <ok|FAIL>, goal build <ok|FAIL>.
```

## Scope rules (summary)

- IN scope: one `.go` file, or one directory holding exactly one Go package.
- OUT of scope (refuse): a directory whose `.go` files span multiple package
  directories or declare multiple package names — i.e. a multi-package module.
- Always work on the user's actual files for a real upgrade, but when
  demonstrating or testing, COPY first — never destroy source the project
  depends on.

## Worked example

`references/example-walkthrough.md` runs the whole pipeline on the bundled
`examples/before/shapes.go` (an iota+method, a closed-interface type-switch, and
two fallible functions) and produces the build-verified
`examples/after/shapes.goal`. Reproduce it to see every step, including the
`goal fix` skips that hand off to the manual step and the emitted-Go proof that
the conversion is behavior-preserving.

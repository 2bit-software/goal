# Dogfood walkthrough — `shapes`

A real, end-to-end run of the skill on a small self-contained Go package,
proving it produces buildable idiomatic goal. The source lives in
`examples/before/shapes.go`; the upgraded result in `examples/after/shapes.goal`.
Both are checked in so you can reproduce and diff them.

`shapes` was chosen because it exercises every manual idiom plus the autofixer
boundary in one ~90-line file: an iota `Kind` with a `Label()` method, a closed
`Shape` interface with a type-switch, and two fallible `(T, error)` functions
(one direct, one propagating).

## Run

```bash
SKILL=.morse-code/skills/go-to-goal-upgrade
SCRATCH=$(mktemp -d)                       # work on a COPY, never tree source
cp $SKILL/examples/before/shapes.go $SCRATCH/

# Step 0 — scope guard (PACKAGE => in scope)
$SKILL/scripts/scope-guard.sh $SCRATCH
# -> PACKAGE <dir> shapes

# Step 1 — rename .go -> .goal
$SKILL/scripts/rename.sh $SCRATCH
# -> renamed shapes.go -> shapes.goal  (no reserved-word collisions; package clause kept)
goal check $SCRATCH                        # valid Go is valid goal: "ok"

# Step 2 — autofix
goal fix -inplace $SCRATCH/shapes.goal
```

`goal fix` reports (and changes nothing here):

```
shapes.goal:74: skipped: [result-sig] `ParseDim` has a non-propagating return; not auto-converted to Result
shapes.goal:80: skipped: [result-sig] exported `MakeRect` has callers fix cannot see; not auto-converted to Result
shapes.goal:82: suggestion: [call-site] manual error handling in `MakeRect`, which is not Result-returning; convert its signature to use `?`
```

This is the autofixer being correctly conservative: it will not lift an exported
signature (it cannot see cross-package callers) nor a non-propagating return.
Those are the MANUAL step's job — and because in `shapes` every caller is in
scope, the skill lifts them by hand. The result is `examples/after/shapes.goal`.

## What converted (manual step)

| # | Before (Go) | After (goal) |
|---|-------------|--------------|
| 1 | `type Kind int; const ( Info Kind = iota; … )` | `enum Kind { Info; Warn; Fatal }` |
| 2 | `func (k Kind) Label() string { switch … }` | `func KindLabel(k Kind) string { return match k { … } }` |
| 3 | `type Shape interface{ Area() float64 }` + `switch s.(type)` | `sealed interface Shape { Area() float64 }` + `match s { *Circle(v) => … }` |
| 4 | `func ParseDim(...) (float64, error)` / `MakeRect` with `if err != nil` | `Result[float64, error]` / `Result[*Rect, error]` with `?` |

## Verify

```bash
goal check examples/after   # ok
goal build examples/after   # builds clean (transpile + go build)
```

Both are green. The emitted Go confirms the lowering:

- `enum Kind` -> `type Kind interface{ isKind() }` + `Kind_Info/Warn/Fatal`
  structs + `isKind()` markers;
- `match k` -> a Go `switch k.(type)`;
- `sealed interface Shape` -> `interface{ Area() float64; isShape() }` with each
  implementor emitting `isShape()`; the exhaustive `match s` -> a type switch
  whose default is `panic("unreachable: non-exhaustive sealed match …")`;
- `Result[T, error]` -> native `(ok T, err error)`; `ParseDim(w)?` ->
  `ww, err := ParseDim(w); if err != nil { return ok, err }`.

The `?` round-trips back to the exact manual `if err != nil` propagation we
started from — proof the conversion is behavior-preserving.

## Report (DECISIONS-style)

```
Scope: PACKAGE shapes (1 file, 1 package) — in scope.
Converted:
  - Kind: iota const block -> enum (pure tag set; no numeric/wire/ordering use)
  - Kind.Label(): method on would-be-enum -> free func KindLabel + match
  - Shape: closed interface + type-switch -> sealed interface + exhaustive match
  - ParseDim, MakeRect: (T,error) -> Result/?; MakeRect uses ? (callers in scope)
Refused / non-fit: none in this example.
Build: goal check OK, goal build OK.
```

# goal — the **Go Augmented Language**

**A thin dialect of Go that transpiles to plain Go**, adding a handful of
correctness features that turn *silent runtime failures* into *compile-time and
test errors* you can act on immediately.

> The same Go you already write — with the footguns turned into checks, and
> nothing else changed.

## Why it exists

AI coding agents (and humans) iterate fastest when a mistake shows up as **fast,
located, machine-checkable feedback**. Go has a few notorious *silent* failure
classes that instead surface at runtime — or never:

- ignored errors (`v, _ := f()`)
- nil-pointer dereferences
- zero-value struct fields you forgot to set
- a `switch` that quietly skips a case you didn't handle

goal turns each of these into a located error — while staying **Go-shaped**, so a
model's existing Go fluency carries straight over. Every divergence from Go has to
earn its keep.

## How it works

goal is a **source-to-source** transpiler: every `.goal` program lowers to
idiomatic Go the compiler already understands. You keep Go's runtime, GC,
toolchain, and standard library for free — goal only adds the front-end
correctness layer, then gets out of the way.

A **static checker** enforces each guarantee *before* lowering, emitting located
`file:line:col` diagnostics — so a forgotten variant, dropped error, or unset field is
caught as a precise error, not a silent zero. It runs by default in the `goal` CLI
(`goal check`, and as a gate on `goal build`/`run`).

This playground is the proof: the transpiler itself is compiled to WebAssembly and
runs **entirely in your browser**. Pick a feature, edit the source, hit **Run**,
and watch it become plain, readable Go.

## What it achieves

- **Sum types & exhaustive `match`** — forgetting a case is a compile error, not a wrong answer.
- **`Result` / `Option` with `?`** — errors and absence are values you can't silently drop.
- **No accidental zero values** — every struct field is set, or you opt out on purpose.
- **Checked contracts** — `implements`, `assert`, and total struct conversions.
- **Doctests** — examples in comments become real tests that can't drift from the code.

All eleven features compose, and everything lowers to Go you could have written by
hand.

**→ Pick a feature on the left to see it live.**

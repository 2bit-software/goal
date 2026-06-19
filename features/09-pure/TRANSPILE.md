# 09-pure — transpile to Go

Governing contract: §8.5 (pure-erasure features) and §8.0 (erasure).

## Erasure vs preservation (§8.0)

- **Erased (static guarantee):** the entire `pure` annotation. Purity is a checked static fact with
  **no runtime representation** — Go has no notion of a pure function, and the verification happened
  in the checker. So `pure func` lowers to a plain `func`; nothing else is emitted. (§8.5: "`pure` →
  erased to a plain `func`.")
- **Preserved (runtime):** nothing specific to this feature — the function body is passed through
  unchanged, exactly as written.
- **Defensive `panic`:** none. The feature proves no unreachability, so no defensive panic applies.

`pure` joins `implements` (07) as a pure-erasure feature: its value lives entirely in the rejected
programs (impure bodies the checker refuses) and in optional later optimizations — never in codegen.

> **Note (not v1):** §8.5 records that a backend *may* later exploit purity for
> memoization / reordering / parallelization. That is an optimization pass, **not** a codegen
> requirement, and is out of scope here — v1 erases and emits the plain `func`.

## Input → output pairs

### 1. Pure free function

```goal
pure func square(x int) int {
    return x * x
}
```
```go
func square(x int) int {
    return x * x
}
```

### 2. Pure method

```goal
pure func (v Vec) Dot(o Vec) float64 {
    return v.x*o.x + v.y*o.y
}
```
```go
func (v Vec) Dot(o Vec) float64 {
    return v.x*o.x + v.y*o.y
}
```

### 3. Mixed pure / impure — only the marked declaration changes

```goal
pure func total(prices []int) int { … }
func report(prices []int) { fmt.Println(total(prices)) }
```
```go
func total(prices []int) int { … }
func report(prices []int) { fmt.Println(total(prices)) }
```

## Lowering rules

The transpiler is a focused recognizer (lex with `text/scanner`, span-splice, `go/format`); it does
**not** parse full Go. It performs exactly one rewrite and passes all other bytes through.

1. Scan tokens for the pair `pure` immediately followed by `func`.
2. For each, emit a replacement that deletes the span from the `pure` token's start up to the `func`
   token's start — turning `pure func` into `func` (and collapsing the intervening space).
3. `pure` in any other position (a variable/field/type identifier named `pure`) has no following
   `func`, so it is never matched and is left untouched.

This mirrors feature 06's `from func` stripping exactly; `pure` carries no tables (no modes, no
conversions) because it has no runtime effect to wire up.

## Strategy forks

None. There is a single lowering — strip the modifier — for both free functions and methods.

## Hygiene

No temporaries are synthesized, so the `__gop_` prefix is not needed here.

## Scope / not-checked (the checker's job, not built)

- **Does not verify purity.** A `pure func` body that performs I/O or mutation is *not* rejected by
  the reference transpiler — it strips the marker and emits the body verbatim. Effect verification
  (and the conservative v1 definition of "effect") is the checker's job (§4.2), excluded by the
  audit's NO-checking-yet constraint.
- **Does not exploit purity.** No memoization / reordering / parallelization — that optional backend
  optimization (§8.5) is deferred.

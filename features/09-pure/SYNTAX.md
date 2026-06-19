# 09-pure — lightweight `pure` annotation

## Final surface syntax

`pure` is a modifier keyword placed immediately before `func`, marking a function (or method) as
side-effect-free:

```goal
pure func square(x int) int { return x * x }
```

It is a **declarable-and-checked** marker, *not* a granular effect system (§4.2). The goal checker
verifies the body performs no I/O or mutation; the marker is **opt-in** (annotate where useful,
silent elsewhere).

### No open syntax choice — proceeding without a question

This feature owns **no** user-facing syntax decision, so (like 07-implements) no `AskUserQuestion`
was raised:

- The spelling `pure func` is **given by the spec** (§4.2 sample) and is the exact shape already
  settled for the parallel modifier `from func` (feature 06). Re-asking would re-litigate the
  settled "modifier-before-`func`" convention.
- `pure` is not a §9 open question — §9 lists no item for it.
- It is purely additive and erased, so there is no construction/match/control-flow surface to
  design.

The one settled convention it inherits: **modifier keywords precede `func`** (`from func`,
`pure func`), and conventional names are kept verbatim.

## Grammar

```ebnf
FuncDecl   = [ "pure" ] "func" [ Receiver ] FuncName Signature [ Body ] .
```

- `pure` may modify both a free function (`pure func f(...)`) and a method
  (`pure func (r R) m(...)`) — it sits before `func` in either case.
- `pure` is a **contextual keyword**: it is the modifier only directly before `func`. Elsewhere
  (a variable, field, or type named `pure`) it is an ordinary identifier and is left untouched.
- At most one `pure` per declaration; it composes positionally with any other future modifier in
  the `[ modifier ] func` slot.

## Worked examples

### 1. Pure free function (the common case)

```goal
pure func square(x int) int {
    return x * x
}
```

### 2. Pure method

```goal
pure func (v Vec) Dot(o Vec) float64 {
    return v.x*o.x + v.y*o.y
}
```

The modifier precedes the receiver-bearing `func`; the lowering treats it identically to a free
function.

### 3. Pure and impure side by side

```goal
pure func total(prices []int) int {
    sum := 0
    for _, p := range prices { sum += p }
    return sum
}

func report(prices []int) {       // ordinary (impure) — performs I/O
    fmt.Println(total(prices))
}
```

A `pure` function is freely callable from impure code; only the annotated declaration carries the
marker (and the checker's obligation).

## Rationale, tied to the two principles

- **Familiarity:** `pure func` reads as an English modifier on an otherwise unchanged Go function
  declaration — no new punctuation, no change to call sites. It reuses the exact `[modifier] func`
  slot established by `from func` (06), so the familiarity cost is already paid.
- **Feedback (located error):** the marker converts a *silent* class — "this function I assumed was
  pure actually mutates/does I/O" — into a located compile error at the annotated declaration
  (checker's job). Because it is opt-in, it spends no budget where purity is irrelevant.
- **Deliberately the *light* version:** a full granular effect system (declaring *which* I/O/tables)
  is refused (§4.2) — best theory, near-zero empirical evidence, high annotation cost. `pure` is the
  single cheap, checkable purity fact only.

## Resolved open questions (§9)

None. §9 lists no open question for `pure`; the spelling and posture are fixed by §4.2.

## Open against spec

- **Effect-checking is out of scope for the reference transpiler.** §4.2 says the checker "verifies
  the absence of effects" and to "keep the definition of 'effect' simple and conservative for v1."
  Per the audit's NO-checking-yet constraint, the reference transpiler does **not** define or verify
  effects — it only erases the marker. What counts as an "effect" is left to the checker workstream.

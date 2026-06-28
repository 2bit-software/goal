# US-007 "Eval functions and calls" — Completeness Audit

Scope audited: `business-spec.md` and `research-summary.md`. Findings are graded
CRITICAL (blocks implementation), MAJOR (implementer must guess), MINOR (polish).
Several findings are grounded against the current `internal/interp` code.

## CRITICAL

### C-1 — Multi-return → multi-assignment contradicts existing code, and the tuple representation is undefined
FR-3 and the acceptance criteria require `a, b := f()` to bind each returned
value positionally. But:
- `execAssign` (internal/interp/interp.go:159) currently hard-rejects
  `len(s.Lhs) != len(s.Rhs)` with `"interp: assignment has N targets but M values"`.
  A single multi-return call is exactly that mismatch (2 targets, 1 RHS
  expression). The spec never acknowledges this guard must change, nor what the
  new rule is.
- The research summary models N>1 results as "a tuple Value (KindSlice acts as
  the carrier ... only when consumed positionally)". There is no `KindTuple`
  (value.go:25-34). Using `KindSlice` makes a multi-return call
  indistinguishable from a function that legitimately returns one `[]T` slice
  bound to a single variable. The disambiguation rule (call-in-RHS-position
  unpacks; everything else does not) is the load-bearing decision and is left
  implicit. The implementer cannot write a correct `execAssign` unpack path
  without inventing it.

### C-2 — `if` is mandatory for acceptance but absent from every functional requirement
The two named acceptance tests (recursive factorial, recursive fibonacci) cannot
be expressed without an `if`, and Out-of-Scope confirms "only the minimum
`if`/`return` needed". Yet no FR mentions `if`, and its semantics are wholly
undefined: condition must be boolean? is `else` / `else if` supported? what error
on a non-boolean condition? are the comparison operators (`<=`, `==`) it needs in
place? "minimum `if`" is not a contract. The implementer must guess the entire
statement.

## MAJOR

### M-1 — `return` with no values / fall-off-the-end is undefined
FR-3 only describes `return` that "produces the function's result". Undefined:
bare `return` (zero values), and a body that reaches its end with no `return` at
all (void function). Both are reachable and need defined behavior for the
`returnSignal` unwind and the call-result contract.

### M-2 — "located" undefined error has no position to locate with
Acceptance requires "a located 'undefined' error". The existing `NotFoundError`
(env.go:12-17) carries only `Name`; `Error()` is `"undefined: "+Name` with no
position. Producing a *located* error requires the call site to attach the
callee's source position (or a wrapping error). The spec does not say where the
location comes from or how it is surfaced.

### M-3 — Receiving-side arity mismatch unspecified
FR-5 covers call-site arity (args vs params). It says nothing about the
assignment side: `a, b := f()` where `f` returns 3 (or 1). With C-1 unresolved
this is doubly undefined. Needs a loud-refusal rule symmetric to FR-5.

### M-4 — Scope/resolution discipline stated only in research, not in the contract
The business spec says "fresh scope" (FR-2) and "resolves the function by name"
(FR-4) but never states whether the call frame chains to the *defining/root*
scope (lexical) or the *caller's* scope (dynamic). Recursion and forward
references depend entirely on this. Research picks lexical-over-root, but the
acceptance contract is silent, so a reader could implement dynamic scoping and
still "pass" the literal FRs.

### M-5 — Argument-passing semantics (value vs reference) unspecified
Nothing states whether a bound argument is copied or aliased. Matters as soon as
a struct or slice is passed and mutated inside the callee. Left to guess.

## MINOR

- **m-1** Argument evaluation order (left-to-right) is unspecified.
- **m-2** No recursion-depth / Go-stack-overflow consideration; deep recursion
  has undefined failure mode and is untested.
- **m-3** Mutual recursion (`f`→`g`→`f`) is neither tested nor explicitly
  excluded. Research's pre-registration implies it works, but FR-4 says only
  "call itself (directly)" — a silent capability gap.
- **m-4** Error-message wording is illustrative, not pinned: `"interp: ... expects
  N args, got M"` (does `...` name the function?) and `"interp: cannot call
  <kind>"`. Tests need a firmer contract.
- **m-5** Zero-arg / zero-param calls (`f()`) and param-name-shadows-a-function
  collisions are untested edge cases.
- **m-6** "Open Questions: None" in both docs is optimistic: the research summary's
  multi-return modelling (C-1) is an unresolved design question presented as
  settled.

## Ambiguous language ("should/might")

The business spec is mostly imperative ("must", "produces"). The permissive
"may" usages (FR-3/FR-4) describe capabilities, not loose requirements, so they
are acceptable. The genuine ambiguity is "minimum `if`/`return` needed" (C-2) —
"minimum" is undefined.

## Assumptions

Choices the spec relies on but does not state explicitly:

1. **Multi-return is carried as a `KindSlice` "tuple"**, unpacked only when a call
   appears directly in an assignment RHS list — and there is no genuine ambiguity
   with a real `[]T` return value. (research only)
2. **Lexical scoping**: call frames chain to the root/defining `Env`, not the
   caller's scope. (research only)
3. **Top-level functions are pre-registered** in the root scope, so forward
   references resolve — which incidentally enables mutual recursion even though
   only direct self-recursion is claimed.
4. **`return` is implemented as a sentinel error (`returnSignal` carrying
   `[]Value`)** unwound at the call boundary. (research only)
5. **Only single-call RHS multi-return is consumed** — no support for Go's
   `f(g())` argument-forwarding of a multi-value call into another call.
6. **`FuncValue` is extended in place** to hold `*ast.FuncDecl` + defining `*Env`
   (the type already exists at value.go:77).
7. The `if`/comparison operators needed for factorial/fibonacci (`<=`, `*`, `-`)
   are assumed already available from prior stories.

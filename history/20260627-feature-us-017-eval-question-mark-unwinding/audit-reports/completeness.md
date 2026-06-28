# US-017 Eval question-mark unwinding — Completeness Audit

Scope: business-spec.md, research.md, technical-requirements-research.md for US-017
(postfix `?` as non-local early return in `internal/interp`). Audited against the
prd.json acceptance criteria and the named corpus oracle fixtures
(`features/05-question-prop`, `features/06-error-e`).

## Summary of findings

- CRITICAL: 1
- MAJOR: 4
- MINOR: 5

---

## CRITICAL

### C1 — The named oracle fixtures do not exercise the behaviors the acceptance criteria require

The acceptance criteria (prd.json AC2 and business-spec) assert the test
"asserts `?` returns early on Err/None, continues on Ok/Some, and applies the
converted error for closed-E", and the closed-E AC says the conversion is
"verified over the 06-error-e `from` shape". research.md lists these fixtures as
the "oracle". But the fixtures' own code never produces the inputs those
criteria need:

- `qprop_result.goal`: `readFile` and `parse` both `return Result.Ok(...)`
  unconditionally. The **Err early-return path is never reachable** from this
  fixture — only Ok-continue is exercised.
- `qprop_option.goal`: `find` and `parent` both `return Option.None`. The
  **Some-continue path is never reachable** — `find(name)?` always early-returns
  None, so `p := parent(...)?` and `Option.Some(p)` are dead.
- `qclosed_prop_from.goal`: `parse` returns `Result.Ok(Config{Raw: s})`
  unconditionally, so `cfg := parse(s)?` **always succeeds and the `from`
  conversion (`ParseError -> AppError`) never fires**. Yet the closed-E AC claims
  it is "verified over the 06-error-e `from` shape".

Only `qclosed_prop_same.goal` can actually produce an `Err` (via `parse("")`).

Consequence: the test cannot satisfy the acceptance criteria by "running the
shapes". It must fabricate Err/None/converted-error scenarios that the fixtures
do not contain, and the spec never says how (synthetic Variant construction?
calling the funcs with crafted args? bespoke `.goal` snippets in the test?). As
written, the criteria phrase "verified over the 05/06 shapes" is **not
achievable from those shapes**, so the AC is unverifiable as specified. This
needs to be resolved before implementation — either point to (or add) fixtures
that actually err/none/convert, or rewrite the AC to describe the synthetic test
inputs the interpreter test will construct.

---

## MAJOR

### M1 — `?` inside a method body is silently unsupported (zero FuncSig)

technical-requirements-research.md, Plan step 1: "push the callee's `FuncSig` in
`callFunc` (and **a zero sig in `callMethod`**), pop via defer." A zero `FuncSig`
has no Result/Option mode, so by FR-5 any `?` evaluated inside a method body
would hit the "non-Result/Option function" branch and **refuse**, even when the
method legitimately returns `Result[T,E]` or `Option[T]`. Nothing in the
business-spec scopes out "`?` inside methods" — Out of Scope only lists implements
dispatch (US-018) as a separate concern, not methods that return Result/Option.
This is an undeclared functional gap: either methods-with-`?` are intentionally
unsupported (then say so, and FR-5's "descriptive refusal" should explain it), or
`callMethod` must thread a real sig. The corpus fixtures are all package-level
funcs, so this gap is invisible to the acceptance shapes but would surface on
real programs.

### M2 — "Open Questions: None" is false; unresolvable callee-E silently skips conversion, contradicting the Error Handling guarantee

business-spec Open Questions states "None ... fully determined". But
research.md / technical-requirements-research.md resolve a real semantic question
with a silent default: the callee error type is read "off the `UnwrapExpr`
operand **when it is a direct call**", and "**Same-E (or unresolvable callee)
needs no conversion**". So when the `?` operand is *not* a direct call (a
variable holding a Result, a nested expression, a method result), the callee E is
unresolvable and **no conversion is applied** — the propagated error is re-wrapped
as the caller's `Result.Err` unchanged.

This contradicts the Error Handling section: "A closed-E propagation that
requires a `from` conversion which is not registered is a located, descriptive
refusal (never a silent zero)." For an unresolvable-callee closed-E mismatch, the
interpreter cannot even tell a conversion is *required*, so it silently emits a
type-mismatched error instead of refusing. The contradiction (and the fact that
this is an open design question, not "None") should be recorded and the intended
behavior chosen.

### M3 — `qprop_erronly` (bare `func error` callee) contradiction between AC and Out of Scope; non-nil error behavior undefined

`qprop_erronly.goal` lives in `features/05-question-prop` and uses `clean()?`
where `clean()` returns `error`. The acceptance criteria say the test runs "over
the 05-question-prop ... shapes", which includes this fixture. But business-spec
Out of Scope says: "`?` on a bare `func(...) error` callee is handled best-effort
(nil continues), but **is not part of the asserted acceptance shapes**." So it is
simultaneously in-scope (named shape set) and out-of-scope (excluded assertion) —
ambiguous which the test must cover.

Worse, only the nil case is specified ("nil continues"). The behavior of `?` on a
**non-nil** `error` is left entirely undefined: does it early-return the
enclosing `Result.Err(err)`? Convert? Refuse? `qprop_erronly`'s `clean()` returns
`nil`, so the non-nil path is never demonstrated and never specified. This is a
missing functional requirement and an untested edge case.

### M4 — No handling specified when the `?` operand is not a Result/Option value

Every FR and the Error Handling section addresses the **enclosing function's**
return shape (FR-5). None addresses the **operand's** runtime shape. If
`expr?` evaluates `expr` to something that is not a Result/Option `Variant`
(e.g. an `int`, `nil`, or a struct — reachable because `internal/interp` "must
NOT gain a dependency on internal/typecheck" and only reads `sema.Info`), the
spec does not say what `evalUnwrap` does. The implied happy path
(`Variant.Tag`/`payloadValue`) would panic or misbehave on a non-variant. A
located, descriptive refusal should be specified here too, mirroring FR-5.

---

## MINOR

### m1 — Empty FuncSig stack / `?` outside any `callFunc`

The current-function sig is "an interpreter-held stack pushed in `callFunc`".
If `?` is reached when the stack is empty (top-level/package-init evaluation, or
the interpreter's host entry before any frame is pushed), the behavior is
unspecified and a naive stack-top read would index-panic. Specify a refusal or
guarantee the stack is non-empty at any `?` site.

### m2 — `?` in mid-expression / nested positions and evaluation order unspecified

`ast.UnwrapExpr` "is an `ast.Expr`, so the single eval seam is a case in
`evalExpr`", which means `?` can appear anywhere an expression can (`f(a?, b?)`,
`a? + b?`, `g(h()?)`). The spec only enumerates statement positions (`name :=
expr?`, `_ := expr?`, bare `expr?`). Evaluation order and side-effect behavior
when an early return fires mid-argument-list (do already-evaluated siblings'
effects persist?) are unspecified. Either constrain `?` to the listed positions
or define mid-expression semantics.

### m3 — `from` conversion keyed by enum E **name** string; collisions and multi-hop unspecified

`FromRegistry[[2]string{calleeE, callerE}]` keys on type *names*. Two distinct
enums named the same in different packages would collide. Also only a single
direct callee-E→caller-E hop is described; transitive/multi-step conversions are
unspecified (presumably unsupported, but say so).

### m4 — Refusal message content/format unspecified

FR-5 and Error Handling require a "located, descriptive refusal" but give no
expected message text, code, or format, so the acceptance check "yields a
descriptive error" cannot assert anything specific. Specify at least the salient
substring/locating info the test should assert.

### m5 — FR-1 wording conflates expression-yield with statement progression

FR-1: "`expr?` ... yields the unwrapped value `v`, and **execution proceeds to
the next statement**." `?` is an expression; in `cfg := parse(raw)?` it yields
`v` into the surrounding expression, not "the next statement". Minor imprecision
that should read "evaluation continues with `v` substituted for `expr?`".

---

## Assumptions

- I treated the prd.json acceptance criteria quoted in the audit prompt and the
  business-spec Acceptance Criteria as the authoritative target, and the
  `features/05-question-prop` and `features/06-error-e` `.goal` fixtures as the
  concrete "oracle" the research.md references. I read those fixtures directly to
  validate coverage claims (C1, M3).
- I assumed the fixtures are golden *transpile* oracles (Go-backend `.go.expected`
  pairs) that were authored for prior stories, not runtime drivers tailored to
  US-017 — hence their funcs returning unconditional Ok/None. If US-017's test is
  expected to author its own `.goal` snippets or construct Variants directly, C1
  is a documentation gap rather than a true coverage gap, but the spec does not
  state this either way.
- I assumed `callMethod`'s "zero sig" (Plan step 1) means a `FuncSig` with no
  Result/Option mode, which FR-5 would treat as a refusal; if `callMethod`
  actually threads the method's real signature, M1 dissolves, but the plan text
  as written says "zero sig".
- I did not inspect `internal/interp` source beyond confirming file/fixture
  existence; findings are about the spec's completeness, not the (unwritten)
  implementation. No spec files were modified.

# AI-Consumer Readiness Audit — US-017 Eval question-mark unwinding

Question audited: could an AI agent implement this spec without guessing or asking
clarifying questions? Scope: business-spec.md, research.md,
technical-requirements-research.md, cross-checked against `internal/interp`,
`internal/sema`, and the `05-question-prop` / `06-error-e` fixtures.

Overall: the core happy-path semantics (Ok/Some unwrap-and-continue, Err/None
early-return) are well-pinned — terms map to real types, the value model is
fixed in `value.go`, and the control-flow mechanism (`returnSignal` recovered in
`callFunc`) exists and is correctly described. The gaps are concentrated in
(a) the closed-E `from` conversion and (b) the unstated test harness/oracle. Two
of those gaps make at least one acceptance criterion **non-verifiable as
written**, which the spec's "Open Questions: None" actively conceals.

Severity counts: **CRITICAL 1, MAJOR 3, MINOR 4.**

---

## CRITICAL

### C-1. The named oracle for AC-5 (`from` conversion) never triggers the conversion at runtime
business-spec AC-5:
> "For a closed-E Result whose callee error type differs from the enclosing
> function's error type, the `from func` conversion is applied to the propagated
> error (verified over the 06-error-e `from` shape)."

research.md:
> "qclosed_prop_from (`from func toApp` ParseError -> AppError conversion)."

The named fixture `features/06-error-e/examples/qclosed_prop_from.goal` defines:
```
func parse(s string) Result[Config, ParseError] {
    return Result.Ok(Config{Raw: s})
}
func load(s string) Result[Config, AppError] {
    cfg := parse(s)?
    return Result.Ok(cfg)
}
```
`parse` **always returns `Result.Ok`** — it has no `Result.Err` path. Therefore
`parse(s)?` always unwraps and continues; the `from`/`toApp` conversion branch is
**never reached** when this program runs. A test that runs this exact "06-error-e
`from` shape" verifies FR-4 *vacuously* — it would pass even if the conversion
code were entirely absent or wrong.

To actually exercise FR-4 an implementer must invent a different program (one
where the callee returns `Result.Err(ParseError...)`), i.e. make a guess the spec
says is unnecessary. This directly contradicts business-spec "Open Questions:
None. The propagation semantics and the `from`-conversion source are fully
determined by ... the corpus fixtures." For the conversion behavior, the corpus
fixture does **not** determine it. This is the single biggest blocker to writing
a meaningful, non-vacuous test assertion from the acceptance criteria.

---

## MAJOR

### M-1. The test oracle / data format is unspecified and the spec's "oracle" framing is misleading
technical-requirements-research.md Plan step 4 says only:
> "Tests: `internal/interp/question_test.go` over 05/06 shapes (stdlib testing,
> no testify)."

research.md labels the fixtures "Fixtures (oracle)". But the established interp
test pattern (e.g. `result_test.go`, `option_test.go`) does **not** load the
`.goal` fixture files. It embeds an inline `const xProgram = ` + "`" + `package ...` + "`" + `,
parses it via `newInterp`, calls `ip.evalExpr(call("fn", ...), ip.root)`, and
asserts on the returned `Value`'s `Kind`/`Variant.TypeID`/`Variant.Tag`/payload.
The `*.go.expected` files in the corpus are the **Go backend output**, not an
interpreter oracle — they cannot be used to assert interpreter runtime values.

So an implementer cannot tell, without guessing: do I parse
`features/05-question-prop/examples/*.goal`, or hand-write inline programs that
mirror those shapes? What exactly is asserted — a `Value` shape, stdout, a
returned error? Neither the "what is observed" nor the "what is the expected
value" half of any acceptance criterion is pinned to the actual harness. The
criteria read as runtime behavior ("evaluates to `v` and continues") but the
codebase observes behavior structurally via `evalExpr` + `Value` inspection;
that bridge is left for the implementer to infer.

### M-2. Runtime trigger condition for "callee error type differs from E" is underspecified
business-spec FR-4/AC-5 phrase the trigger as "the propagated error's type
differs from E" — a runtime-sounding condition. But the actual mechanism
(research.md / technical-requirements-research.md) resolves callee E only
*statically* off the operand:
> "Callee E type comes from `FuncSignatures[calleeName].E` (calleeName read off
> the `UnwrapExpr` operand when it is a direct call). Same-E (or unresolvable
> callee) needs no conversion."

This means: if the `?` operand is **not** a direct call (e.g. `x?` where `x` is a
variable holding a `Result.Err`), callee E is unresolvable, so **no conversion is
applied** and the raw error propagates — even if a `from` conversion *would* be
required for type-correctness. That is a silent type mismatch, and it is the
exact failure mode the Error-Handling section claims to forbid ("never a silent
zero"). The business-spec FR/AC never state this "direct call only" limitation;
it appears only in the research notes. An implementer reading business-spec alone
would build the wrong (or an over-broad) trigger. The two documents disagree on
when conversion fires.

### M-3. Cross-container `?` behavior is undefined
Acceptance criteria only pair like-with-like: Result-`?` inside a Result function,
Option-`?` inside an Option function. The spec never says what happens when the
operand's container and the enclosing function's shape differ — e.g. `?` on an
`Option.None` inside a `Result`-returning function, or `?` on a `Result.Err`
inside an `Option`-returning function. FR-2/FR-3 describe the wrap purely in
terms of the *enclosing* function shape, implying the operand container is
ignored — which would silently mis-wrap. The likely real answer is "sema
(`internal/sema/question.go`) already rejects these at check time, so interp may
assume well-typed input," but the spec never states that assumption or that
interp may rely on it. An implementer must guess whether to defend against, or
trust, container mismatch.

---

## MINOR

### m-1. "located, descriptive refusal" is not an assertable string
FR-5 / AC: "`?` in a non-Result/Option function yields a descriptive error."
"Descriptive" and "located" are not pinned to any substring or format, so two
implementers would assert different text. The codebase convention is
`interp: <pos>: <message>` (e.g. `fmt.Errorf("interp: %s: ...", call.Pos())`),
but the spec never names it. The test author ends up asserting on a message they
themselves invented — acceptable, but not derivable from the criterion alone.

### m-2. Fallible `from` conversion outcome is unspecified
`sema.ConvEntry` carries a `Fallible bool` (a `from func` may return `(T, error)`
/ a `Result`). The spec assumes the conversion yields a plain converted error
("`Result.Err(convertedError)`") and is silent on what happens if the `from func`
itself *fails* during propagation. Undefined behavior for an existing data shape.

### m-3. Bare `func(...) error` callee semantics ("nil continues") are vague
Out of Scope: "`?` on a bare `func(...) error` callee is handled best-effort (nil
continues), but is not part of the asserted acceptance shapes." Under the
universal tagged-union `Value` model, what "nil continues" yields (what value
`expr?` evaluates to, what a non-nil error early-returns as) is not defined. It is
explicitly de-scoped from acceptance, so low-risk, but an implementer touching it
still has to guess.

### m-4. Spec overclaims determinacy
business-spec "Open Questions: None ... fully determined by the existing sema
facts ... and the corpus fixtures." Given C-1 (fixture cannot verify the
conversion), M-1 (no oracle), and M-2 (conversion trigger differs between docs),
this claim is false and, worse, suppresses the questions an implementer should
ask. A truthful "Open Questions" section would surface the conversion-fixture gap
and the harness choice.

---

## What IS well-specified (for balance)

- Terms are defined and map to real, existing symbols: `Result`/`Option`,
  `Ok`/`Err`/`Some`/`None`, closed-E, `from func`, `returnSignal`, `FuncSig`,
  `FromRegistry`/`ConvEntry`. Confirmed in `internal/interp/value.go`,
  `internal/interp/interp.go`, `internal/sema/sema.go`.
- The value/data format is fixed: `Variant{TypeID, Tag, payload}` with
  `resultTypeID/resultOkTag/resultErrTag/resultErrField`,
  `optionTypeID/optionSomeTag/optionNoneTag`, read via `payloadValue`.
- State transitions are explicit and unambiguous: Ok/Some -> unwrap payload +
  continue; Err -> early-return enclosing `Result.Err`; None -> early-return
  enclosing `Option.None`.
- The control-flow mechanism is correctly described and already exists:
  `returnSignal{vals}` raised mid-expression, recovered by `errors.As` in
  `callFunc`/`callMethod` (verified at `interp.go:240`/`:279`).
- The `same-E` acceptance case (AC-6) IS verifiable: `qclosed_prop_same.goal`'s
  `parse` does return `Result.Err(ParseError.Empty)` on empty input, so the
  no-conversion path is genuinely exercisable.

## Verdict

Not yet implementable without clarifying questions. The happy path could be built
blind, but a conscientious implementer would have to stop and ask at least three
questions: (1) "the `from` fixture never returns an error — what program should
actually test FR-4?" (2) "do tests load `.goal` files or use inline programs, and
do I assert on `Value` shape or output?" (3) "does the `from` conversion fire only
for direct-call operands, and is propagating an unconverted error for non-direct
operands acceptable?" Resolving C-1 and M-1/M-2 would make the spec
agent-ready.

## Assumptions

- I judged the test harness against the *existing* interp test convention
  (inline `const program` + `newInterp` + `evalExpr` + `Value` assertions, per
  `call_test.go:23`, `result_test.go`, `option_test.go`); the spec does not name
  a harness, so if a fixture-loading harness is planned elsewhere, M-1 softens.
- I read `qclosed_prop_from.goal` as the literal, current oracle for AC-5. If the
  intent is that the implementer *adapts* the fixture (adds an Err path), C-1
  downgrades to MAJOR — but nothing in the spec authorizes adapting it, and it is
  cited as a determining oracle ("fully determined by ... the corpus fixtures").
- I assumed sema (`internal/sema/question.go`) already rejects cross-container and
  unregistered-conversion `?` at check time; I did not exhaustively verify that
  sema covers every case interp would otherwise face, so M-3's "interp may trust
  well-typedness" is my inference, not a spec statement.
- "Closed-E" / `ModeResultClosed` and the open-vs-closed distinction were taken
  as already-established US-012..US-016 vocabulary (confirmed present in
  `internal/sema`), not re-litigated here.
- I did not run the interpreter against the fixtures (the `?` eval seam is the
  unimplemented subject of this story); findings are from static reading of the
  specs, the fixtures, and the surrounding interp/sema code.

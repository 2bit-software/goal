# AI-Consumer Readiness Audit — US-007 Eval functions and calls

Scope audited: `business-spec.md` + `research-summary.md`, judged against the real
seams in `internal/interp/` (interp.go, eval.go, env.go, value.go) and the AST node
shapes in `internal/ast/ast.go`.

Verdict: implementable in outline — every AST seam the spec leans on (`FuncDecl`,
`FuncType.Params/Results`, `CallExpr`, `ReturnStmt`, `IfStmt`) exists as described,
and the `returnSignal`-as-sentinel-error pattern threads cleanly through the
existing `execStmt`/`evalExpr` dispatch. But the multi-return value model is
internally contradictory, and the test-observation path is undefined. An agent
would have to guess on those.

## CRITICAL

### C-1: Multi-return tuple carrier has no discriminator and collides with real slices
FR-3 / AC "a multi-assignment receives each positionally" depends on the research's
proposed model: *"a call producing N>1 results yields a tuple Value (KindSlice acts
as the carrier at the call site only when consumed positionally)."* This is
unimplementable as written:

- `value.go` has no `KindTuple`. A multi-return is proposed to reuse `KindSlice`,
  but a genuine slice value (`s := makeSlice()`) is *also* `KindSlice`. Given a bare
  `Value{Kind: KindSlice}`, nothing distinguishes "2-tuple from a 2-result call"
  from "an actual 2-element slice." The spread logic has no signal to key off of.
- `execAssign` (interp.go:158-161) hard-rejects any count mismatch:
  `if len(s.Lhs) != len(s.Rhs)` → error. A real multi-assign `a, b := f()` is
  `len(Lhs)==2, len(Rhs)==1` and is rejected *before* any spreading can happen. The
  spec never states that this guard must be reworked, nor what the new rule is.

An agent must invent the entire protocol: when does a call wrap its results in a
slice (presumably keyed on `len(FuncDecl.Type.Results)` flattened names), and how
does `execAssign` decide to spread vs. assign-as-single. The research's stated
solution actively misleads. This needs a concrete rule, e.g. "when `len(Lhs)>1 &&
len(Rhs)==1 && Rhs[0] is *ast.CallExpr`, evaluate it, require a `KindSlice` of
length `len(Lhs)`, and bind positionally."

## MAJOR

### M-1: No defined seam for tests to observe a return value
"User Interactions" / acceptance say tests "assert returned values," but the only
public entry is `Interp.Run() error` (interp.go:48), which runs `main` and returns
*only an error* — there is no API that calls a function and yields its `Value`, and
nothing surfaces a variable's value after a run. The spec never defines how a test
extracts `factorial(5) == 120`. In-package tests *could* call `ip.evalExpr` on a
hand-built `*ast.CallExpr`, but that only works if functions are already registered
in a reachable scope — see M-2. The observation path is left to guesswork.

### M-2: When/where top-level functions get registered is unspecified
FR-1 says functions are "resolvable by name"; the research says "pre-registered in
the root scope." But the timing is load-bearing and undefined: today registration
happens nowhere, and `Run()` builds `main`'s scope as `ip.root.NewChild()`
(interp.go:53). If registration happens only inside `Run()`, a test that drives
`evalExpr` directly (the only apparent observation path, M-1) sees an empty root and
every call fails with `undefined`. The spec must state that registration happens in
`New()` (or an explicit step) so funcs exist independent of `Run()`.

### M-3: "located undefined error" contradicts the reused error type
AC says calling an undefined name yields a *"located 'undefined' error,"* and Error
Handling says to reuse "the existing `*NotFoundError`." But `NotFoundError`
(env.go:12-17) carries only `Name` and renders `"undefined: " + Name` — it has **no
position/location**. An agent cannot write an assertion for "located" against this
type without inventing a position field. Either drop "located," or specify that
`NotFoundError` (or a wrapper) must gain a `token.Pos`. As written the AC is not
satisfiable from the prescribed type.

## MINOR

### m-1: Parameter flattening across multi-name fields not called out
`FuncDecl.Type.Params` is a `*FieldList` whose `Field.Names` can hold several names
sharing one type (`func f(a, b int)`, ast.go:56-60). FR-2's "bind positionally to
declared parameters" requires flattening `Names` across `Fields`. Standard Go-AST
handling, but unstated; the arity check in FR-5 depends on counting flattened names,
not fields.

### m-2: `if`/`return` semantics left as "minimum needed"
Out-of-scope says only the `if`/`return` "needed for recursive factorial and
fibonacci." `IfStmt` (ast.go:762-768) has `Init`, `Cond`, `Body`, `Else`. The spec
doesn't say whether `Else`/`Init` must be handled or whether `Cond` must be a bool
kind. Discoverable from fib (`if n < 2 { return n }`), but an agent guesses the
boundary.

### m-3: Acceptance criteria omit concrete inputs/outputs
"returns the correct factorial" / "correct fibonacci number" pin no input N or
expected value, so the test inputs (e.g. `factorial(5)==120`, `fib(10)==55`) are the
agent's choice. Conventional, but not assertion-ready as written.

### m-4: Arity-mismatch error format is a template, not a string
Error Handling gives `"interp: ... expects N args, got M"` with a literal `...`
(presumably the function name). Fine for substring assertions, but the exact text
isn't pinned. Same for `"interp: cannot call <kind>"` (kind from
`Value.Kind.String()`).

### m-5: `FuncValue` extension lives only in research, not the business spec
The business spec's FR-1 says nothing about how a function becomes a value; the
needed `FuncValue{Name}` → `{*ast.FuncDecl, *Env}` extension (value.go:96-98) appears
only in the research summary. Acceptable since both files ship together, but the
business spec alone is insufficient.

## Assumptions

- I treated `business-spec.md` and `research-summary.md` as a single consumable
  package; findings that the research resolves but the business spec omits are
  downgraded accordingly (e.g. m-5).
- I assumed tests are in-package (`package interp`) and may reach unexported seams
  like `evalExpr`/`root`, since the existing test files (`eval_test.go`,
  `assign_test.go`) are in-package; M-1/M-2 are framed on that basis.
- I assumed "the existing `*NotFoundError`" in Error Handling refers to the type in
  env.go:12, which has no position field.
- I assumed `factorial`/`fibonacci` use only `if`+`return`+recursion (no loops),
  consistent with US-008 deferring the control-flow suite.
- I did not run the code or tests; seam judgments are from reading the named files.

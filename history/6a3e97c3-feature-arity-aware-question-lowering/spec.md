# Feature Specification: Arity-Aware `?` Lowering

**Feature Branch**: `main` (no branching — worked directly on main per request)
**Created**: 2026-06-26
**Status**: Draft
**Input**: User description: "Make the `?` arity-aware (the correct fix). Teach the lowering
that an error-only callee gets the 1-value form (`if __goal_err := f(); __goal_err != nil`).
Extend `internal/analyze/foreign.go` to record imported-function return arity; in-file
arities are known via `FuncSignatures`. Package-mode only (single-file is foreign-blind);
gives clean `?` on `os.MkdirAll` etc."

## User Scenarios & Testing *(mandatory)*

The "user" here is a goal-language programmer writing `.goal` source; the observable behavior
is the Go that the transpiler emits and whether it compiles.

### User Story 1 - Propagate an error-only call with `?` (Priority: P1)

A programmer calls a function whose only return value is an `error` (e.g. `os.MkdirAll`,
`json.Unmarshal`, or an in-file `func clean() error`) and propagates failure with a bare
`expr?` statement inside a `Result`-returning function.

**Why this priority**: This is the core defect. Today the emitted Go does not compile, so the
single most common shape of `?` — "do this side-effecting step, bail on error" — is unusable.

**Independent Test**: Transpile a `Result`-returning function containing `someErrOnlyCall()?`
and assert the emitted Go is `if __goal_err := someErrOnlyCall(); __goal_err != nil { return
__goal_ok, __goal_err }` and that it builds.

**Acceptance Scenarios**:

1. **Given** an in-file `func clean() error` and a `Result[T, error]` caller containing
   `clean()?`, **When** transpiled, **Then** the guard binds a single `__goal_err` (no blank
   identifier) and the package compiles.
2. **Given** a package-mode program importing a Go package with `func Mkdir(p string) error`
   and a caller containing `ext.Mkdir(p)?`, **When** transpiled, **Then** the guard is the
   one-value form `if __goal_err := ext.Mkdir(p); __goal_err != nil { … }`.

---

### User Story 2 - Discarding the value of a multi-return call still works (Priority: P1)

A programmer discards the success value of a `(value, error)` (or wider) call while
propagating its error, via a bare `expr?` or `_ := expr?`.

**Why this priority**: The fix must not regress the case that works today; the existing
two-value form must still be produced when the callee actually returns a value.

**Independent Test**: Transpile a discard `?` over a callee known to return `(T, error)` and
assert the emitted guard keeps exactly one blank identifier.

**Acceptance Scenarios**:

1. **Given** an in-file `func produce() Result[int, error]` and a caller with `produce()?`,
   **When** transpiled, **Then** the guard is `if _, __goal_err := produce(); __goal_err !=
   nil { … }` (unchanged from today).
2. **Given** a package-mode callee `func Read() (int, int, error)` and a caller with
   `ext.Read()?`, **When** transpiled, **Then** the guard discards both values:
   `if _, _, __goal_err := ext.Read(); __goal_err != nil { … }`.
3. **Given** the existing binding form `name := produce()?`, **When** transpiled, **Then**
   its output is byte-for-byte unchanged from before this feature.

---

### User Story 3 - No regression when arity cannot be resolved (Priority: P1)

A programmer uses a foreign callee in **single-file** mode (no package directory, so foreign
arity is unknowable), or a callee shape the analyzer cannot resolve (e.g. a method call, or a
non-call rhs). The feature must degrade to **exactly today's behavior** — never crash and
never break code that compiles today.

**Why this priority**: Single-file transpile is foreign-blind by design and the existing
two-value discard form already compiles for value-returning foreign callees; silently changing
it would regress working programs. Bare `expr?` and `_ := expr?` are *semantically identical*
discards (so arity must NOT be inferred from which one was written).

**Independent Test**: Transpile a single-file program with an unresolved foreign `expr?` and
assert the output is byte-for-byte identical to the pre-feature output (the two-value form).

**Acceptance Scenarios**:

1. **Given** single-file mode and an unresolved foreign discard `?` (bare or `_ :=`), **When**
   transpiled, **Then** the existing two-value form `if _, __goal_err := rhs; … }` is emitted
   unchanged — no panic, deterministic output. (Error-only foreign callees in single-file
   therefore remain unsupported; this is the accepted package-mode-only limitation.)
2. **Given** any **in-file** callee in single-file mode, **When** transpiled, **Then** arity is
   resolved exactly (in-file tables are always available) — so an in-file `func clean() error`
   gets the correct one-value form even in single-file mode.
3. **Given** a method-call or non-call `?` rhs whose callee cannot be keyed, **When**
   transpiled, **Then** the two-value default applies (same as today).

---

### Edge Cases

- **`func f()` with no error return used with `?`**: the callee yields nothing to propagate;
  this is already rejected/deferred by the must-use rules and is out of scope here — the
  arity logic must not change that behavior.
- **Multi-file package**: an error-only callee defined in a sibling `.goal` file resolves via
  the merged `BuildPackage` tables, exactly like any other in-file symbol.
- **Foreign function with a named single result** (`func Mk(p string) (err error)`): arity is
  still 1; the one-value form is emitted.
- **Method-call callee** (`f.Close()?`) and **non-call rhs** (a bare identifier / index
  expression): callee not keyable — treated as unresolved, two-value default (US3).
- **Resolved error-only callee in the binding form** (`x := clean()?` where `clean` returns
  only `error`): cannot bind a value that does not exist → goal-level diagnostic (FR-009), not
  silent broken Go.
- **Closed-E `?` and Option `?`**: untouched — these are separate passes already operating on
  a single value; this feature only affects open-E `ModeResult` lowering.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The open-E discard `?` lowering MUST emit exactly `arity − 1` blank identifiers
  before `__goal_err`, where `arity` is the callee's resolved return-value count.
- **FR-002**: The analyzer MUST record a return arity for every in-file function: a
  `Result[T, error]` function resolves to arity 2; an `Option[T]`/closed-E `Result` to 1; any
  other function to its raw count of return values.
- **FR-003**: In package mode, the foreign-enrichment step MUST record the return arity of
  imported, package-qualified Go functions referenced as `?` callees, keyed by the `alias.Func`
  the goal source uses.
- **FR-004**: The lowering MUST resolve callee arity from the analyzer tables — in-file by
  bare name, foreign by `alias.Func` selector.
- **FR-005**: When callee arity cannot be resolved (unresolved foreign callee, method call, or
  non-call rhs), the discard lowering MUST emit the existing two-value form `if _, __goal_err
  := rhs; …` — identical to today's output. Arity MUST NOT be inferred from whether the source
  wrote a bare `expr?` versus `_ := expr?` (they mean the same thing); both default identically.
- **FR-006**: For every callee whose lowering compiles today — the binding form `name :=
  expr?` and any discard over a value-returning (arity ≥ 2) callee — the emitted Go MUST remain
  byte-for-byte unchanged. (A resolved error-only discard, which does *not* compile today, is
  the case this feature deliberately changes.)
- **FR-007**: Foreign function enrichment MUST remain non-fatal and offline-testable: an
  unresolved import leaves arity unknown (triggering FR-005), never an error, and MUST be
  exercisable through an injected `DirResolver` against a `testdata` fixture. The needed-alias
  collection MUST be extended (or paired with a sibling collector) so an import referenced only
  as a `?` callee — with no `derive`/`from` use — is still parsed; otherwise enrichment no-ops.
- **FR-008**: Single-file transpile MUST stay foreign-blind — no new IO is introduced on that
  path; only in-file arities are resolved there.
- **FR-009**: When a callee resolves to arity ≠ 2 in the **non-discard** binding form `name :=
  expr?` (e.g. binding a value from a resolved error-only callee), the pass MUST emit a
  goal-level diagnostic rather than silently emitting non-compiling Go.
- **FR-010**: Foreign entries added to the signature table MUST carry only their arity and
  leave `Mode` at its zero value (`ModeNone`), so existing iterations over the table (e.g.
  the closed-E prelude check) are unaffected.

### Key Entities

- **Callee arity**: the number of values a called function returns at `?`-lowering time
  (after signature lowering), i.e. the count that determines how many blank identifiers the
  discard guard needs.
- **Foreign function record**: an imported package-level Go function's `alias.Func` key mapped
  to its return arity, produced by foreign enrichment in package mode.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A `Result`-returning function that calls `os.MkdirAll(p)?` (or any error-only
  callee) in package mode transpiles to Go that compiles — 0 `assignment mismatch` errors.
- **SC-002**: 100% of existing `?` transpile tests pass unchanged (no output regressions).
- **SC-003**: Discard `?` over a callee of arity N emits exactly N−1 blank identifiers for
  N ∈ {1, 2, 3}.
- **SC-004**: Foreign arity is verified with zero dependence on the real Go stdlib (fixture
  resolver only), keeping the analyze/check test suites offline.

## Testing Requirements *(mandatory)*

### Test Strategy

Zero-dependency stdlib `testing` (no testify). Three existing harnesses carry the load:

- **Golden regression lock**: `?` output is pinned by `features/05-question-prop/examples/
  qprop_{discard,result,option}.go.expected` (run via `TestTestdata` / single-feature
  regression). These goldens — especially `qprop_discard` (in-file arity-2 discard) — are the
  byte-for-byte FR-006 lock; they MUST stay unchanged. New goldens are added for the
  error-only and multi-return discard shapes.
- **Compile proof (SC-001)**: the package-mode `go build ./...` harness
  (`internal/pipeline/pipeline_package_test.go`, `TestTranspilePackageCrossFile`) writes the
  emitted package to a temp dir and builds it. The `os.MkdirAll`-style error-only case is
  proven by *actually compiling* it here, not by string-matching guard shape.
- **Foreign-arity unit test**: a focused `internal/analyze` test using the injected
  `DirResolver` against a `testdata` fixture. The existing `testdata/extpkg` fixture (currently
  struct-only) gains an exported `func` decl (e.g. `func Mkdir(p string) error` and a
  `(T, error)` func) so arity parsing is exercised offline.

The in-file arity model mirrors the existing reverse-direction logic in `internal/fix/
propagate.go` (which already distinguishes "the call's sole output is the error" from
value-returning calls); reuse its reasoning and any selector helper rather than reinventing.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration/Golden | Discard `?` over arity 1/2/3 callees emits 0/1/2 blank identifiers |
| FR-002 | Unit | `analyzeSig` records arity 2 for Result, 1 for Option, raw count for `func() error` / `func() (T,error)` / `func()` |
| FR-003 | Unit | Foreign enrichment records `alias.Func → arity` for imported `func` decls via fixture resolver |
| FR-004 | Integration | In-file callee resolves by name; foreign callee resolves by `alias.Func` |
| FR-005 | Golden | Unresolved foreign discard (bare and `_ :=`) emits the existing two-value form, unchanged |
| FR-006 | Golden | `qprop_discard`/`qprop_result` goldens + `name := produce()?` byte-for-byte unchanged |
| FR-007 | Unit | Import referenced only as a `?` callee is still parsed; unresolved import yields no error; resolver injectable, offline |
| FR-008 | Integration | Single-file transpile resolves in-file arity, performs no foreign IO |
| FR-009 | Integration | `x := clean()?` over a resolved error-only callee → goal-level diagnostic, not broken Go |
| FR-010 | Unit | Foreign signature entries leave `Mode == ModeNone`; `NeedsResultPrelude` unaffected |
| SC-001 | Compile | `go build ./...` package harness compiles an error-only `?` (e.g. `os.MkdirAll`) |

### Edge Case Coverage

- Error-only callee with a named result (`(err error)`) → arity 1 → one-value form.
- Sibling-file error-only callee in a multi-file package → resolved via merged tables.
- Method-call / non-call `?` rhs → unresolved → two-value default, no panic (FR-005).
- Binding form over a resolved error-only callee → diagnostic (FR-009).
- Closed-E and Option `?` → output unchanged (separate passes).

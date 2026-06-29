# Research findings — US-008 idiomatic audit: parser

Scope: intra-package, behavior-preserving idioms only (no cross-package caller
edits), per the self-host idiomatic plan. The US-003 verbatim self-host is the
behavioral oracle.

## Candidate 1: Result/? for `(T, error)` parser functions  → DOES NOT APPLY
- Exhaustive grep of `selfhost/parser/*.goal` for `func ... error` returns exactly
  two hits: `ParseFile` (exported) and `errorf` (returns nothing; it is the error
  *recorder*).
- The parser uses an **error-accumulator** design: `parser.errs []error`, appended
  by `errorf`; every internal helper returns a bare AST node (or `nil`) and never
  an `error`. There is no `if err != nil { return ..., err }` propagation anywhere
  (grep for `err != nil` / `, err` finds only `ParseFile`'s `errors.Join`).
- The single `(T, error)` function, `ParseFile(src) (*ast.File, error)`, is the
  exported entry point pinned by the oracle's ported `parser_test.go`. Converting
  it to `Result` would break the public signature the tests pin — explicitly
  forbidden. `goal fix` already refuses it (result-sig SKIP).
- Conclusion: no intra-package Result/`?` surface exists. Genuine refusal, recorded.

## Candidate 2: switch-over-in-file-enum → match  → DOES NOT APPLY
- `selfhost/parser` declares NO `enum` (it parses enums in other source; it has
  none of its own).
- Every `switch` is over a non-enum scrutinee: `token.Kind` (`type Kind int`, not
  an enum per US-005), a `token.Token`, type-switches `x.(type)` over `ast`
  category interfaces (cannot be sealed per US-007 §9), or bare boolean
  `switch { }`. None is a closed-enum scrutinee, so `match` has no legal subject.
- Conclusion: no switch->match candidate. Genuine refusal, recorded.

## Candidate 3: Option for comma-ok / bool helpers  → DOES NOT APPLY
- The bool-returning funcs (`at`, `onNewLine`, `startsType`, `atTypeParams`,
  `nameThenType`, `isTypeSwitchGuard`, `startsExpr`, `startsArmStmt`,
  `isContextual`, `startsTypeKind`) are pure predicates, not fallible lookups with
  a missing-value case; `Option` would not be behavior-preserving or meaningful.
- No internal `(T, bool)` comma-ok helper exists.

## Machine check (AC-2)
`goal fix selfhost/parser/*.goal`: no content diff on any file; the only stderr is
`parser.goal:57 skipped: [result-sig] ParseFile has a non-propagating return; not
auto-converted to Result` — a deliberate non-auto-convertible skip, i.e. zero
auto-convertible propagation sites remain.

## Confidence: High
The parser's error-accumulator architecture is unambiguous from the source, and
the machine check confirms it. Outcome mirrors US-005/006/007: a recorded DECISION
with no `.goal` source change, because the only behavior-preserving forms lie
outside `selfhost/parser` (the oracle-pinned public API) or do not exist (no enum).

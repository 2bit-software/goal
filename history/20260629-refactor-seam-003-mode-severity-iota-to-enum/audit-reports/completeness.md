# Completeness Audit — SEAM-003

## Findings

### MINOR-1: switch-true sites with embedded Mode comparisons
`question.goal` appendQuestionResolved is a `switch { case csig.Mode == ...: }`
(switch-true), not a switch-over-Mode. The Mode `==` cases must be hoisted to a
value-position `match` (bound to bools) while the non-Mode arms (Arity/EndsInError)
stay in the switch-true. Spec FR-3 covers this ("==/!=/plain-switch ... become
match"); the audit just flags that the mechanical shape differs per site. Not a
spec gap — an implementation note already captured in
technical-requirements-research.md.

### MINOR-2: zero `Diagnostic{}` / `FuncSig{}` returns
Several functions return `Diagnostic{}` or `FuncSig{}` zero literals paired with a
bool flag (e.g. `(Diagnostic{}, true)`, `(FuncSig{}, false)`). After conversion
their Severity/Mode field is nil. Verified these are never `match`ed/`String()`d
when the bool says "skip/not-ok" — callers guard on the bool first. FR-4 is
satisfied by fixing the sites that DO flow into a match (foreign.goal:222,
calleeMode). No additional spec requirement needed.

## Assessment

No CRITICAL or MAJOR findings. The spec is complete and testable; every
acceptance criterion is a command-checkable gate (task check/build/fixpoint) or a
greppable invariant (no plain switch/== over the enums).

## Assumptions

- The conversion is total (no member kept as iota) — justified by the verified
  absence of any numeric/index/range/wire use of Mode or Severity.
- "corpus behavioral tier unchanged" is proven by `task check` + `task fixpoint`
  (the existing gate topology), not a new test.

# Implementation Plan — US-012 idiomatic audit: typecheck

## File Inventory

### New Files
None.

### Modified Files
| File | Changes |
|------|---------|
| `DECISIONS.md` | Append a `## self-host idiomatic audit — US-012 (typecheck)` section recording the per-function refusals with reasons and the verification commands. |
| `prd.json` | Set US-012 `passes: true` (finalization step, after green verification). |
| `progress.txt` | Append the US-012 iteration entry (finalization step). |

No `.goal` source files change: the package survey is conclusive that no behavior-preserving,
non-oracle-pinned idiomatic conversion exists.

## Package Structure

`selfhost/typecheck/` (unchanged):
- `checker.goal` — `TypeChecker` interface + `GoTypesChecker.Check` (interface method).
- `typecheck.goal` — `Load` (exported entry), `Package`, `Diagnostic`.
- `implements.goal`, `mustuse.goal`, `nozero.goal` — depth checks returning `[]Diagnostic`.

## Dependency Graph

1. Confirm machine check: `goal fix selfhost/typecheck/*.goal` produces no source diff.
2. Author the DECISIONS.md refusal section.
3. Run verifyCommands + port gate (verify step).
4. Finalize: prd.json passes:true, progress.txt entry, commit.

## Interface Contracts

Unchanged (and deliberately so — these are the oracle-pinned signatures):
- `func Load(pkg *project.Package) (*Package, error)`
- `TypeChecker interface { Check(pkg *project.Package) ([]Diagnostic, error) }`
- `func (GoTypesChecker) Check(pkg *project.Package) ([]Diagnostic, error)`

## Integration Points

None — no source change. The DECISIONS.md entry is documentation; prd.json/progress.txt are
loop bookkeeping.

## Refusal rationale (the substance of this story)

- `Load` (exported): all internal propagation WRAPS context (`fmt.Errorf("...: %w", err)`),
  and `?` propagates the error UNCHANGED — converting would drop the context (behavior change).
  Its callees (`backend.TranspilePackage`, `parser.ParseFile`, `go/parser.ParseFile`) are
  Go-tuples, not Result, so there is nothing to `?`. It is oracle-pinned (6+ `p, err :=
  Load(...)` tuple call sites across the depth-test suite) AND has an in-tree caller (`Check`);
  the safety rule converts an exported fn ONLY when it has no in-tree callers AND no oracle
  test. Refuse.
- `GoTypesChecker.Check` (interface method): pinned by `var _ TypeChecker = GoTypesChecker{}`
  and `tc.Check(pkg)` two-value calls; changing it to Result changes the `TypeChecker`
  interface contract (the seam's whole purpose is a swappable interface). Its lone `return
  nil, err` is pure propagation, but the host cannot become Result without breaking the
  interface. Refuse.
- `litClass` (`type litClass int` + iota classElided/classGeneric): no switch over it exists
  (the only switch is a TYPE switch over `go/ast` node types); it is consumed via
  `kind == classGeneric` and a `return 0, false` numeric literal. A goal `enum` lowers to a
  sealed interface with no integer identity, breaking `==` and `return 0`. Keep as iota int.

## Testing Strategy

- Machine gate: `goal fix selfhost/typecheck/*.goal` — must show no source diff.
- Behavioral gate: the selfhost port test (`go test ./internal/selfhost -run
  TestPorted...typecheck` / via `task check`) transpiles `selfhost/typecheck` and runs the
  copied `internal/typecheck` depth tests against it.
- Project gates: `task check`, `task build`, `task fixpoint` (byte-identical bootstrap).

# Implementation Plan — SEAM-003

## File Inventory

### New Files
None. (No new tests needed: the conversion is selfhost-only and proven by the
existing internal/selfhost port gates + `task fixpoint`, exactly as SEAM-002.)

### Modified Files
| File | Changes |
|------|---------|
| `selfhost/sema/sema.goal` | `Mode` iota -> `enum Mode { ModeNone ModeResult ModeResultClosed ModeOption }` |
| `selfhost/sema/check.goal` | `Severity` iota -> `enum Severity { Error Warning }`; `String()` `if s==Warning` -> `match` |
| `selfhost/sema/resolve.goal` | `FuncSig{...Mode: Mode.ModeNone}`; `sig.Mode = Mode.ModeX`; switch over `sig.Mode` -> two value-matches for Arity/EndsInError |
| `selfhost/sema/question.goal` | `caller.Mode != ModeResult` -> bool match; appendQuestionResolved switch-true Mode cases -> precomputed bool matches; closedQuestionDiags `!known \|\| csig.Mode != ModeResultClosed` -> guarded bool match |
| `selfhost/sema/mustuse.goal` | `ok && (sig.Mode == ModeResult \|\| ...)` -> `ok &&` guarded bool match |
| `selfhost/sema/foreign.goal` | foreign `FuncSig{...}` add `Mode: Mode.ModeNone` (line ~222) — CRITICAL nil-fault fix |
| `selfhost/sema/{implements,convert,fields,assert,check}.goal` | requalify `Severity: Error/Warning` -> `Severity: Severity.Error/Severity.Warning` |
| `selfhost/backend/lower.goal` | needsResultPrelude loop `sig.Mode == sema.ModeResultClosed` -> bool match |
| `selfhost/backend/emit.goal` | `sig.Mode == sema.ModeResultClosed` (426, 1772) -> guarded bool match; FuncSig construction `Mode: sema.ModeResult` -> `sema.Mode.ModeResult` (1901,1904); calleeMode return guard + `== sema.ModeResultClosed` at 2129 -> match |
| `selfhost/typecheck/mustuse.goal` | `!ok \|\| sig.Mode != sema.ModeResult` -> guarded bool match; `Severity: sema.Error/Warning` -> `sema.Severity.Error/Warning` |
| `selfhost/typecheck/implements.goal` | `Severity: sema.Error` -> `sema.Severity.Error` |
| `selfhost/typecheck/nozero.goal` | `Severity: sema.Error` -> `sema.Severity.Error` |
| `DECISIONS.md` | record conversion, supersede US-011 "Mode and Severity stay iota" |
| `prd.json` | SEAM-003 passes:true (after green) |
| `progress.txt` | append SEAM-003 entry |

## Dependency Graph (edit order; tree is red until all land — §9/undefined-symbol)

1. Enum definitions: sema.goal (Mode), check.goal (Severity + String()).
2. Same-package sema consumers: resolve.goal, question.goal, mustuse.goal,
   foreign.goal, + Severity requalification in implements/convert/fields/assert/check.
3. Cross-package consumers: backend/{lower,emit}.goal, typecheck/{mustuse,implements,nozero}.goal.
4. Docs: DECISIONS.md.
5. Verify: task check, task build, task fixpoint.

## Interface Contracts

```goal
// sema.goal
enum Mode { ModeNone  ModeResult  ModeResultClosed  ModeOption }

// check.goal
enum Severity { Error  Warning }
func (s Severity) String() string {
    return match s { Severity.Warning => "warning"  _ => "error" }
}

// emit.goal — calleeMode must never return nil
func (e *emitter) calleeMode(x ast.Expr) sema.Mode {
    // ... guard !ok cases -> sema.Mode.ModeNone
    sig, found := e.info.FuncSignatures[id.Name]
    if !found { return sema.Mode.ModeNone }
    return sig.Mode
}
```

Conversion idioms:
- Value-position bool match: `isClosed := match sig.Mode { sema.Mode.ModeResultClosed => true  sema.Mode.ModeResult => false  sema.Mode.ModeOption => false  sema.Mode.ModeNone => false }`
- Guarded (short-circuit before match): `isClosed := false; if ok { isClosed = match sig.Mode {...} }`

## Integration Points

All FuncSig values reaching a Mode match originate from resolve.goal `funcSig`
(sets Mode), foreign.goal (must set Mode.ModeNone), or emit.goal construction
(sets Mode). All Diagnostic values reaching Severity String() set Severity
explicitly; zero `Diagnostic{}` returns are bool-gated and never rendered.

## Testing Strategy

No new tests. Gates: `task check` (includes internal/selfhost port gates that
transpile selfhost/{sema,backend,typecheck} as enums and run relocated/white-box
tests against them, plus the corpus behavioral tier), `task build`, and
`task fixpoint` (stage1==stage2 on the new enum/match source). A PostToolUse hook
runs `task check` after each edit; transient failures mid-refactor are expected —
only the final green state matters.

# Technical Requirements / Research — SEAM-003

## Prior art (proven machinery)

- SEAM-CAP (fb92fa9): cross-package enum-match lowering in the backend.
- SEAM-CAP-2 (7279312): cross-.goal-package enum/sema-fact propagation during
  the self-host build (enrichForeign reads sibling .goal sources).
- SEAM-002 (6c256c0): FuncMod/ChanDir iota -> enum, tree-wide. Established the
  exact conversion pattern reused here.

## Conversion pattern (from SEAM-002 progress notes)

1. Replace `type X int` + `const (... iota ...)` with `enum X { Variant... }`.
2. Enum ZERO VALUE is nil, NOT the first variant. iota gave ModeNone=0,
   Error=0; an enum does not. Set the field explicitly at EVERY constructor or a
   later `match` faults at runtime on nil.
3. Variant references become `Enum.Variant` (same pkg) / `pkg.Enum.Variant`
   (cross pkg): e.g. `ModeResult` -> `Mode.ModeResult`,
   `sema.ModeResultClosed` -> `sema.Mode.ModeResultClosed`, `Error` ->
   `Severity.Error`.
4. Every ==/!=/plain-switch consumer becomes `match`. For a comparison inside a
   boolean guard, bind a bool via a value-position `match`; keep any `!ok`/`!known`
   short-circuit BEFORE the match so a zero-value (nil) enum is never matched.
5. A `_ => ...` rest-arm lowers to a real Go `default` (nil-safe); full
   enumeration with no `_` lowers to a panicking default (nil faults). Use
   full enumeration where exhaustiveness is wanted and the value is always set.

## Sites identified (audit)

Definitions:
- selfhost/sema/sema.goal: `Mode` iota -> enum.
- selfhost/sema/check.goal: `Severity` iota -> enum + `String()` method.

Mode consumers (==/!=/switch -> match):
- sema/question.goal: 49, 76-82 (switch-true with Mode cases), 113, 130.
- sema/mustuse.goal: 114.
- sema/resolve.goal: 278/290/292/295 (construction), 299-304 (switch).
- backend/lower.goal: 120 (needsResultPrelude loop).
- backend/emit.goal: 426, 1772, 1901/1904 (construction), 2129, calleeMode 2235-2244.
- typecheck/mustuse.goal: 81.

Severity consumers:
- sema/check.goal String() `if s == Warning`.
- All `Severity: Error/Warning` literals (sema) and `Severity: sema.Error/Warning`
  (typecheck) — requalify to `Severity.Error` / `sema.Severity.Error`.
- typecheck/typecheck.goal:114 field type `Severity sema.Severity` (unchanged).

Critical nil-fault sources to fix explicitly:
- sema/foreign.goal:222 `FuncSig{Arity:..., EndsInError:...}` -> add `Mode: Mode.ModeNone`
  (these foreign sigs land in info.FuncSignatures, iterated by needsResultPrelude's match).
- backend/emit.goal calleeMode: guard the missing-map-key case, return sema.Mode.ModeNone.

No numeric `sema.Severity(x)` conversions exist in the tree (verified) — total conversion.

## Gates

`task check`, `task build`, `task fixpoint`; corpus behavioral tier unchanged.
Known: commit with `commit.gpgsign=false` (1Password non-interactive signing).

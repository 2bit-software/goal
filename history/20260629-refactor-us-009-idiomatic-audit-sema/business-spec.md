# US-009 Idiomatic audit: sema — Business Specification

## Overview

selfhost/sema is the first ported compiler package with a real fallible
((T,error)/Result/`?`) surface. This audit makes it read as idiomatic goal:
package-internal fallible helpers that can be expressed with Result/Option + `?`
without changing observable behavior are expressed that way, and diagnostic/mode
kinds become enums + match where they genuinely fit. Go-isms that cannot be
idiomatized without breaking behavior, changing an oracle-pinned public signature,
or requiring cross-package caller edits are recorded as deliberate decisions.

## Functional Requirements

### FR-1: Convert the one genuine in-package Result/? site
The single-file checker entry point (`Analyze`) SHALL be expressed as a
Result-returning function using `?` propagation for its parse step, provided the
conversion preserves observable behavior and the emitted Go signature.

### FR-2: Refuse non-fitting conversions with recorded reasons
Every other fallible function and every candidate enum/match kind that does NOT
fit (exported + oracle-pinned, cross-package consumed, comma-ok control-flow bool,
multi-value, error-accumulator, non-propagating tail return, or ordered iota int)
SHALL be left as-is and recorded with its reason in DECISIONS.md.

### FR-3: No machine-convertible propagation sites remain
After the audit, `goal fix` over the package SHALL report no remaining
auto-convertible propagation sites (only documented skips/suggestions for the
refused boundaries).

## Acceptance Criteria

- [ ] Fallible resolution/check functions use Result/Option with `?` where it fits;
      the audit converts every site that genuinely fits and refuses the rest.
- [ ] Diagnostic/mode kinds are expressed as enums + match where they fit, or the
      refusal is recorded in DECISIONS.md.
- [ ] `goal fix` over selfhost/sema reports no remaining auto-convertible
      propagation sites (produces no source diff; only documented skip/suggestion
      reports remain).
- [ ] sema port tests pass against the transpiled package (internal/selfhost sema
      behavioral gate green).
- [ ] `task check`, `task build`, and `task fixpoint` are green; the fixpoint is
      byte-identical across goal-c-1/goal-c-2.

## User Interactions

None directly. The artifact is the goal-source compiler package and DECISIONS.md;
behavior is observed through the compiler's tests and the self-host fixpoint.

## Error Handling

Behavior preservation is the contract: the converted `Analyze` emits the same Go
((T,error)) and surfaces a parse failure as a returned error exactly as before;
rejected programs still surface as Error-severity diagnostics, not returned errors.

## Out of Scope

- Cross-package signature changes (any exported sema API consumed by
  backend/typecheck/project/pipeline/main, or pinned by oracle tests).
- Sealing/enum-ifying Mode or Severity (ordered iota ints consumed cross-package).
- switch->match where no in-file goal `enum` scrutinee exists.
- Any change to internal/sema (the trusted source); only selfhost/sema is audited.

## Open Questions

- None. The worklist is fully enumerated by `goal fix` + a manual survey of every
  error-returning function and iota-kind type in the package.

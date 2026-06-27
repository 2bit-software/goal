# Plan Audit — Coverage

## Findings

No CRITICAL findings.
No MAJOR findings.

### Traceability

| Requirement | Plan element |
|-------------|--------------|
| FR-1 refuse effect | emitStdout denied branch returns error, no write |
| FR-2 named | CapabilityError.Cap + message includes Stdout name |
| FR-3 located | CapabilityError.Pos + Pos.String() in message |
| FR-4 nothing written | denied branch skips `write` |
| FR-5 granted unchanged | grant path untouched; regression test |

Every acceptance criterion maps to a test in the Testing Strategy. No plan
element lacks a requirement (no scope creep): the only production change is the
typed error + the gate branch + threading the position to the single call site.

## Assumptions

- Error message format `interp: <line:col>: capability denied: <Cap> not granted`
  matches the interpreter's existing located-refusal style.

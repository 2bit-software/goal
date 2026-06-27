# Plan Audit — Coverage

Every acceptance criterion maps to a plan element:

- AC "Some construction" -> step 2/3 (evalOptionCtor + evalCallMulti interception)
  + option_test.go assertion.
- AC "None construction" -> step 4 (evalSelector guard) + option_test.go.
- AC "match Some binds unwrapped" -> step 5 (armScopeFor unwrap) + match test.
- AC "match None runs None arm" -> selectMatchArm (unchanged) + match test.
- AC "unit test over 04-option shape" -> option_test.go (new file).

No scope creep: no plan element lacks a backing requirement.

No CRITICAL/MAJOR findings.

## Assumptions

- Existing test helpers (`newInterp`, `call`) are reused, not redeclared.

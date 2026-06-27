# Tasks — US-010 Eval builtins and methods

- [ ] T1: Add `panicSignal{value Value}` error sentinel to interp.go.
- [ ] T2: Add `methods` registry field + `registerMethods()` (called from New).
- [ ] T3: Intercept builtin + method-selector calls in `evalCallMulti`.
- [ ] T4: Implement `evalBuiltin` (len/append/make/panic) + helpers.
- [ ] T5: Implement `callMethod` + receiver helpers (copy for value receiver).
- [ ] T6: Add `builtins_test.go` covering all acceptance criteria.
- [ ] T7: Run verifyCommands (build, vet, test) green.

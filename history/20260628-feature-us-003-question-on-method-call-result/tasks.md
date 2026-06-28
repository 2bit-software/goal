# Tasks — US-003

- [ ] T1 (analyze): add `Return FuncSig` to `analyze.Method`; add
  `methodResultSig`; wire into `methodFrom`. (foundation)
- [ ] T2 (analyze): `ResolveCallee` returns `m.Return`; resolve interface
  receivers via `t.Interfaces`. (depends T1)
- [ ] T3 (sema): add `Return FuncSig` to `sema.Method`; populate in
  `resolveMethod` and `resolveInterface`. (foundation)
- [ ] T4 (backend): add `emitter.recvTypes`; build/save/restore in `funcDecl`;
  extend `calleeSig` to resolve receiver-bound method selectors. (depends T3)
- [ ] T5 (tests): checker test (accept Result method `?`, reject non-error
  method `?`) + backend test (bind+propagate Result method `?`, single-var
  error-only method `?`). (depends T2, T4)
- [ ] T6: run verifyCommands (build, vet, test) green.

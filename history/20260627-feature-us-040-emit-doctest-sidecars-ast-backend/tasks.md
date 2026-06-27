# Tasks — US-040

- [ ] T1: Add internal/backend/doctest.go with renderDoctests(f) + emitDoctests(f, info).
- [ ] T2: Wire goBackend.Emit (backend.go) to set Output.Test = emitDoctests(file, info).
- [ ] T3: Add backend_test.go tests: doctest tier (RunDoctest), exec tier
        (RunDoctestExec, -short skip), and an enum-lowering encoding assertion.
- [ ] T4: Run verifyCommands (go build/vet/test) + the 11-doctests doctest tier; green.

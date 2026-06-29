# Plan Audit: Coverage — US-001

| Spec criterion | Plan element |
|----------------|--------------|
| selfhost/backend holds backend as .goal mirroring internal/backend | Step 1 (copy 6 files) |
| BuildTranspiled compiles backend + deps | Step 3 COMPILE gate |
| Fixture-free backend tests pass against transpiled package | Steps 2+3 BEHAVIORAL gate |
| task check / build / fixpoint green | Step 4 |

No scope creep: the only "extra" element (splitting backend_test.go) is required to
satisfy AC #3's "minus those needing repo-relative fixtures". No CRITICAL/MAJOR.

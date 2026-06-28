# Plan Audit: Coverage — US-006

## Requirement → Plan trace

| Spec item | Plan element | Covered |
|-----------|--------------|---------|
| FR-1 pipeline consumes manifest | Step 2 (TestCorpusTranspile/Doctest via corpus runners) | Yes |
| FR-2 checker consumes manifest | Step 1 (TestCorpusCheck via corpus.RunCheck) | Yes |
| FR-3 no coverage regression | Manifest indexes the same 51+4+50 cases; runners exercise all | Yes |
| FR-4 fail loudly on empty | "loud zero guard" t.Fatalf in each test | Yes |
| AC grep pipeline | Remove feature globs + mustFormat | Yes |
| AC grep check | Remove testdata/check walk + helpers | Yes |
| AC build/vet/test | Step 3 verify | Yes |

No CRITICAL, no MAJOR findings. No scope creep — plan touches only the two named files.

## Assumptions
- Manifest case counts (51 transpile / 4 doctest / 50 check) are current and stable
  (verified by US-002/US-005 tests that still pass).
- Preserving `TestRegistryRuns` is desired (it is unrelated to the hardcoded paths).

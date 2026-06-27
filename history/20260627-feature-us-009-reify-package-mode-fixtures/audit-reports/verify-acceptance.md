# Verify — Acceptance Coverage

Full suite: `go build ./...`, `go vet ./...`, `go test ./... -count=1` — all green.

| Acceptance criterion | Evidence |
|----------------------|----------|
| Cross-file & foreign-derive sources exist as on-disk multi-file fixtures | `testdata/package/cross-file-demo/{math,types}.goal`, `testdata/package/foreign-derive/conv.goal` |
| Each fixture declares an import map | `pkg.json` `imports` field per fixture (empty for cross-file; extpkg for foreign) |
| Manifest indexes them as Mode=package cases | `corpus/manifest.json` has 2 `"mode":"package"` cases; `TestGenerateCounts` asserts package==2 |
| Single-file counts unchanged (51/50/4) | `TestGenerateCounts` asserts file-mode transpile 51, check 50, doctest 4 |
| Runner executes every Mode=package case and all pass | `TestPackageRunner` (RunPackage transpiles, validates Go, builds each package incl. wired foreign import) — passes |
| Build/vet/test green | confirmed |

No CRITICAL or MAJOR findings.

## Assumptions
- Package-mode pass bar is transpile + valid-Go + `go build ./...` (the
  foreign-derive output does compile, so the risk-mitigation fallback was not
  needed).

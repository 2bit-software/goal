# Verification — Acceptance Coverage

Gates run (all green):
- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (every package)
- `go test ./internal/corpus -run TestSemaFieldsRunner -v` — 9/9 subtests PASS

Acceptance criteria → evidence:

| Criterion | Evidence |
|-----------|----------|
| sema implements the check (Warnings/Errors) | internal/sema/fields.go `CheckFields`, wired into `Check` |
| single omission → ``omits required field `y``` | incomplete_single (`// want`) PASS |
| multi omission → ``omits required fields `email`, `admin``` | incomplete_struct (`// want`) PASS |
| variant omission → ``omits required field `h``` | variant_incomplete (`// want`) PASS |
| unresolved type → Warning `field-completeness deferred`, no Error | deferred_unknown_type (`// want`) PASS |
| `...defaults` / `...derive` → no Error | defaults_optout, derive_spread_optout PASS |
| complete + tagged struct + data-less variant → no Error | complete, tagged_struct, variant_incomplete (Dot/Circle) PASS |
| match-arm binding → no Error | match_binding_arm PASS (VariantPattern, not VariantLit) |
| every 08-no-zero-value case via corpus runner | TestSemaFieldsRunner drives all 9, t.Fatalf on zero |

Every acceptance criterion maps to a passing golden case.

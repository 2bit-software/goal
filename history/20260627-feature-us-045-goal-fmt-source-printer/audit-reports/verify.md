# Verify — goal fmt source printer (US-045)

## Project verify gates (prd.json verifyCommands)

- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages, including the new
  `internal/goalfmt` and `cmd/goal` bootstrap-golden test)

## Acceptance criteria

- AC-1 (corpus-wide idempotency): `TestIdempotentOverCorpus` asserts
  `Source(Source(src)) == Source(src)` for every unique `.goal` input in the
  corpus manifest — PASS.
- AC-2 (comments retained): `TestPreservesComments` asserts every `//`/`///`
  comment in a sample survives formatting — PASS.
- AC-3 (error path): `TestRejectsUnparseable` asserts malformed source returns an
  error; manual `goal fmt nonexistent.goal` reports a non-zero failure — PASS.
- AC-4 (meaning preserved): the idempotency test re-runs `Source` (which re-parses)
  so formatted corpus output must still parse — PASS.

## Manual smoke

- `goal fmt features/12-derive-convert/examples/to_storage.goal` is a byte-for-byte
  no-op on already-formatted source (idempotent), and a mis-indented sample is
  normalized to tab indentation.

No CRITICAL or MAJOR findings. Feature works as specified.

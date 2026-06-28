# Cleanup notes

Deleted:
- internal/check/ (entire package incl. check_test.go, foreign_test.go, testdata/extpkg)
- internal/corpus/check_runner_test.go (drove the deleted check.Analyze)
- internal/corpus/parity_test.go (US-003 legacy-vs-sema gate — moot once legacy gone)

Repointed consumers onto sema / token (see scope.md):
- cmd/goalc, internal/guide, internal/lsp (4 src + 4 tests), internal/typecheck
  (4 src + 2 tests), internal/corpus (check_runner.go, sema_checker.go + 4 test callers).

Follow-on fixups required by the deletion:
- internal/lsp/testdata/extpkg/types.go (new): minimal self-contained foreign
  fixture replacing the deleted internal/check/testdata/extpkg, so the LSP
  foreign-derive test no longer depends on the deleted dir (or on analyze's, which
  US-010 deletes). TestForeignDeriveResolves repointed to testdata/extpkg.
- internal/guide/guide_test.go scanLiveCodes now scans ../sema (was ../check).
- internal/guide/catalog.go: removed two legacy-only codes the AST checker does not
  emit — `question-not-statement` (now enforced structurally by the parser) and
  `unresolved-result-use` (superseded by `unresolved-result-discard`).
- AI-KNOWLEDGE-BOOTSTRAP.md regenerated (the two catalog codes dropped from the doc).

Verification: `task check` (go vet + full suite) and `task build` both green.
Acceptance criteria: internal/check removed; no live import outside attic/_cut;
corpus repointed to sema and CheckerFunc removed.

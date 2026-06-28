# Research Findings — US-003 differential parity

## Method

Wrote a throwaway probe test in `internal/corpus` that, for every `KindCheck`
case in the committed manifest, ran both `check.Analyze` (legacy lexical) and
`SemaCheck` (AST) and diffed findings by (file, line, feature, code, severity).

## Divergences discovered (the complete set over the corpus)

### A. AST fires Error where legacy deferred (Warning) — documented improvements

Three `12-derive-convert` bodyless-derive cases. The legacy checker reads the
target type name lexically and swallows the trailing `// want` comment into the
type name, so it cannot resolve the (in-file) struct and defers with
`unresolved-derive-type` (warning). The AST checker resolves the struct and
fires the real Error:

| file | line | sema (Error) | legacy (Warning) |
|------|------|--------------|------------------|
| 12-derive-convert/fallible_in_total.goal | 24 | fallible-in-total-derive | unresolved-derive-type |
| 12-derive-convert/unbridged_field.goal | 19 | unbridged-field | unresolved-derive-type |
| 12-derive-convert/unsourced_field.goal | 18 | unsourced-field | unresolved-derive-type |

The existing `// want` markers already contain the sema Error message substrings
(they happen to also be matched by the legacy warning because legacy swallowed
the same comment text), so the markers already express the sema behavior — no
edit required, only DECISIONS.md documentation.

### B. AST surfaces an extra deferral Warning legacy omits

| file | line | sema (Warning) | legacy |
|------|------|----------------|--------|
| 06-error-e/defer_err_value.goal | 16 | unresolved-err-value | (none) |

`Result.Err(e)` returns a bound variable; sema surfaces a located
`unresolved-err-value` deferral, legacy stays silent. Both are non-rejecting
deferrals (warnings may go unclaimed by `// want` markers), so no marker change
is needed; recorded as a documented divergence.

## Conclusion

Four documented divergences total. All other corpus check cases agree exactly on
(file, line, feature, code, severity). The parity gate will subtract these four
(as an explicit, DECISIONS.md-backed allowlist) and require the remainder to be
identical, failing on any new undocumented divergence AND on any stale allowlist
entry that no longer reproduces.

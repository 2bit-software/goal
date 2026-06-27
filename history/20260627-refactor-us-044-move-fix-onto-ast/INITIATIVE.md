# US-044 Move goal fix onto the AST

**Type**: refactor
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop constraint)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Migrate internal/fix so it consumes the parsed goal AST (parser.ParseFile +
sema.Resolve) to locate idiomatize candidates, instead of re-lexing with
scan.Lex and rebuilding facts with analyze.Build. It must emit the same byte
rewrites (Result-signature conversion, `?` propagation collapse, Option
nil-check collapse) and the same Suggest/Warn/Skip reports (switch-over-enum,
call-site). The existing fix test suite must pass unchanged in observable
behavior.

# Implementation Plan — US-009 Port sema package to goal

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `selfhost/sema/analyze.goal` | verbatim copy of internal/sema/analyze.go |
| `selfhost/sema/assert.goal` | verbatim copy of internal/sema/assert.go |
| `selfhost/sema/check.goal` | verbatim copy of internal/sema/check.go |
| `selfhost/sema/convert.goal` | verbatim copy of internal/sema/convert.go |
| `selfhost/sema/fields.goal` | verbatim copy of internal/sema/fields.go |
| `selfhost/sema/foreign.goal` | verbatim copy of internal/sema/foreign.go (go/parser, go/format, go/types pass through) |
| `selfhost/sema/implements.goal` | verbatim copy of internal/sema/implements.go |
| `selfhost/sema/mustuse.goal` | verbatim copy of internal/sema/mustuse.go |
| `selfhost/sema/package.goal` | verbatim copy of internal/sema/package.go |
| `selfhost/sema/question.goal` | verbatim copy of internal/sema/question.go |
| `selfhost/sema/resolve.goal` | verbatim copy of internal/sema/resolve.go |
| `selfhost/sema/sema.goal` | verbatim copy of internal/sema/sema.go |

### Modified Files
| File | Changes |
|------|---------|
| `internal/selfhost/port_test.go` | add TestPortedSemaPackage (compile + behavioral gates) |
| `prd.json` | set US-009 passes:true |
| `progress.txt` | append US-009 entry |

## Package Structure

```
selfhost/sema/
  analyze.goal assert.goal check.goal convert.goal fields.goal
  foreign.goal implements.goal mustuse.goal package.goal
  question.goal resolve.goal sema.goal
```

Test (.go, dropped from .goal copy): no reflection/dump file exists in sema, so
all 12 non-test sources port; only the *_test.go files stay in internal/sema.

## Dependency Graph

1. selfhost/token, selfhost/ast, selfhost/parser (already ported — US-005/007/008)
2. selfhost/sema (this story) — imports token, ast, parser; foreign go/* pass through
3. internal/selfhost/port_test.go TestPortedSemaPackage — exercises the gates

## Interface Contracts

Reuses existing harness, unchanged:

```go
func BuildTranspiled(layout map[string]*project.Package) error
func BuildAndTest(relDir string, pkg *project.Package, testFiles []string, deps map[string]*project.Package) error
func project.Discover(dir string) ([]*project.Package, error)
```

Layout/deps keys are module-relative dirs: internal/token, internal/ast,
internal/parser, internal/sema. (lexer is NOT a direct sema import — verify
sema's imports; include lexer only if parser transpilation needs it in the
layout. parser depends on lexer, so the transpiled parser .go imports
goal/internal/lexer; the layout therefore must also carry lexer.)

## Integration Points

internal/selfhost/port_test.go gains TestPortedSemaPackage following
TestPortedParserPackage exactly: Discover each selfhost/<pkg>, assert names,
BuildTranspiled over the full layout {token,lexer,ast,parser,sema},
BuildAndTest("internal/sema", semaPkg, <self-contained test files>, deps=
{token,lexer,ast,parser}).

## Testing Strategy

- COMPILE gate (FR-2): BuildTranspiled over the layout.
- BEHAVIORAL gate (FR-3): BuildAndTest with the self-contained internal/sema
  *_test.go files. Exclude foreign_test.go and package_test.go (depend on
  internal/sema/testdata/extpkg, absent from the temp module). Candidate
  inclusion set: assert, check, convert, implements, mustuse, question,
  resolve, sema (each defines its own analyzeX helper; hasDiag is only in the
  excluded package_test.go). Confirm empirically — narrow if a copied test
  references a symbol outside the included set.
- Project gates: task check, task build, task fixpoint.

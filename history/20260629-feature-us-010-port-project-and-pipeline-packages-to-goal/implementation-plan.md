# Implementation Plan — US-010 Port project and pipeline packages to goal

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `selfhost/project/project.goal` | Verbatim goal port of `internal/project/project.go` (package discovery). |
| `selfhost/pipeline/pipeline.goal` | Verbatim goal port of `internal/pipeline/pipeline.go` (output types). |
| `selfhost/pipeline/sourcemap.goal` | Verbatim goal port of `internal/pipeline/sourcemap.go` (//line source map). |

### Modified Files
| File | Changes |
|------|---------|
| `internal/selfhost/port_test.go` | Add `TestPortedProjectPackage` and `TestPortedPipelinePackage` running BuildTranspiled (compile gate) + BuildAndTest (behavioral gate). |
| `prd.json` | Set US-010 `passes: true` (after green). |
| `progress.txt` | Append US-010 entry. |

No changes to `internal/selfhost/selfhost.go` — the existing multi-entry
BuildTranspiled and deps-aware BuildAndTest already cover this story (same shape
as the sema port).

## Package Structure

```
selfhost/
  token/   token.goal           (US-005, existing)
  lexer/   lexer.goal           (US-006, existing)
  ast/     *.goal               (US-007, existing)
  parser/  *.goal               (US-008, existing)
  sema/    *.goal               (US-009, existing)
  project/  project.goal        (NEW)
  pipeline/ pipeline.goal       (NEW)
            sourcemap.goal      (NEW)
```

## Dependency Graph

1. Existing ported deps: token, lexer, ast, parser (already under selfhost/).
2. `selfhost/project` — imports parser (-> lexer/ast/token transitively).
3. `selfhost/pipeline` — imports ast, parser, token (parser -> lexer).
4. port_test additions — depend on 2 and 3 plus the existing harness.

## Interface Contracts

No new interfaces. The ports are verbatim copies; public API is identical to
`internal/project` and `internal/pipeline`. Harness contracts reused as-is:

```go
selfhost.BuildTranspiled(layout map[string]*project.Package) error
selfhost.BuildAndTest(relDir string, pkg *project.Package,
    testFiles []string, deps map[string]*project.Package) error
```

Layout/deps for both new packages (keyed by module-relative dir):
`internal/token`, `internal/lexer`, `internal/ast`, `internal/parser`, plus the
package under test (`internal/project` or `internal/pipeline`).

## Integration Points

- `selfhost/<pkg>/*.goal` is discovered by `project.Discover` (used by both the
  port_test and `task fixpoint`).
- port_test functions live in `internal/selfhost/port_test.go` (package
  `selfhost_test`), mirroring `TestPortedSemaPackage`.

## Testing Strategy

- COMPILE gate: BuildTranspiled over the full layout (deps + package under test).
- BEHAVIORAL gate: BuildAndTest copying only self-contained existing tests:
  - project: `../project/project_test.go` (stdlib-only, temp-dir based).
  - pipeline: `../pipeline/sourcemap_test.go` (white-box, strings/testing only).
  - EXCLUDE `../pipeline/pipeline_test.go` (imports backend + corpus, reads the
    repo-relative corpus manifest — unfit for the throwaway temp module).
- Project-wide gates: `task check`, `task build`, `task fixpoint`.

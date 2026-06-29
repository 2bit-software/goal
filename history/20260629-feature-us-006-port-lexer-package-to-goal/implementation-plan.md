# Implementation Plan — US-006 Port lexer package to goal

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `selfhost/lexer/lexer.goal` | Verbatim copy of `internal/lexer/lexer.go` as goal source — the goal-native tokenizer importing the ported `goal/internal/token`. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/selfhost/selfhost.go` | Extend `BuildAndTest` to also transpile a set of dependency packages into the temp module, so a ported package with an in-module import (lexer -> token) compiles and tests. |
| `internal/selfhost/port_test.go` | Add `TestPortedLexerPackage`: discover selfhost/token and selfhost/lexer, run `BuildTranspiled` (compile gate) over both, and `BuildAndTest` (behavioral gate) on lexer with token as a dep, against `../lexer/lexer_test.go`. |
| `prd.json` | Set US-006 `passes: true` (after green). |
| `progress.txt` | Append US-006 entry. |

## Package Structure

```
selfhost/
  main.goal
  token/
    token.goal      (US-005)
  lexer/
    lexer.goal      (NEW — imports goal/internal/token)
internal/selfhost/
  selfhost.go       (MODIFIED — BuildAndTest gains deps)
  port_test.go      (MODIFIED — adds lexer test)
```

## Dependency Graph

1. `selfhost/lexer/lexer.goal` (copy of source; depends only on already-ported token)
2. `internal/selfhost/selfhost.go` BuildAndTest extension (independent harness change)
3. `internal/selfhost/port_test.go` lexer test (depends on 1 and 2)

## Interface Contracts

Extend BuildAndTest to carry dependency packages:

```go
// deps are additional packages (keyed by module-relative dir, e.g.
// "internal/token") transpiled into the same temp module so the package under
// test can import them. Pass nil when the package has no in-module deps.
func BuildAndTest(relDir string, pkg *project.Package, testFiles []string, deps map[string]*project.Package) error
```

The existing token call updates to pass `nil` for deps.

The lexer test builds the layout once and reuses it:

```go
layout := map[string]*project.Package{
    "internal/token": tokenPkg,
    "internal/lexer": lexerPkg,
}
selfhost.BuildTranspiled(layout)                               // compile gate
selfhost.BuildAndTest("internal/lexer", lexerPkg,
    []string{"../lexer/lexer_test.go"},
    map[string]*project.Package{"internal/token": tokenPkg})   // behavioral gate
```

## Integration Points

- `project.Discover(dir)` reads a selfhost package dir into a `*project.Package`.
- `selfhost.BuildTranspiled` / `selfhost.BuildAndTest` are the two reused gates
  from US-002/US-005; `writePackage` already handles per-package emission.
- `task fixpoint` auto-discovers the new selfhost/lexer dir (no harness change).

## Testing Strategy

- Compile gate: `BuildTranspiled` over {token, lexer} must `go build ./...` clean.
- Behavioral gate: `BuildAndTest` copies the trusted `internal/lexer/lexer_test.go`
  beside the transpiled lexer and runs `go test ./internal/lexer`.
- Project gates: `task check`, `task build`, and `task fixpoint` must stay green.

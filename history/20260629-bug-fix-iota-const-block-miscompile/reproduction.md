# Reproduction

## Failing tests

- `internal/parser/parser_test.go` :: `TestParseIotaConstBlockBoundaries`
- `internal/backend/backend_test.go` :: `TestASTEngineEmitsIotaConstBlock`

## How to run

```
go test ./internal/parser/ -run TestParseIotaConstBlockBoundaries -count=1
go test ./internal/backend/ -run TestASTEngineEmitsIotaConstBlock -count=1
```

## Failure output (before fix)

Parser: `const group has 2 specs, want 3 (Red, Green, Blue)`

Backend:
```
Green and Blue collapsed into a name+type spec:
const (
	Red   Color = iota
	Green Blue
)
```

## Why this captures the bug

The parser produces 2 specs instead of 3 because parseValueSpec consumes the
bare `Blue` on the next line as the type of `Green`. The backend faithfully
emits that wrong AST as `Green Blue`, dropping Blue's iota value. Both layers
are pinned so the parser fix is proven and the round-trip emission is guarded.

## Assumptions

- A grouped const/var spec occupies a single source line (idiomatic Go; the
  language inserts a semicolon at the newline). The fix uses token line numbers
  to reconstruct that boundary. Multi-line single specs after `=` (value lists)
  are unaffected — the ambiguity is only the optional type before `=`.
- The backend `spec`/`genDecl` emitters already render bare-name specs
  correctly, so no backend change is required — only the parser.

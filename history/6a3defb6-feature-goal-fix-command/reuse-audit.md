---
status: complete
updated: 2026-06-25
---

# Reuse Audit: `goal fix`

Goal is zero-dependency and already lexical-rewrite-shaped, so most of `fix` is reuse, not
new code. Findings per planned component:

| Planned item | Verdict | Existing code | Action |
|---|---|---|---|
| Token lexing | **DUPLICATE** | `scan.Lex` (`internal/scan/scan.go:30`) | Use directly |
| Splice edits | **DUPLICATE** | `scan.Splice`, `scan.Replacement` (`scan.go:47,55`) | Use directly |
| Balanced-delimiter / ident helpers | **DUPLICATE** | `scan.MatchParen/MatchBrace/LeadIdent/SplitAssign` | Use directly |
| Signature/mode/enum tables | **DUPLICATE** | `analyze.Build`, `BuildPackage`, `Tables`, `FuncSig`, `Mode*`, `Enum` (`analyze.go:149,179`) | Use directly |
| File discovery | **DUPLICATE** | `project.Discover` (`internal/project/project.go:53`) | Use directly |
| CLI flag parsing | **OVERLAP (Extend)** | `parseFlags` (`cmd/goal/main.go:130`) handles only `--emit` | Add a small `parseFixFlags` modeled on it (needs `-inplace`, file-or-dir path); do not overload `parseFlags` |
| Write-back to disk | **OVERLAP (Reuse pattern)** | `emitFiles` (`main.go:436`) = `os.WriteFile`+log | Mirror the pattern for `-inplace`; emitFiles itself is shaped for transpiled Go output, so a thin `writeFixed` is clearer than reusing it verbatim |
| Function-span + enclosing-mode lookup | **DUPLICATE but UNEXPORTED** | `funcSpans`, `sigAt` (`internal/pass/pass.go:44,55`) | **Refactor**: relocate to `internal/analyze` as exported `FuncSpans`/`SigAt` (both already operate on `*analyze.Tables`/`FuncSig`); repoint `internal/pass`. `fix` then imports them. Avoids a fix→pass dependency and avoids reimplementation. |
| Zero-value literal for DR-2 matching | **DUPLICATE but UNEXPORTED** | `zeroLit` (`internal/pass/defaults.go:113`) | **Refactor**: relocate to `internal/analyze` as exported `ZeroLit(typ, decls, depth)` (depends only on `map[string]string` decls); repoint `internal/pass`. `fix` reuses it to recognize bare-propagation zeros. |
| `?`/Result/Option lowering templates | **RELATED (inverse reference)** | `pass.Question` (`question.go:48-61`), `pass.Result` (`result.go`) | Not called by `fix`, but they are the *spec of the inverse*: `fix` output, when lowered by these, must reproduce the input. Used as the golden-test oracle, not imported. |
| The rewrite rules themselves (propagate, result-sig, callsite, match) | **NONE** | No existing fix/lint/migrate code (confirmed: no `goal fix`/`lint`/`migrate`) | Create new in `internal/fix` |

## Net effect on the plan

- One small **enabling refactor** lands first (Phase A): export `FuncSpans`/`SigAt`/`ZeroLit`
  + name constants from `internal/analyze`, repoint `internal/pass`. Pure move, behavior
  unchanged, existing pass tests are the guard.
- Everything else under `internal/fix` is genuinely new (the rule logic) built on reused
  primitives. No duplication of lexing, splicing, table-building, or discovery.
- The transpiler's existing `?`/Result passes become the **test oracle** (`lower(fix(x))`
  must equal `lower(x)`), giving correctness for free without new verification machinery.

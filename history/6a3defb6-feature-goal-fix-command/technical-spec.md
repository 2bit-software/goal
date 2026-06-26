# Technical Spec: `goal fix`

## Architecture

`fix` is a new lexical source-to-source rewriter — the **inverse** of the lowering passes —
built on the existing `scan`/`analyze`/`project` primitives. No AST, no new infrastructure.

```
cmd/goal/main.go
  guideCommands += {Name:"fix", ...}
  run(): case "fix": parseFixFlags → cmdFix
  cmdFix(paths, inplace, out, errOut):
     project.Discover → for each Package:
        tables := analyze.BuildPackage(srcs)      // package-wide signatures/enums
        for each File: newSrc, reports := fix.File(file.Src, tables)
        default:  print newSrc to out
        -inplace: if newSrc != src → os.WriteFile, log "fixed <path>"
        reports → errOut (suggestions, skips, exported warnings)

internal/fix/
  fix.go        // Fix orchestrator + types (Change, Report)
  propagate.go  // collapse bare propagation → `?`   (DR-1,2,5,6)
  resultsig.go  // (T,error) → Result[T,error] + return rewrite (DR-3)
  callsite.go   // update call sites of converted fns (DR-7,9)
  match.go      // switch over in-file enum → match  (DR-8)  [P3]
  fix_test.go, *_test.go, testdata/fix/<rule>/{before,after}.goal
```

### Enabling refactor (Phase A, from reuse audit)

Relocate from `internal/pass` to `internal/analyze`, exported, behavior unchanged:
- `FuncSpans(toks, *Tables) []FuncSpan` and `SigAt(spans, off) (FuncSig, bool)`
- `ZeroLit(typ string, decls map[string]string, depth int) string`
- name constants if needed (`fix` generates its own `?`, so it mostly needs `ZeroLit` for
  *matching*, not the `__goal_*` names).
Repoint `internal/pass` to the new homes; existing pass tests guard the move.

## Core types

```go
package fix

type Change struct { File string; Line int; Rule string; Before, After string }
type Report struct { File string; Line int; Level Level; Rule, Msg string } // Level: Suggest|Warn|Skip

// File applies all enabled fixers to one file's source to a fixed point.
// Returns the rewritten source (== src if nothing changed), the changes applied,
// and reports for candidates not safely fixed.
func File(src string, t *analyze.Tables) (out string, changes []Change, reports []Report, err error)
```

### Orchestration: fixed-point loop (FR-011)

```
out := src
for i := 0; i < maxIters; i++ {          // maxIters small, e.g. 5; guards oscillation
    toks := scan.Lex(out)
    spans := analyze.FuncSpans(toks, t)
    reps, ch, rep := runEnabledRules(out, toks, spans, t)
    if len(reps) == 0 { break }
    out = scan.Splice(out, 0, len(out), reps)
    changes, reports = append(changes, ch...), append(reports, rep...)
    // tables are rebuilt next iter only if a signature changed (resultsig sets a flag)
}
```
A signature conversion in iteration *i* re-marks a function `ModeResult`, so iteration *i+1*
re-lexes and the now-legal body propagations collapse. This naturally orders the
interdependent rules and yields the FR-011 fixed point. `t` is rebuilt from `out` between
iterations when any rule reports a signature change (so `FuncSpans` sees the new mode).

## Rule designs

### propagate.go — collapse bare propagation → `?` (DR-1/2/5/6)

For each `if` token whose enclosing span (`SigAt`) is `ModeResult`/`ModeOption`:
1. Match the condition exactly: Result → `<e> != nil`; Option → `<o> == nil`
   (`scan` token walk between `if` and `{`).
2. Match the body exactly one statement: Result → `return <ops...>, <e>` or
   `return Result.Err(<e>)`; Option → `return Option.None` / `return <zero>`.
   Bodies with extra statements, wrapping, logging → skip (Report Suggest).
3. Find the binding statement immediately preceding the `if` (no statement between):
   `SplitAssign` on that line; LHS must bind `<e>` (Result) / `<o>` (Option).
4. DR-2: each non-error operand in `<ops...>` equals `analyze.ZeroLit(retType_i, decls)`
   (whitespace-trimmed textual compare). Mismatch → skip.
5. DR-5: scan the raw `src[bindStart:ifEnd]` span for `//` or `/*`; if present → skip
   (would drop a comment).
6. DR-6: if the binding RHS spans >1 line → skip.
7. Emit `Replacement{bindStart, ifEnd, "<name> := <rhs>?"}` (keep) or
   `"_ := <rhs>?"` (discard). Option also rewrites later `*o` uses → `<name>` within scope
   (DR-4); if `o` escapes → skip the whole block.

### resultsig.go — `(T,error)` → `Result[T,error]` (DR-3)

For each function span with `Mode==ModeNone` whose signature's last result is `error` and
which has exactly one non-error result `T`:
1. Whole-function check: **every** `return` in the body is either a bare propagation
   (DR-1 shape, ops are zeros) or a single-value success `return v, nil`. Any decorated /
   non-conforming return → do not convert; Report Skip with `file:line`.
2. Rewrite signature `(T, error)` → `Result[T, error]` (reuse the inverse of
   `rewriteResultSignature`’s span logic).
3. Rewrite returns: `return v, nil` → `return Result.Ok(v)`; decorated error returns that
   survived (only if step 1 allowed them — it doesn't, so none) ; bare props are left for
   propagate.go to collapse next iteration.
4. Set `signatureChanged=true` so the orchestrator rebuilds tables.
5. If the function is exported → Report Warn (DR-9 / FR-008).

### callsite.go — update call sites of converted functions (DR-7)

After a function `F` converts (track the set of converted names this run):
1. Find call expressions `F(...)` in the package set bound as `x, err := F(...)` followed by
   a bare-propagation `if`.
2. If the enclosing function (`SigAt`) is `ModeResult`/`ModeOption` and the call is not in a
   closure/`defer`/`go` body → it will be collapsed by propagate.go on the next iteration
   (no special code needed — it's the same shape). 
3. If the enclosing function is **not** Result/Option, or the call is in a closure/defer/go →
   leave unchanged, Report Suggest "manual follow-up: caller is not Result/Option-returning".

This means callsite.go is mostly a **reporter**: the actual `?` rewrite at legal sites falls
out of propagate.go once `F` is `Result`. Its job is to enumerate the *illegal* sites the
fixed-point loop will not touch, so the user is warned (FR-007).

### match.go — switch over in-file enum → match (DR-8) [P3]

For each `switch <scrut> { ... }` where `scan.LeadIdent` of `<scrut>` or the case-label
qualifier names a key in `t.Enums`: rewrite `switch`→`match`, `case Enum.V:`→`Enum.V =>`,
preserve arm bodies, drop `break`. Body-local, no contract change.

## CLI

- `parseFixFlags(args) (paths []string, inplace bool, err error)`: like `parseFlags` but
  recognizes `-inplace`; path may be a file or dir (default `.`).
- `cmdFix`: for a single `.goal` file path, fix just that file; for a dir, `project.Discover`.
  Default → stdout; `-inplace` → write changed files only, log `fixed <path>` to stdout.
- Streams: rewritten source / write logs → `out`; reports → `errOut` (FR-013).
- Exit non-zero only on operational error (FR-014); reports/warnings do not fail.
- Register in `guideCommands` with a usage line; appears in top-level help.

## Testing

- **Golden** (`internal/fix/testdata/fix/<rule>/{before,after}.goal`): table test lexes
  `before`, runs `fix.File`, asserts `== after`; then feeds `after` back and asserts
  no-change (FR-011); then asserts byte-equality outside changed spans (FR-010).
- **Oracle**: for each `after.goal`, run it through the transpile pipeline and assert it
  lowers without error, and (where feasible) that `lower(after) == lower(before)` —
  the inverse-of-lowering correctness from the spike.
- **Negative fixtures**: wrapped error, decorated return, multi-value signature, comment in
  block, multi-line RHS, escaping Option pointer, caller-not-Result → assert *no* change +
  a Report.
- **CLI** (`cmd/goal/main_test.go`, `goalModule`+`run`): default prints & writes nothing;
  `-inplace` writes only changed files; exported-warning on stderr; file & dir paths; exit
  codes. stdlib `testing` only (no testify).

## Risks & mitigations

- *Lexical matching brittleness* → keep each rule's match strict; when unsure, **skip +
  report** (FR-012). The oracle test catches any rewrite that produces invalid goal.
- *Call-site ripple beyond package* → exported warning + `goal check` as the safety net;
  documented as best-effort.
- *Comment loss* (scanner skips comments) → DR-5 raw-source comment scan before any
  multi-statement removal.

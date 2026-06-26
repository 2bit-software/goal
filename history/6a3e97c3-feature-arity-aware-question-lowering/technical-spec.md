# Technical Spec: Arity-Aware `?` Lowering

Implements `spec.md`. Design is grounded in `research.md`; all uncertain areas were validated
by reading existing analogues (no spikes needed — see Spike Decision below).

## Overview

The open-E discard `?` lowering in `internal/pass/question.go` emits a fixed `_, ` prefix. We
make it emit `arity − 1` blank identifiers, where `arity` is the callee's resolved return
count. Arity is sourced from `analyze.Tables.FuncSignatures` — populated for in-file functions
by `analyzeSig`, and for imported package-level functions by an extension to
`internal/analyze/foreign.go` (package mode only). When arity is unresolved, the existing
two-value form is emitted unchanged (no regression).

## Changes by file

### 1. `internal/analyze/analyze.go`

**`FuncSig` gains a field** (struct at `analyze.go:36`):

```go
type FuncSig struct {
    Name  string
    Mode  Mode
    T     string
    E     string
    Arity int // return-value count at ?-lowering time; 0 when unknown/void
}
```

**`analyzeSig` (`analyze.go:635`) computes `Arity` for every function.** Restructure so arity
is computed *before* the early-return guard that protects Result/Option bracket indexing —
otherwise a single-return `func f() error` (where `pc+2 == BodyOpen`) returns early with
arity 0.

⚠️ **Do NOT slice the return clause from `f.ParamsClose`.** `scan.ParamsClose` (scan.go:152)
scans *back* from the body and returns the first depth-0 `)`, which for a **parenthesized**
return type `(int, error)` is the *return type's own* close paren — so `src[toks[pc].End:body]`
is empty and arity counts 0 for every parenthesized multi-return. Instead derive the true
parameter-list close from the name token:

1. `paramOpen := f.NameTok + 1`; require `toks[paramOpen].Text == "("` (else bail, arity 0).
   `paramClose := scan.MatchParen(toks, paramOpen)` — the actual param-list `)`, correct for
   bare, bracketed, and parenthesized return types and for methods (NameTok is post-receiver).
2. `retText := strings.TrimSpace(src[toks[paramClose].End:toks[f.BodyOpen].Start])`;
   `sig.Arity = countReturns(retText)`.
3. Run the existing Result/Option detection (still keyed off `pc := f.ParamsClose`, which is
   correct for the bracketed `Result[…]`/`Option[…]` cases it handles; guarded by
   `pc+2 < f.BodyOpen`).
4. Override by mode: `ModeResult → 2`, `ModeOption`/`ModeResultClosed → 1`. `ModeNone` keeps
   the counted value.

Keep the existing `pc < 0` guard before any `toks[pc]` index (`BodyOpen` is always ≥ 0 —
`ScanFuncs` drops bodyless funcs — so the panic risk is `pc`/`paramClose < 0`, not `BodyOpen`).
**Write the S1 arity unit test first** — it is what catches this slice bug.

**New helper `countReturns`** (reuses the existing `splitTopLevel`):

```go
// countReturns reports how many values a function's return clause yields:
// "" → 0, a single bare type → 1, a parenthesized list → its top-level-comma entry count
// ("(a, b int, err error)" → 3, "(int, error)" → 2).
func countReturns(ret string) int {
    ret = strings.TrimSpace(ret)
    if ret == "" { return 0 }
    if strings.HasPrefix(ret, "(") && strings.HasSuffix(ret, ")") {
        inner := strings.TrimSpace(ret[1 : len(ret)-1])
        if inner == "" { return 0 }
        return len(splitTopLevel(inner))
    }
    return 1
}
```

`splitTopLevel` already respects `()[]{}` depth, so `Result[T, error]` counts as 1 (overridden
to 2 by mode) and `map[K]V` as 1.

### 2. `internal/analyze/foreign.go`

**Parse package-level foreign functions.** Extend the package read so it returns funcs
alongside structs. Refactor `foreignStructs(dir, alias)` →
`foreignDecls(dir, alias) (alias string, structs map[string][]Field, funcs map[string]int, err error)`
(single parse pass; update the one caller). The funcs map is keyed `alias.Func → arity`.

In the decl loop, alongside the existing `*ast.GenDecl` struct handling, add:

```go
if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Name.IsExported() {
    funcs[alias+"."+fd.Name.Name] = resultArity(fd.Type)
}
```

```go
// resultArity counts a func type's return values (an unnamed result is one; a named group
// "(a, b int)" is len(Names)).
func resultArity(ft *ast.FuncType) int {
    if ft.Results == nil { return 0 }
    n := 0
    for _, f := range ft.Results.List {
        if len(f.Names) == 0 { n++ } else { n += len(f.Names) }
    }
    return n
}
```

Methods (`fd.Recv != nil`) are skipped — receiver-typed `?` callees are out of scope.

**Merge funcs into the tables** in `EnrichForeign`'s load loop (`foreign.go:91`):

```go
for name, arity := range funcs {
    t.FuncSignatures[name] = FuncSig{Arity: arity} // Mode left ModeNone (FR-010)
}
```

Keys contain a `.`; in-file keys are bare names → no collision. (`FuncSig.Name` is never read
for foreign entries — every consumer keys by the map key or reads `Mode`/`T`/`E` — so leave it
empty rather than mis-deriving it with `lastSegment`, which splits on `/` not `.`.)

**Collect `?`-callee aliases so the right imports load.** `EnrichForeign` early-returns when
`needed` is empty, and `neededAliases` only scans `derive`/`from` type positions. Add a sibling
collector and union it in. There is no back-matching paren helper, so reuse `CalleeKey` on each
`?`'s rhs instead of walking tokens backward:

```go
// questionCalleeAliases returns the package qualifiers used at the head of a `?` callee
// (`alias.Func(...)?`), so an import referenced only by `?` is still parsed.
func questionCalleeAliases(srcs []string) map[string]bool {
    out := map[string]bool{}
    for _, src := range srcs {
        toks := scan.Lex(src)
        for q := range toks {
            if toks[q].Text != "?" { continue }
            lineStart := strings.LastIndexByte(src[:toks[q].Start], '\n') + 1
            _, rhs, _ := scan.SplitAssign(src[lineStart:toks[q].Start])
            if alias, _, ok := strings.Cut(scan.CalleeKey(rhs), "."); ok {
                out[alias] = true
            }
        }
    }
    return out
}
```

Union into `needed` in `EnrichForeign` (`needed[a] = true` for each). `CalleeKey` returns a
dotted key only for a qualified call, so a plain in-file `?` callee contributes nothing.

### 3. `internal/scan/scan.go`

**New `CalleeKey`** — turns an rhs into a `FuncSignatures` lookup key:

```go
// CalleeKey returns the lookup key for the function called at the head of expr:
// a package-qualified call "os.MkdirAll(p)" → "os.MkdirAll"; a plain call "doThing(x)" →
// "doThing"; anything that is not a simple (optionally-qualified) call → "".
func CalleeKey(expr string) string
```

Rule: leading identifier; if immediately followed by `.` + identifier, append `.ident`; then
stop at the first `[` or `(` (so a generic instantiation `f[T](x)` → `f`, resolving in-file).
Returns "" for an index/paren-led expr or a non-identifier head. A deeper chain
`pkg.Sub.Func(x)` yields `pkg.Sub` (absent from the tables → unknown → safe default). Reuses the
lexer for robustness against strings/whitespace.

**S2 test table** must include: `"os.MkdirAll(p)"`→`"os.MkdirAll"`, `"doThing(x)"`→`"doThing"`,
`"f.Close()"`→`"f.Close"`, `"f[T](x)"`→`"f"`, `"pkg.Sub.Func(x)"`→`"pkg.Sub"`, `"xs[0]"`→`""`,
`"(a+b)"`→`""`, leading-whitespace `"  os.Mkdir(p)"`→`"os.Mkdir"`.

Add `MatchParenBack` only if no existing helper finds the `(` matching a given `)`.

### 4. `internal/pass/question.go`

In the `ModeResult` branch (`question.go:50`), resolve arity and shape the guard:

```go
arity, known := calleeArity(t, rhs)
if discard {
    n := 2 // FR-005 default when unknown
    if known && arity >= 1 { n = arity }
    blanks := strings.Repeat("_, ", n-1)
    text = fmt.Sprintf("if %s%s := %s; %s != nil {\nreturn %s, %s\n}", blanks, errName, rhs, errName, okName, errName)
} else {
    if known && arity != 2 {
        return "", fmt.Errorf("`?` binds a value but %s returns %d value(s); write a bare `…?` to propagate only the error", scan.CalleeKey(rhs), arity)
    }
    text = fmt.Sprintf("%s, %s := %s\nif %s != nil {\nreturn %s, %s\n}", name, errName, rhs, errName, okName, errName)
}
```

```go
func calleeArity(t *analyze.Tables, rhs string) (int, bool) {
    key := scan.CalleeKey(rhs)
    if key == "" { return 0, false }
    sig, ok := t.FuncSignatures[key]
    return sig.Arity, ok
}
```

`strings.Repeat("_, ", n-1)` is guarded by `n >= 1` (the `arity >= 1` check), so it never goes
negative. `n == 1` → no blanks → `if __goal_err := rhs; …`. `n == 2` → `_, ` (unchanged). The
enclosing-function return stays `__goal_ok, __goal_err` (driven by the caller, not the callee).

Closed-E (`internal/pass/closed.go`) and Option are untouched.

## Spike Decision

No spikes. Every risk is covered by an existing analogue in-tree:
- foreign `func` parsing ≈ existing struct parsing in `foreignStructs` (same `go/ast` walk);
- arity reasoning ≈ `internal/fix/propagate.go` ("the call's sole output is the error");
- top-level return counting ≈ existing `splitTopLevel`.
The end-to-end compile risk is retired by the `go build ./...` package test (SC-001).

## Behavior preservation

- `qprop_discard.go.expected` (in-file arity-2 discard) — callee resolves to arity 2 → `_, `
  prefix → byte-for-byte unchanged.
- `qprop_result` / binding form — non-discard path unchanged for arity-2 callees.
- Single-file foreign discard — unresolved → default 2 → unchanged.

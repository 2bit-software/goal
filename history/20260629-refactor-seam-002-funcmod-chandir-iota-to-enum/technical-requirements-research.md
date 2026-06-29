# Technical requirements / research — SEAM-002

## Enabling capabilities (already landed)

- SEAM-CAP (fb92fa9): cross-package enum-match lowering in the backend
  (matchQualifier resolves `pkg.Enum.Variant` SelectorExpr; enumOf resolves
  imported enums).
- SEAM-CAP-2 (7279312): cross-`.goal`-package enum/sema-fact propagation during
  the real `goal build ./selfhost` bootstrap. `goalForeignDecls` reads sibling
  `.goal` sources and projects exported enums into `info.Enums`; `enumRef` lowers
  bare cross-package construction `pkg.Enum.Variant` -> `pkg.Enum(pkg.Enum_Variant{})`.

Proven forms (SEAM-CAP/CAP-2 fixtures): value-position cross-package match
(`return match m {...}`), statement-position match, and bare cross-package
variant construction. Qualified pattern is `pkg.Enum.Variant`.

## Definitions (selfhost/ast)

- `FuncMod` — selfhost/ast/goal_decl.goal:20 (`type FuncMod int` + iota block).
- `ChanDir` — selfhost/ast/ast.goal:672 (`type ChanDir int` + iota block).

## Consumers to convert

- selfhost/sema/question.goal:210 — `fn.Mod != ast.FuncPlain` (in a boolean).
- selfhost/sema/resolve.goal:218 — `d.Mod == ast.FuncFrom || d.Mod == ast.FuncDerive`.
- selfhost/sema/resolve.goal:458 — `switch x.Dir { ChanDir cases }` (value strings).
- selfhost/sema/convert.goal:34 — `fd.Mod != ast.FuncDerive` (in a boolean).
- selfhost/backend/emit.goal:361 — `switch d.Mod { FuncMod cases }` (control flow).
- selfhost/backend/emit.goal:2309 — `switch x.Dir { ChanDir cases }` (emit prefix).
- selfhost/ast/ast.goal:170 — `d.Mod != FuncPlain` (same-package boolean).
- selfhost/parser/parser.goal:226,229 — construction `ast.FuncFrom`/`ast.FuncDerive`.
- selfhost/parser/parser.goal:511,515,520 — construction `ast.SendRecv`/`RecvOnly`/`SendOnly`.
- selfhost/parser/parser.goal:365 — `&ast.FuncDecl{}` zero-value gap: enum zero is
  nil, not FuncPlain. Set `Mod: ast.FuncMod.FuncPlain` explicitly.

## Oracle-test divergence (the one real tension)

`internal/ast/ast_test.go` is the only port-gated test file referencing these
symbols. internal/ast (Go) stays iota; selfhost/ast (goal) becomes enum. The
shared file cannot compile against both. Resolution: relocate the FuncMod
assertions into a new internal-only `internal/ast/funcmod_test.go` (NOT in the
port-gate test list), leaving ast_test.go FuncMod/ChanDir-free so it compiles
against the enum-transpiled selfhost/ast. internal/ast itself keeps Go iota (it
is the bootstrap reference compiler — its AST need not mirror selfhost's repr).

goal_stmt_test.go also references the symbols but is NOT in the port-gate set
(only parser_test.go is) and tests internal/parser against internal/ast — both
Go iota — so it stays unchanged. sema/backend port-gate test files do not
reference these symbols. Corpus/parser-snapshot goldens test the unchanged Go
compiler, so they do not move.

## Conversion idiom

Replace `==`/`!=` boolean uses with a value-position `match` bound to a bool,
preserving short-circuit by splitting any `!ok ||` type-assert guard out first.
Replace value-switches with value-position `match`. Replace the funcDecl control
switch with `isDerive := match d.Mod {...}` + `if isDerive`.

## Per-site target forms (resolves audit findings)

Enum declarations (selfhost/ast), doc text preserved:

```
enum FuncMod {        // selfhost/ast/goal_decl.goal
    FuncPlain
    FuncFrom
    FuncDerive
}
enum ChanDir {        // selfhost/ast/ast.goal
    SendRecv
    SendOnly
    RecvOnly
}
```

Consumer conversions (target bodies):

- question.goal:210 — split the `!ok` guard out FIRST to keep short-circuit, then
  match-bound bool (a naive hoist of the match above the `!ok` check nil-derefs):
  ```
  fn, ok := d.(*ast.FuncDecl)
  if !ok { continue }
  plain := match fn.Mod {
      ast.FuncMod.FuncPlain  => true
      ast.FuncMod.FuncFrom   => false
      ast.FuncMod.FuncDerive => false
  }
  if !plain || fn.Recv != nil || fn.Name == nil || fn.Type == nil || fn.Body == nil { continue }
  ```
- resolve.goal:218 — tagless `switch {}` STAYS; only the inner `==` becomes a
  bool: `isConv := match d.Mod { ast.FuncMod.FuncFrom => true; ast.FuncMod.FuncDerive => true; ast.FuncMod.FuncPlain => false }` then `case isConv:`.
- convert.goal:34 — split `!ok`, then `isDerive := match fd.Mod { ast.FuncMod.FuncDerive => true; FuncFrom => false; FuncPlain => false }`; `if !isDerive { continue }`.
- ast.goal:170 (same-package) — `notPlain := match d.Mod { FuncMod.FuncPlain => false; FuncMod.FuncFrom => true; FuncMod.FuncDerive => true }`; `if notPlain && d.ModPos != (token.Pos{}) { return d.ModPos }`.
- emit.goal:361 funcDecl — replace control switch with `isDerive := match d.Mod {...}` + `if isDerive { e.deriveDecl(d); return }`; the `default: e.fail(...)` arm is dropped. (Avoids empty/no-op match arms; FuncPlain/FuncFrom fall through to ordinary emission.)
- resolve.goal:458 typeString ChanType — value-position match; the former
  `default` (SendRecv) becomes the explicit `ast.ChanDir.SendRecv => "chan " + typeString(x.Value)` arm; RecvOnly/SendOnly keep their bodies.
- emit.goal:2309 chanType — statement-position match; former `default` body
  `e.p("chan ")` becomes the `ast.ChanDir.SendRecv` arm.

Construction sites (FR-4 qualified spelling — supersedes the old tokens above):

- parser.goal:226/229 — `p.parseModFuncDecl(ast.FuncMod.FuncFrom / .FuncDerive)`.
- parser.goal:511/515/520 — `Dir: ast.ChanDir.SendRecv`, `c.Dir = ast.ChanDir.RecvOnly / .SendOnly`.
- parser.goal:365 — `fd := &ast.FuncDecl{Mod: ast.FuncMod.FuncPlain}` (zero-value fix).

Reference-only sites (NO change — type annotations / plain assignment):

- goal_stmt.goal:47 — `parseModFuncDecl(mod ast.FuncMod)` (param type unchanged).
- goal_stmt.goal:50 — `fd.Mod = mod` (assigns an existing enum value).
- ast.goal:161 — field `Mod FuncMod` (field type unchanged).

## Port-gate list (test-divergence resolution, concretely)

The "port-gate list" is the hard-coded `[]string` of `_test.go` paths passed to
`selfhost.BuildAndTest` in `internal/selfhost/port_test.go`. The ast entry is
`port_test.go:142`: `BuildAndTest("selfhost/ast", astPkg, []string{"../ast/ast_test.go"}, deps)`.
Resolution: move the FuncMod assertions (currently `internal/ast/ast_test.go`
~lines 247-284, inside `TestWalkGoalDeclChildren`) into a NEW internal-only file
`internal/ast/funcmod_test.go` (package `ast`). NOTE: `collect`/`assertChildren`
in ast_test.go are FUNCTION-LOCAL closures inside `TestWalkGoalDeclChildren`, not
package-level helpers — the new test must RE-CREATE them (and import `token`).
Also keep the existing per-constant docs as ordinary `//` comments, NOT `///`
(a `///` DOC_COMMENT token between enum variants reaches `p.ident()` and fails
the parse). Do NOT add the new file to the slice at
port_test.go:142 — leaving `ast_test.go` FuncMod/ChanDir-free so it compiles
against BOTH Go-iota internal/ast (task check) and enum-transpiled selfhost/ast
(port gate). `internal/ast` itself stays Go iota.

## Gates

`task check` (includes the corpus behavioral gate
`internal/corpus.TestASTEngineWholeCorpusBehavioralGate` and the
`internal/selfhost` port gates), `task build`, `task fixpoint` all green; corpus
behavioral tier unchanged. Watch fixpoint (touches ast + sema + backend).

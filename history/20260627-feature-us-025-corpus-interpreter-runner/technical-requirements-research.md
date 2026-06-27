# Technical Requirements & Research — US-025

## Existing seams to reuse

- `corpus.Case` / `corpus.Load` / `manifestPath` / `repoRoot` — the corpus model
  and the committed manifest. Doctest cases are `Kind=doctest`, `Mode=file`, with
  `Input` a repo-relative `.goal` path (e.g. `features/11-doctests/examples/add.goal`).
- The Go behavioral doctest tier is `corpus.RunDoctestExec` (behavior_runner /
  doctest_behavior_runner): it transpiles, writes an isolated temp module, and runs
  `go test`. The interpreter analogue evaluates each `///  >>>` doctest expression
  directly and compares the rendered result.
- `internal/interp` is the tree-walking interpreter. `interp.New(file, info, opts...)`
  registers top-level funcs / enums / methods into the root scope; `ip.evalExpr(expr,
  ip.root)` evaluates an expression to a single `Value` (see call_test.go's evalFn).
  `Value.String()` renders int → `5`, string → `"abab"` (strconv.Quote), bool →
  `true` — exactly the Go-literal form the doctest `Expected` lines are written in
  (the doctest sidecar pastes the expected line verbatim as the `want :=` literal).
- Doctests are structured on the AST: `ast.FuncDecl.Doc.Doctests` is `[]*ast.Doctest`
  with `Input string` (text after `>>> `) and `Expected []string` (following lines).
  `backend/doctest.go renderDoctests` is the reference extraction shape.

## Approach

1. Add an exported `interp` entry that evaluates a file's doctests. Parse the source
   (`parser.ParseFile`), resolve (`sema.Resolve`), build the interpreter, and for each
   `FuncDecl.Doc.Doctests` entry parse the `Input` expression and `evalExpr` it against
   the root scope. The interpreter package has no `ParseExpr`, so wrap the input in a
   tiny `func __doctest__() { __dt := <input> }` file, parse it, and lift the assign
   RHS expression node — then evaluate that node against the original program's interp.
2. Compare `Value.String()` of the result to the trimmed joined `Expected`. A mismatch
   is a returned failure; a parse/eval problem is a returned error.
3. `corpus.RunInterp(root, c)` reads the case `Input` file (guarding `Kind=doctest`,
   `Mode=file`), calls the interp doctest runner, and turns any failure or error into
   a case-identified error (mirroring the other corpus runners' error wrapping). It
   fails loudly if the case yields zero doctests.
4. A `corpus` test iterates the manifest's doctest cases through `RunInterp`.

## Dependency note

`internal/corpus` will import `internal/interp` (new edge). `interp` does not import
`corpus`, so there is no cycle. `interp` already imports parser/sema/ast — no new
go/types or typecheck dependency is introduced (US-022 envelope holds).

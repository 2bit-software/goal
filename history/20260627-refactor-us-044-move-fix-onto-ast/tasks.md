# Tasks — US-044 Move goal fix onto the AST

- [ ] T1: Rewrite `fix.go` `File` loop to parse with `parser.ParseFile` +
  `sema.Resolve` each iteration, build a `typeDecls` map from the AST, dispatch
  the four AST fixers, apply via `scan.Splice`, run call-site reporting at
  convergence. Keep helpers; drop `scan.Lex`/`analyze.Build` usage. On parse
  error, no-op the iteration.
- [ ] T2: Rewrite `resultsig.go` to detect `(T, error)` from `FuncType.Results`
  and classify returns over `ReturnStmt` nodes (skip nested FuncLit).
- [ ] T3: Rewrite `propagate.go` for the value-binding and init-guard forms over
  `AssignStmt`/`IfStmt`/`ReturnStmt`, plus Option `*o`→`o` deref rewrites.
- [ ] T4: Rewrite `match.go` to walk `SwitchStmt` nodes and report enum switches.
- [ ] T5: Rewrite `callsite.go` to walk `IfStmt` value-binding guards in
  ModeNone functions.
- [ ] T6: Run `go build/vet/test ./...`; confirm `./internal/fix` and
  `./internal/lsp` suites pass unchanged; remove any scratch files.

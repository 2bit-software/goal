# Scope — US-002 Rename typecheck package to selfhost

## What is being refactored and why

internal/typecheck (the depth checker: mustuse / nozero / implements analyses,
~772 LOC across 5 non-test files) is the last in-scope package not yet mirrored
under selfhost/ as goal source. This completes step 1 of 3 (the verbatim rename)
of the self-host idiomatic plan.

## What the old code looks like

- internal/typecheck/{checker,implements,mustuse,nozero,typecheck}.go
- In-module deps: goal/internal/{token,ast,parser,sema,project,backend}
  (backend pulls in pipeline + lexer transitively).
- Foreign passthrough imports: fmt, go/ast, go/importer, go/parser, go/token,
  go/types, sort, strings.
- Bare reserved words (match/enum/assert) appear ONLY inside string-literal
  error text (mustuse.go:93,194) — no identifier collisions, so verbatim port.
- Tests: checker_test, implements_test, mustuse_test, nozero_test,
  typecheck_test — all white-box (package typecheck), stdlib + project/sema
  imports, no repo-relative fixtures (the `.goal` suffix checks are on synthetic
  filenames). All five are usable in the behavioral gate.

## What the new code should look like

- selfhost/typecheck/{checker,implements,mustuse,nozero,typecheck}.goal —
  byte-for-byte copies of the .go sources (Go superset is valid goal).
- A new TestPortedTypecheckPackage in internal/selfhost/port_test.go using the
  established discoverPorted helper: COMPILE gate via BuildTranspiled over the
  full 9-entry layout {token,lexer,ast,parser,sema,project,pipeline,backend,
  typecheck}; BEHAVIORAL gate via BuildAndTest("internal/typecheck", ...) over
  the dep closure {token,lexer,ast,parser,sema,project,pipeline,backend} with
  the five typecheck test files.

## What should NOT change

- No behavior changes; verbatim rename only. No idiomatization (Result/?, enum,
  match) — deferred to US-004 / US-012.
- internal/typecheck stays in place (side-by-side); selfhost/typecheck is new.
- task fixpoint must stay byte-identical (selfhost/typecheck auto-covered by
  project.Discover).

# NEXT-SESSION — the unified front-end is COMPLETE (11/11); next is the checker

The unified front-end is **built and proven**. One pipeline (`internal/pipeline`) transpiles a
`.goal` program using *any combination* of the 11 features, replacing the standalone reference
transpilers. All 11 reference example suites pass through it unchanged (regression-locked), and a
growing `testdata/` of genuinely multi-feature programs — combinations no single reference can
produce — round-trips to correct, **independently-compiling** Go.

---

## What exists now (the architecture, as built)

```
goal/
  go.mod                      module goal, go 1.26
  cmd/goalc/main.go           CLI: goalc <file.goal|-> -> Go on stdout; goalc -test -> doctest sidecar
  internal/scan/              shared low-level: Token, Lex, Replacement, Splice, match*, ScanFuncs,
                              line/ident helpers, BaseType, LeadIdent, MatchQualifier, MatchBodyBrace
  internal/analyze/           name-keyed tables built ONCE from original source (read-only to passes):
                              FuncSignatures (open/closed Result mode + T/E), Enums, Sealed, Structs,
                              TypeDecls, FromRegistry (ConvEntry{Name,Fallible})
  internal/pass/              one file per construct; each Run(src, *Tables) (string, error),
                              re-lexes, splices, formats NOTHING
  internal/pipeline/          ordered Passes + driver; threads source, formats ONCE; Output{Go, Test}
  testdata/                   multi-feature .goal/.go.expected (the real proof)
```

### Pass order (`pipeline.Passes`)
`implements → defaults → result → option → question → closed → derive → assert → match → enums`,
then **format once**. Doctests run separately on the *original* source (Output.Test).

### The design rules that held (do not regress)
- **Name-keyed tables, never byte offsets.** Passes splice, so offsets shift; every cross-pass fact is
  keyed by symbol name and survives re-lexing. Each pass re-lexes the current source.
- **Format once** at the very end (driver only); no pass calls `go/format`.
- **Partition shared surface by a table fact, don't double-claim:**
  - `match` — claimed by result (open Result), option (Option), closed (closed Result), or match (enum),
    chosen via `MatchQualifier` + callee mode. See `calleeIsClosed`.
  - `type T struct implements X, Y { … }` — the implements pass strips the clause and emits, per
    interface, a marker method when it is `Sealed` else a compile-time assertion (the enums pass no
    longer touches `implements`).
  - `?` — open/Option in the question pass, closed in the closed pass; `funcSpans`/`sigAt` centralize the
    open-vs-closed decision so the two never both claim a `?`.
  - `from func` — ONE registry (analyze); the derive pass strips every leaf; closed-E and derive both read it.
- **Order constraint:** `match` before `enums` (enum patterns must be consumed before the construction
  rewrite); `closed`/`result` before `enums` (generated constructors may contain enum values lowered last).

### Composition bugs found & fixed (regression cases exist for each)
- `match` passes greedily claimed every match → gated by qualifier + mode.
- `__gop_some` (Option box) vs `__gop_o` (`?`-Option temp) must stay distinct.
- doctest extraction missed `from func`/`derive func` → strips the modifier first.
- open-E and closed-E Result in the SAME file → mode partition (see `testdata/open_closed_mix.goal`).

---

## Verification discipline (keep doing all of it)
1. `go vet ./...` clean.
2. `go test -count=1 ./...` — `internal/pipeline` runs three suites: multi-feature `testdata/`,
   single-feature regression (every `features/NN/examples`), and doctests (Output.Test for feature 11).
3. **Independently compile** every generated program in a throwaway module (golden files are generated
   from the tool, so the test is partly tautological — compilation is the real check). For runtime-
   preserved output (assert, doctests, derive) also **run** it (`go test` on the sidecar passes).

---

## Next workstream — THE CHECKER (the largest unbuilt piece)

Every reference deferred its *static guarantee* to a checker that does not exist yet. The front-end
keeps the "located error, not silent footgun" discipline (it defers what it cannot resolve), but emits
no diagnostics. The checker is where the thesis lands:

- exhaustiveness (02 match), must-use (03 Result), field-completeness (08), `implements` satisfaction (07),
  static asserts (10), conversion completeness / registry totality (12),
  closedness & From-totality (06).
- Pairs with codegen's erasure-with-defensive-`panic` (§8.0): today every pass lowers proven-valid input;
  the checker is what *proves* it, turning UB-on-malformed-input into located compile errors.

Suggested shape: a `internal/check` package reading the same `analyze.Tables` (extend them as needed),
run by the driver BEFORE lowering, accumulating diagnostics with source positions. Start with the two
highest-value, self-contained checks — match exhaustiveness (02) and field-completeness (08) — then
must-use (03) and conversion totality (12).

### Deferred front-end forms (each noted in its DECISIONS.md entry; pick up with the checker)
- §8.7 immediate-vs-stored Result/Option (sum-encoding fallback for stored-as-value).
- value-position / untyped `name := match` (needs the checker's inferred result type).
- nested Err-patterns; assert build-tag stripping; doctest methods & multi-line expected.
- feature-12 map/Option/nested-struct recursion; the two bespoke shapes (pmk_upgrade, patterns JSON).

---

## Governing files
- `TODO.md` — 11 live boxes checked; 09-pure cut (per-feature artifact pointers).
- `DECISIONS.md` — the choice/assumption/refusal ledger, §01–§12.
- `goal-design-spec.md` — **read-only**, covers 01–11.
- `FEATURE-AUDIT-PROMPT.md` — the completed per-feature loop (for any *future* feature).

## Housekeeping
- No `.sentrux/`; not indexed in codebase-memory-mcp. Offer to set up once if you touch much code.
- Commits on `main`. Use `mcp__zombiekit__git` + `/commit-message`; stage only the turn's artifacts.
- The 11 standalone `features/NN/transpiler/` reference transpilers still build and are the source of
  truth for each feature's lowering; the unified front-end reuses their logic, re-keyed by name.
  (Cut features are frozen under `features/_cut/` and are not part of the build.)

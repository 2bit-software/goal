# Verification — US-027 Resolve symbols by AST walk

## Acceptance criteria

- AC-1 "internal/sema builds enums/structs/signatures/from-registry/methods by
  walking the AST": MET. `sema.Resolve(*ast.File)` populates Info.Enums,
  Info.Structs, Info.FuncSignatures, Info.FromRegistry, Info.Methods (plus
  Sealed) from the parse tree (internal/sema/resolve.go).
- AC-2 "a test asserts sema resolves the same symbols as analyze.Build for a set
  of representative inputs, including a struct field whose type contains an
  embedded comma (the analyze comma-split bug)": MET.
  - `TestResolveMatchesAnalyze` compares enum variants/fields, struct fields,
    sealed set, the four function signatures (mode/T/E/arity/ends-in-error), the
    from-registry entries, and per-receiver method names against `analyze.Build`.
  - `TestResolveStructCommaFieldType` resolves a generic-typed field
    `Result[int, error]` to a single correct field AND asserts `analyze.Build`
    mishandles the same comma-bearing type (the documented parseStructBody bug).

## Verify gates (prd.json verifyCommands)

- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages, including internal/sema)

## Scope / blast radius

- New: internal/sema/resolve.go, internal/sema/resolve_test.go.
- Modified: internal/sema/sema.go (placeholder Info -> populated fact tables;
  New() still returns an empty Info).
- No back-end/driver change: backend.Transpile still calls sema.New(); Resolve is
  the future entry point (US-032+). No other package touched.

## Findings

No CRITICAL/MAJOR. Implementation matches the spec.

## Assumptions
- Parity is single-file `analyze.Build` (not BuildPackage), per the story.
- Type strings are compared modulo whitespace (semantic equality); the AST
  printer canonicalizes spacing where analyze captures verbatim source.
- The comma-bug exemplar uses a generic instance `Result[int, error]` rather than
  a func-typed field, because the no-semicolon lexer makes a func-typed struct
  field followed by another field ambiguous to the parser (its trailing field
  reads as the func's result type). The generic instance is bracket-delimited,
  so it parses unambiguously while still defeating analyze's whitespace split.
- Method Arity/EndsInError use raw result counts (no Result/Option override),
  matching analyze.methodFrom.

# AI-Consumer Readiness Audit — US-036

## Verdict: implementable without clarifying questions.

- All terms (§8.1 encoding, sum variant struct `Enum_Variant`, guard variable,
  exported field) are defined in-repo by the existing enum lowering (US-033) and
  the legacy `internal/pass/match.go` reference.
- The type-switch shape, the panicking-default message string, and the position
  wrapping (`return <body>` / `name = <body>`) are specified exactly and are
  byte-checkable against the 02-match goldens (build+vet, not exact text here).
- Acceptance criteria are concrete enough to write test assertions: drive each
  case through `backend.Transpile` + `corpus.RunCompile` and assert no error.

## Minor notes
- The default message must match `unreachable: non-exhaustive <Enum> (compiler
  invariant violated)` — copy verbatim from `internal/pass/match.go`.
- The guard variable is emitted only when some arm references its binding; an
  unused-but-declared guard would fail to compile.

## Assumptions
- Value-position lowering covers `return match` and typed `var name T = match`
  only; untyped `name := match` is out of scope (legacy refused it).
- Enum classification uses `matchQualifier` + `enumOf(info, …)`; Result/Option
  matches keep their existing dedicated lowering.

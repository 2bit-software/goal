# Audit — AI-Consumer Readiness

## Verdict: implementable without clarifying questions

- All terms (open-E / closed-E Result, must-use, deferral, sealed) are defined in the
  spec and in `REWRITE-ARCHITECTURE.md` / the lexical checker docs.
- Data shapes are concrete: the `Info` model (`internal/sema/sema.go`) already carries
  `FuncSignatures` (Mode/E/Arity/EndsInError), `Enums.VSet`, `Methods.Sig`, `Sealed`,
  and `FromRegistry`. The one gap — in-file interface method sets — is named explicitly
  in the technical-requirements note and resolved by extending `Info`.
- State transitions (ok / Error / deferral) are enumerated per check with the exact
  diagnostic code for each outcome.
- Acceptance criteria are test-shaped: "every case in `testdata/check/<dir>` passes
  through the corpus runner" maps directly to a `TestSema…Runner` that walks the
  manifest, mirroring the existing `sema_checker_test.go` / `sema_fields_test.go`.
- Message wording is not guessed: it mirrors the lexical checker so the `// want`
  substrings match.

No jargon-without-definition. No missing field/type specs. Ready.

# US-024 Gate: parse 100% of corpus — Business Specification

## Overview

The AST front-end parser must accept every goal source input in the project's
golden corpus. This gate proves the parser grammar is complete against the spec
corpus and unblocks all later AST-backend phases, which depend on a parser that
never rejects valid goal source.

## Functional Requirements

### FR-1: Whole-corpus parse gate
The test suite SHALL iterate every `.goal` input referenced by the corpus
manifest (file-mode inputs and every file of each package-mode case) and parse
each one through the AST front-end parser.

### FR-2: Zero parse errors
Every corpus `.goal` input SHALL parse with zero parse errors.

### FR-3: Loud failure listing
If any input fails to parse, the test SHALL fail and report each failing input
by path together with its parse error, so a maintainer can see exactly which
inputs regressed.

### FR-4: Grammar completeness for existing valid syntax
The parser SHALL accept the goal syntax already present in the corpus, including
multi-element generic type-argument lists, type-literal operands used in
expression position, and enum payload fields written with or without a colon —
without any source `.goal` file being modified to fit the parser.

## Acceptance Criteria

- [ ] A test enumerates every unique `.goal` input in the corpus manifest and
      parses it.
- [ ] All enumerated inputs parse with zero errors.
- [ ] When an input fails, the test names that input and surfaces its error
      (verified by the test reporting a non-zero failure count loudly).
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

Developer-facing only: `go test ./...` runs the gate; a regression that makes any
corpus input unparseable turns the gate red with the offending input(s) listed.

## Error Handling

The parser already returns accumulated, position-tagged errors (joined) and never
panics on malformed input. The gate consumes that error value: a non-nil error
for any input is a failure, attributed to that input's path.

## Out of Scope

- Semantic analysis, type checking, lowering, or code emission of the parsed
  trees (later stories).
- Changing or normalizing any corpus `.goal` source file.
- AST snapshot goldens (US-025).
- Func-literal operands and other Go-subset forms not exercised by the corpus.

## Open Questions

None. The corpus is fixed and the parser's required grammar is fully determined
by the inputs it must accept.

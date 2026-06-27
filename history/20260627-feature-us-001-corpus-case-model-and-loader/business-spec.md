# Corpus Case Model and Loader — Business Specification

## Overview

The golden test suite is currently coupled to package layout via hardcoded
directory lists and relative paths. This feature introduces a runner-independent
data model for golden cases plus a loader that reads a manifest from disk, so any
future runner (transpile, check, doctest, behavioral) can consume one shared
description of the corpus regardless of where the source files live.

This is Phase 0.1 of the AST front-end rewrite: it lays the foundation only — it
defines the model and loader. It does not generate the real manifest, move source
files, or build runners (those are later stories).

## Functional Requirements

### FR-1: Case model
A Case SHALL describe a single golden case with these fields: a unique identity,
a kind, an input, an expected result, a mode, and a normalization strategy.

### FR-2: Kinds
A Case kind SHALL be one of: transpile, check, or doctest.

### FR-3: Modes
A Case mode SHALL be one of: file or package.

### FR-4: Normalization
A Case SHALL carry a normalization strategy describing how its expected output is
compared (e.g. gofmt-normalized, or none).

### FR-5: Manifest loader
The system SHALL provide a loader that, given a path to a manifest file, returns
the collection of cases (a Manifest) or an error if the file cannot be read or
parsed.

## Acceptance Criteria

- [ ] A Case exposes ID, Kind, Input, Expected, Mode, and Normalize fields.
- [ ] Kind admits exactly transpile, check, doctest.
- [ ] Mode admits exactly file, package.
- [ ] Loading a valid manifest returns every case it contains, with all fields
      populated as written.
- [ ] Loading a missing or malformed manifest returns an error, not a panic.
- [ ] A fixture manifest containing one transpile, one check, and one doctest
      case loads, and each field of one case of each kind matches the fixture.

## User Interactions

No end-user UI. The audience is the compiler engineer / test harness: a Go API
`corpus.Load(path)` returning a `corpus.Manifest`, and the `corpus.Case` value
type.

## Error Handling

- Missing manifest file: loader returns a descriptive error.
- Malformed manifest (invalid JSON): loader returns a descriptive parse error.
- No panics on bad input.

## Out of Scope

- Generating the real manifest over the existing corpus (US-002).
- Building transpile/check/doctest/behavioral runners (US-003+).
- Moving or rewriting any existing corpus source files.

## Open Questions

- None. The model and loader are fully specified by the story and the Phase 0.1
  plan in REWRITE-ARCHITECTURE.md.

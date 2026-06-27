# US-028 Gate: script-to-module no-op — Business Specification

## Overview

goscript programs run under a tree-walking interpreter for fast, toolchain-free
iteration. The promise of the project is that such a program "graduates" into a
compiled Go+ module without rewriting: the same source that the interpreter runs
also transpiles and builds as a Go module via the existing AST backend, with the
same observable behavior. This feature adds a conformance gate that proves that
promise — the script-to-binary upgrade is a no-op.

## Functional Requirements

### FR-1: Sample program runs under the interpreter
A representative sample goscript program — one that exercises a genuine goal
construct (an enum plus a value-position `match`) and prints to standard output —
runs under the goscript interpreter and produces observable output.

### FR-2: Same source builds as a Go+ module
The same, unchanged sample source transpiles via the existing AST backend and
builds and runs as a Go module, producing observable output.

### FR-3: Outputs are identical (the no-op)
The output observed from the interpreter run and the output observed from the
transpiled-then-built binary, for the same unchanged source, are identical.

### FR-4: Divergence fails loudly
If the two paths produce different output (or either path fails to run/build),
the gate fails with a descriptive message identifying both observed outputs (and
the underlying failure), never a silent pass.

## Acceptance Criteria

- [ ] A test runs the sample goscript program under the interpreter and captures
      its standard output.
- [ ] The same test transpiles the unchanged source via the AST backend, builds
      it as a Go module, runs the binary, and captures its standard output.
- [ ] The test asserts the two captured outputs are equal and non-empty.
- [ ] The sample program exercises at least one genuine goal construct (not pure
      Go), so the no-op upgrade is meaningful.
- [ ] If the outputs differ, the test fails and reports both outputs.

## User Interactions

No new user-facing surface. The behavior being asserted is the existing
contract: a user can run a program under the interpreter and later build the same
source as a module with equivalent results.

## Error Handling

A transpile failure, a build failure, an interpreter run failure, or an output
mismatch each fails the gate with a case-identified, descriptive message.

## Out of Scope

- Asserting parity across the WHOLE corpus under both paths (US-027 already gates
  whole-corpus behavioral parity under interpretation).
- Multi-file / package-mode programs (the interpreter run path is single-file).
- Any new CLI surface or `goal` subcommand.
- Performance comparison between the two paths.

## Open Questions

None — both paths and the temp-module/toolchain pattern already exist in-repo and
are exercised by existing green tests.

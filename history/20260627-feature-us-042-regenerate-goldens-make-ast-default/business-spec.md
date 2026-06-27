# US-042 Regenerate goldens and make AST default — Business Specification

## Overview
The AST front-end (the new backend) becomes the canonical transpile path. The
checked-in `.go.expected` goldens are regenerated from the AST backend's real
output so the exact (byte-level, gofmt-normalized) conformance tier measures the
new engine. The `goal` CLI defaults to the AST engine; the legacy splice engine
stays reachable behind `--engine=splice` for one release.

## Functional Requirements

### FR-1: AST engine is the CLI default
`goal build` and `goal run` use the AST engine when no `--engine` flag is given.
`--engine=splice` still selects the legacy engine; `--engine=ast` is accepted and
is a no-op relative to the default.

### FR-2: Goldens reflect AST output
Every `.go.expected` golden for a file-mode transpile case (and every doctest
sidecar golden) equals the AST backend's output for that input, after
gofmt-normalization. Goldens are produced by the backend, never hand-edited.

### FR-3: Exact tier is green on the default engine
The exact transpile/doctest conformance tier compares each regenerated golden
against the AST backend output and passes for every case.

### FR-4: Splice remains available
The splice engine remains selectable via `--engine=splice` and remains exercised
by the behavioral conformance tier (build/vet/run), so it is a working fallback
for one release.

### FR-5: Documentation reflects the new default
CLI usage text and the generated AI knowledge guide describe the AST engine as
the default and the splice engine as the legacy opt-in.

## Acceptance Criteria

- [ ] With no `--engine` flag, the resolved engine is the AST engine.
- [ ] `--engine=splice` and `--engine=ast` are both accepted; an unknown value errors.
- [ ] Every regenerated `.go.expected` equals the AST backend output (gofmt-normalized).
- [ ] The exact transpile tier passes for every file-mode transpile case on the AST engine.
- [ ] The exact doctest tier passes for every doctest case on the AST engine.
- [ ] The whole-corpus behavioral gate still passes after the flip.
- [ ] The generated AI knowledge guide and CLI usage text name AST as the default.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions
- `goal build [path]` / `goal run [path]` — now AST by default.
- `goal build --engine=splice [path]` — opt into the legacy engine.

## Error Handling
- An unknown `--engine` value remains a usage error naming the bad value.
- A bare `--engine` with no value remains a usage error.

## Out of Scope
- Deleting the splice passes / dead token scanning (that is US-043).
- Moving fix/fmt/LSP onto the AST (US-044+).
- Any change to manifest case counts or paths.

## Open Questions
- None. The seam (Transpiler interface) and the proven snapshot `-update` pattern
  make the engine swap and golden regeneration mechanical.

# Completeness Audit — US-004

## Findings

No CRITICAL or MAJOR findings. The spec covers the happy path (FR-1, FR-2), the
single error case (FR-3, missing main), and explicitly scopes out statement/
expression evaluation, args/returns, CLI, and capabilities.

### MINOR-1: Multiple `func main` / method named main
The spec says "a plain function with no receiver". It does not state behavior if
two `main` functions exist or a method is named `main`. Resolution: pick the
first top-level plain `func main`; ignore methods (Recv != nil). Low impact — the
sema/parser front-end and Go itself forbid duplicate top-level `main`, so this is
defensively handled, not a real gap.

### MINOR-2: Package-mode entry
AC mentions "(or package)". This story uses single-file `*ast.File` as the
canonical input (matching the trivial AC program); package-mode entry can be a
thin wrapper later. Not blocking.

## Assumptions

- The interpreter is constructed from a single `*ast.File` (single-file mode);
  the AC program is single-file. Package-mode is deferred.
- `sema.Resolve` (not `sema.New`) is used in the test so the real front-end runs.
- An empty `main` body is a legitimate no-op returning nil.

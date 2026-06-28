# Scaffold notes — US-010

This refactor is a **pure relocation / archive**, not a code rewrite. There is no
new implementation to build side-by-side: the superseded `features/NN/transpiler/`
modules are simply moved to `attic/features/NN/transpiler/`.

- New files/functions created: none. (No new code; only a directory move.)
- How it differs from old: identical contents, new location under `attic/`.
- Adapters/wrappers: none needed. The directories are standalone Go modules
  (own `go.mod`) excluded from the root `goal` module, so nothing in the root
  module references them — relocation is inert to root build/vet/test.
- Independent testability: the root module's `go build/vet/test ./...` is the
  guard; it excluded these modules before and after the move, so it must stay
  green. The archived modules retain their own `go.mod`/tests should anyone wish
  to build them in place under `attic/`.

The actual move is performed at the **cutover** step (via `git mv`), which is the
correct place for a relocation in this workflow. Verification before cutover
confirms the root module is already green.

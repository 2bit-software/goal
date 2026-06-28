# Audit — AI-Consumer Readiness

An implementer can proceed without clarifying questions:

- Exact output text is specified by the in-repo reference (`internal/pass/
  enums.go`, `internal/pass/implements.go`) and the checked-in `.go.expected`
  goldens for the 7 example cases.
- sema fact shapes (Enum/Variant/Field, Sealed) are concrete and already
  populated by Resolve.
- The pointer-receiver determination is specified: a type with any
  pointer-receiver method uses `(*T)(nil)`, else `T{}`.
- Acceptance is machine-checkable: prd verifyCommands + a backend test driving
  the 7 cases through `corpus.RunCompile` (build + vet).

One implementation choice left to the implementer (non-blocking): lowering is
folded into the emitter rather than a separate `lower` package — the arch doc
marks `lower`/`ir` as optional, and emit.go already does direct emission.

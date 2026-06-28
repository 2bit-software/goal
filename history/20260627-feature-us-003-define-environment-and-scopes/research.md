# Research — US-003 Environment and scopes

## Summary

A lexical environment is the standard tree-walking-interpreter scope chain: a
parent-linked map of name -> Value. The canonical design (Crafting
Interpreters, go/types Scope, gopls) is:

- `Env` holds `vars map[string]Value` and a `*Env parent`.
- `Define(name, val)` binds in the current (innermost) scope, overwriting any
  binding already in THIS scope.
- `Lookup(name)` checks the current scope, then walks `parent` until found or
  the root is exhausted; on exhaustion it returns a not-found error.
- `NewChild()` returns a fresh `Env` whose parent is the receiver. A binding of
  the same name in the child shadows the parent; the parent binding is intact
  and visible again once the child is discarded.

## Project fit

- internal/interp already defines `Value` (value.go, US-002). Env stores
  `Value`, no new value plumbing needed.
- Zero-dependency, stdlib `testing` only (no testify).
- The not-found result is a named error so a later eval story surfaces a
  located "undefined: x" rather than a silent zero Value.

## Confidence: High. The pattern is textbook and the value type already exists.

## Open questions: none material for this story. Assignment-through-chain
(write to an existing outer binding vs. define-in-place) is an eval concern
(US-006); this story only needs Define/Lookup/NewChild.

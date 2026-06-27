# Environment and Scopes — Business Specification

## Overview

The goscript interpreter needs lexical scoping so that variables resolve and
shadow correctly during evaluation. This feature introduces an `Env`: a
parent-linked chain of name-to-value bindings that models nested lexical
scopes. Inner scopes shadow outer ones, and lookups fall through to enclosing
scopes until a binding is found or the chain is exhausted.

## Functional Requirements

### FR-1: Define a binding
An `Env` SHALL bind a name to a runtime value in its own (innermost) scope.
Defining a name already present in the same scope replaces its value.

### FR-2: Look up a binding
An `Env` SHALL resolve a name by checking its own scope first, then walking
toward the root through parent scopes, returning the first binding found.

### FR-3: Open a child scope
An `Env` SHALL produce a child scope whose parent is the receiver. Bindings in
the child are independent of the parent.

### FR-4: Shadowing
A binding in an inner scope SHALL take precedence over a same-named binding in
any outer scope, without mutating the outer binding. The outer binding is
visible again once the inner scope is discarded.

### FR-5: Not-found is an explicit error
Looking up a name bound in no scope along the chain SHALL return a named
not-found error, never a silent zero value.

## Acceptance Criteria

- [ ] Defining a name in a scope and looking it up in that same scope returns
      the defined value.
- [ ] A name defined only in an outer scope is found via lookup from an inner
      child scope (parent fall-through).
- [ ] A name redefined in an inner child scope returns the inner value from the
      child, while the outer scope still returns the outer value (shadowing,
      non-destructive).
- [ ] Looking up a name that is bound nowhere in the chain returns a not-found
      error identifying the missing name.

## User Interactions

No direct user interaction. `Env` is an internal runtime API consumed by the
interpreter's evaluation stories (US-004 onward).

## Error Handling

A lookup miss returns a not-found error value (carrying the missing name) and a
false/zero second result, so callers must handle absence explicitly rather than
receiving a usable-looking zero value.

## Out of Scope

- Assignment semantics (writing through to an existing outer binding vs.
  defining in place) — that is an evaluation concern (US-006).
- Typed environments / type checking — types are checked statically and erased
  at runtime.
- Concurrency safety — the interpreter is single-threaded in this phase.

## Open Questions

None.

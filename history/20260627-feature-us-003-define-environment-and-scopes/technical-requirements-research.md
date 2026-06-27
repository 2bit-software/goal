# Technical Requirements / Research — US-003

- New code lives in `internal/interp` (alongside value.go, US-002).
- Zero-dependency: stdlib only; tests use stdlib `testing` (NO testify).
- Env is the lexical scope chain the later eval stories (US-005 onward) read
  and write through. Define binds in the current scope; Lookup walks up the
  parent chain; NewChild opens an inner scope that shadows its parent.
- Lookup of an undefined name returns a located/named not-found error rather
  than a zero Value silently.

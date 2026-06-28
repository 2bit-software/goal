# Audit: Completeness — US-011

## Findings

### MINOR — Literal sub-kinds granularity unspecified
FR-1 says "the literal classes". The Go convention (`go/token`) distinguishes INT,
FLOAT, IMAG, CHAR, STRING. The spec does not mandate the exact split. Resolution:
follow `go/token`'s literal kinds (INT, FLOAT, IMAG, CHAR, STRING) — sufficient for
the lexer stories and not blocking.

### MINOR — Round-trip scope for operators
FR-3 requires round-trip for "every keyword/lexeme kind". Sentinels (ILLEGAL, EOF),
IDENT, and literal kinds have no fixed source spelling, so they are reasonably
excluded from name→Kind lookup. The spec already scopes the round-trip to
"keyword/punctuation lexeme", which is consistent. No change needed.

### MINOR — Pos comparison shape
FR-4 mandates comparability ordered by Offset but does not name the method. Resolution:
expose a `Less(Pos) bool` (offset-ordered). Implementation detail, non-blocking.

## No CRITICAL or MAJOR findings.

The lexeme inventory is fully determined by REWRITE-ARCHITECTURE.md §1.1/§1.2 and the
existing `internal/scan` keyword set; acceptance criteria are testable as written.

## Assumptions
- Literal kinds follow Go's split (INT/FLOAT/IMAG/CHAR/STRING).
- `Pos.Less` is offset-ordered (Line/Col carried for diagnostics, not for ordering).
- A minimal `Token{Kind, Lit, Pos}` aggregate may be defined for the lexer to emit,
  but is not required by the US-011 gate.
- Contextual keywords (`implements`/`sealed`/`from`/`derive`) are excluded from the
  keyword table by design (they remain IDENT).

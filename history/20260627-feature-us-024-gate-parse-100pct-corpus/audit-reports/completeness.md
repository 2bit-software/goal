# Completeness Audit — US-024

## Findings

No CRITICAL findings. No MAJOR findings.

### MINOR-1: "Unique input" definition
FR-1 says "every unique `.goal` input". Doctest cases share an Input path with
their transpile twin, and package cases carry their files under `Package.Files`
rather than `Input`. The spec resolves this in FR-1 ("file-mode inputs and every
file of each package-mode case") and the technical research notes the dedupe.
Adequately specified; flagged only so the implementer dedupes by path.

### MINOR-2: Empty-corpus guard
The spec does not state behavior when the manifest yields zero inputs. The
established corpus-runner convention (US-003..US-008) is a loud `t.Fatalf` on
zero cases. The implementer SHALL follow that convention. Non-blocking.

## Assumptions

- The gate parses inputs only; it does not assert AST shape (that is US-025).
- "Parse with zero errors" == `parser.ParseFile` returns a nil error.
- Source `.goal` files are immutable; only the parser is changed.

# Technical Requirements & Research — US-024

## Placement of the gate test

- `internal/corpus` already loads `corpus/manifest.json` (`corpus.Load`) and owns
  the runner pattern (RunTranspile/RunCheck/RunDoctest/RunCompile). Add a
  `Parser` interface + `RunParse` runner consistent with that pattern, and a gate
  test in `package corpus`.
- No import cycle: nothing in corpus's dependency tree (pipeline, check, project)
  imports `internal/parser`, so corpus may import parser in one direction.
- The manifest has 107 cases (51 file transpile + 50 check + 4 doctest + 2
  package). Unique `.goal` inputs = file-mode `Input` paths plus each package
  case's `Package.Files`. Doctest cases share Input with their transpile twin, so
  dedupe by path.

## Parser grammar gaps the gate exposes (from a recon pass over the corpus)

Recon: 34/104 unique inputs failed to parse. Categories:

1. **Multi-element type-argument / index lists** (30 cases): `Result[int, error]`,
   `Result[[]byte, error]`, `map[string]string` (map already handled). The type
   name index (`typeNameFrom` → `parseIndexSuffix`) and the expression postfix
   index parse only ONE element then `expect(RBRACK)`, so a comma fails. Need a
   comma-separated list; >1 element → a new `ast.IndexListExpr` (parallel to
   go/ast), 1 element → existing `ast.IndexExpr`. Elements that are themselves
   types (`[]byte`) must parse — handled by gap 2 below making `parseOperand`
   accept type-literal starts so `parseExpr` covers them uniformly.

2. **Type-literal operands in expression position** (2 cases): `[]byte(p)`
   conversion and `map[string]string{}` composite literal. `parseOperand` does
   not accept `[`/`map`/`struct` as the start of an operand. Extend it to parse
   those as type expressions, and extend `compositeOK` to allow ArrayType/MapType/
   StructType (and IndexListExpr) so a trailing `{...}` is taken as a composite
   literal and a trailing `(...)` as a conversion call.

3. **Optional-colon enum payload fields** (1 case):
   `Active { since int }` (Go-style, no colon) vs the more common
   `Active { since: Time }`. `parsePayloadField` hard-requires the colon. Make the
   colon optional.

## Verification

- New `corpus.RunParse` gate test over the whole manifest (all unique inputs).
- Re-run the three project verify gates.

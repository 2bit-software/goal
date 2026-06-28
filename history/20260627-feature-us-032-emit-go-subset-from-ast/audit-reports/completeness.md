# Audit 1: Completeness — US-032

## Findings

### MINOR — Empty / boundary switch forms unstated
The spec calls out expression `switch` with `case`/`default` but does not
explicitly enumerate the tag-less switch (`switch { case cond: }`) or a switch
with an init statement (`switch x := f(); x { ... }`). Implementation SHALL
handle both since the parser (US-018) produces them; the behavioral fixture
should exercise at least the tag-and-default form. Non-blocking: the AC "build
and vet cleanly" transitively forces correctness for whatever forms the fixture
uses.

### MINOR — "full ordinary-Go subset" defined by reference
FR-1 says "every ordinary-Go construct that goal source can contain." The
authoritative definition is the parser's node set, not an in-spec list. This is
acceptable because the parser is the single source of truth, but the spec relies
on that external reference.

## No CRITICAL or MAJOR findings

The requirement is a single, well-isolated gap (switch emission) with a concrete,
already-built verification path (corpus behavioral tier). Happy path, error
path (unsupported-construct error), and the verify gates are all covered by the
acceptance criteria.

## Assumptions
- The "Go subset" is exactly the set of nodes the existing parser produces;
  forms outside it (labeled/send/select/type-switch, variadic call spread) are
  out of scope and need no emitter support.
- A single enriched plain-Go fixture exercised through the behavioral tier is a
  sufficient AC-2 witness; per-corpus-case coverage is not required by the story.

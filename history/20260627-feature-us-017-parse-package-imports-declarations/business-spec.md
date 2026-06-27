# Parse package, imports, declarations — Business Specification

## Overview
The goal front-end rewrite replaces token-splicing with a real lexer → parser → AST
pipeline. This story adds the first parser tier: turning a goal source file's
top-level structure into an `ast.File`. It parses the package clause, the import
block(s), and the func/type/var/const declarations of the Go subset, so later
stories (statements, expressions, checks, backends) operate on a real tree.

## Functional Requirements

### FR-1: Package clause
The parser SHALL parse `package <name>` and record the package keyword position and
the package name on the resulting file.

### FR-2: Imports
The parser SHALL parse single (`import "p"`) and grouped (`import ( ... )`) import
declarations, including named (`m "p"`), blank (`_ "p"`), and dot (`. "p"`) imports,
recording each as an import spec reachable both from the file's import list and its
declaration list.

### FR-3: Type/var/const declarations
The parser SHALL parse single and grouped `type`, `var`, and `const` declarations of
the Go subset into general declarations whose specs carry the declared names, the
optional type expression, and (for var/const) the optional initializer values.

### FR-4: Function declarations
The parser SHALL parse function and method declarations (optional receiver, name,
parameters, results) into function declaration nodes. The function body is captured
as a block spanning its braces.

### FR-5: Type expressions
The parser SHALL parse the Go-subset type expressions needed for declaration shape:
qualified names, pointer, array/slice, map, struct (with its field list), interface,
function, channel, and single-index forms.

## Acceptance Criteria
- [ ] Parsing a Go-subset goal file yields a file node whose package name matches.
- [ ] Single, grouped, named, blank, and dot imports all appear in the file's imports.
- [ ] Grouped and single type/var/const declarations produce the expected ordered
      declaration list with the right spec names and kinds.
- [ ] A function declaration with a receiver, parameters, and results parses with the
      correct name and signature shape.
- [ ] The whole representative sample parses with no error, and a test asserts the
      declaration-list shape (count, kinds, spec names).
- [ ] Build, vet, and the full test suite stay green.

## User Interactions
Programmatic only: `ParseFile(src) (*ast.File, error)` from `internal/parser`.

## Error Handling
On malformed input the parser SHALL return a non-nil error carrying the offending
source position; on well-formed Go-subset input it returns a nil error.

## Out of Scope
- Statement-body parsing (function bodies are captured as an unparsed brace span) — US-018.
- Full precedence-climbing / unary / postfix-`?` expression parsing — US-019.
- Goal-specific declarations: enum, sealed interface, implements, from/derive — US-020+.
- Comment/trivia attachment and formatting fidelity — US-045.

## Open Questions
None — scope is bounded by the Go-subset AST already defined in `internal/ast`.

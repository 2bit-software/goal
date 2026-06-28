# Business Spec — US-001 Foreign enrichment to sema.Info

## Outcome

The AST-based checker can reason about types and methods that live in imported
packages, so checks (and later the backend) no longer need the legacy
analyze.Tables to resolve a `derive func` / `from func` / `?` callee whose
source, target, or receiver type is out-of-package.

## Requirements

- sema.Info exposes a ForeignMethods map keyed by the goal-source spelling of an
  imported method (`pkg.Type.Method`), valued by its return signature.
- Imported struct field sets are populated into the same Structs map sema
  already exposes (keyed `pkg.Type`, fields typed qualified by the import
  alias).
- A function sema.EnrichForeign resolves imported package declarations through
  the Go toolchain, driven by the parsed file's import list (the AST), not by
  re-lexing source.
- The resolver (DirResolver) and its default implementation (DefaultResolver)
  are provided by internal/sema so callers and tests can inject a fixture
  resolver.
- The new code reads imports from the AST; it never calls scan.Lex or
  analyze.ParseImports.

## Acceptance

For a multi-file fixture importing a package with an exported struct and an
exported method, sema.EnrichForeign produces ForeignMethods + Structs entries
identical (field-for-field) to what analyze.EnrichForeign produces for the same
input.

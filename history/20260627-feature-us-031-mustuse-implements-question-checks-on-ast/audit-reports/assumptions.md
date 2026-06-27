# Assumptions

1. **Scope = the three dirs that map to the named checks.** US-031 covers must-use
   (03-result), implements (07-implements), and question/closed-E `?` (05-question-prop
   has no fixture dir; 06-error-e is the closed-E `?` fixtures). 10-assert and
   12-derive-convert are explicitly later stories and are NOT wired here.

2. **`?` sites are checked only at well-formed statement positions** (assignment RHS or
   bare expression statement). This matches 100% of the corpus and avoids the
   `question-not-statement` Error firing on positions the grammar would not produce.

3. **Closed-E `Result.Err` passthrough is simplified to "non-`E.Variant` arg ⇒ defer"**
   rather than porting the lexical checker's full param/var/match-binding passthrough
   analysis. Produces no false Error and passes every fixture.

4. **Interface method signatures are normalized with the same helpers as concrete
   methods** (`paramTypeListFL`/`joinTypes` → `params|results`) so the two sides compare
   correctly; interface `Method.Raw` is left empty so the "missing method" suggestion
   renders `Name() results` (matching the lexical `// want` markers).

5. **`sema.Check` aggregates the new checks**, so the existing 02/08 runners also
   execute them; verified by inspection that no 02/08 fixture triggers a new Error.

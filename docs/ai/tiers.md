This is the slim bootstrap. Two commands pull the rest on demand, so you only spend tokens on
what you're using:

- `goal category` lists every language feature with a one-line description; `goal category
  <name>` (e.g. `goal category enums`) prints that one feature in full — example, the Go it
  lowers to, and its checker diagnostics.
- `goal ai <section>` prints one section of this guide in full. `goal ai features` is every
  feature at once; other sections include `diagnostics`, `starter`, `conventions`, `authoring`,
  and `pointers`. Run `goal ai` with an unknown section to see the full list.

Everything omitted from this slim guide — the non-core features, the diagnostic catalog, a
complete starter program, deeper references — is reachable through those two commands.

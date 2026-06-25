These tokens and shapes are recognized verbatim — do not deviate:

- **Qualified construction:** `EnumName.Variant(...)`, `Result.Ok(...)`, `Result.Err(...)`,
  `Option.Some(...)`, `Option.None`. Never bare `Ok` / `Some`.
- **Brace-named payloads** in enum variants: `Variant { field: Type, field2: Type }`.
- **Newline-separated** enum variants and `match` arms (commas only inside a single payload;
  a one-line variant list may use `;` separators).
- Conventional tokens verbatim: `Ok` `Err` `Some` `None` `=>` `_` `?`.
- Modifiers precede `func`: `from func`, `derive func`.
- Synthesized temporaries are prefixed `__goal_` — never name your own identifiers that way.

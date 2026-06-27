# Business Spec — US-033 Lower and emit enums and implements

## What

As a goal developer, I need enum / sealed-interface / `implements` lowering on
the new AST backend so closed types work through `--engine=ast`.

## Outcomes

- An `enum Name { ... }` declaration lowers to the closed-sum Go encoding:
  `type Name interface{ isName() }`, one `type Name_Variant struct{...}` per
  variant, and a `func (Name_Variant) isName() {}` marker method per variant.
- A variant construction lowers to its Go encoding:
  - data-less `Name.Variant` -> `Name(Name_Variant{})`
  - payload `Name.Variant(label: v)` -> `Name(Name_Variant{Label: v})`
    (labels exported, nested constructions lowered recursively).
- A `sealed interface Name {}` lowers to `type Name interface{ isName() }`.
- A struct `implements` clause lowers to:
  - the marker method `func (T) isI() {}` when `I` is a sealed interface, or
  - the compile-time assertion `var _ I = T{}` (or `var _ I = (*T)(nil)` when
    `T` has a pointer-receiver method) for an ordinary interface.
  The `implements` clause is stripped, leaving a plain Go struct.

## Acceptance criteria

1. lower+backend produce the sum encoding for enums and the assertion/marker for
   implements.
2. The 01-enums and 07-implements transpile cases pass the behavioral tier
   through the new (AST) backend.

## Constraints

- Zero-dependency; stdlib `testing` only.
- Encoding must build + vet (behavioral tier); exact-golden parity is US-042.

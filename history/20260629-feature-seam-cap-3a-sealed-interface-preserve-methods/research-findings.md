# Research Findings

This is an internal transpiler change; no external/web research applies. Findings
come from direct codebase investigation.

## Confirmed facts

- `ast.SealedInterfaceDecl` already carries `Methods *FieldList` (parsed by
  `parser.parseSealedInterfaceDecl`, which calls `parseInterfaceBody`). The body is
  NOT lost at parse time — only at emit time. (internal/ast/goal_decl.go:103-108;
  internal/parser/goal_decl.go:82-89). Same in selfhost/ast/goal_decl.goal and
  selfhost/parser/goal_decl.goal.
- `emit.go sealedInterfaceDecl` discards `d.Methods` and calls
  `genSealedInterface(name)` (lower.go), which hardcodes `type Name interface{ isName() }`.
- The emitter already knows how to render interface methods: `interfaceType`
  (emit.go:521-540) emits each `*ast.Field` method as `Name(params) results` via
  `identList` + `funcSig` + `expr`. This loop is the reusable rendering path.
- Output is gofmt-normalized (`backend.Format` -> `format.Source`), so exact
  whitespace/newlines in the emitter are not load-bearing.
- All existing sealed-interface fixtures use an empty body (`sealed interface Shape {}`),
  so keeping the empty case byte-identical preserves fixpoint and existing goldens.
- selfhost/backend/emit.goal interfaceType / funcSig / identList are identical to
  internal/, so the mirror edit is line-for-line the same.

## Chosen approach

When `Methods` is empty: keep `genSealedInterface` (compact). When non-empty: emit a
multi-line interface with the declared methods (reusing the interfaceType method-emit
loop, extracted into a shared helper) plus the marker method `isName()`.

## Confidence: High. The fix is localized to one emitter method per side plus a tiny
shared helper; no parser/sema change needed.

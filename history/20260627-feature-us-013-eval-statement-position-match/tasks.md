# Tasks — US-013 Eval statement-position match

- [x] T1. Extend `evalSelector` (internal/interp/eval.go) to read a payload field
  off a `KindVariant` receiver via `Value.Field`, keeping struct-field reads and
  the non-struct/non-variant refusal.
- [x] T2. Add `execMatch` / `execArm` / `execArmBody` (internal/interp/interp.go):
  tag dispatch, payload binding into a child scope, generic arm-body execution,
  and the `unreachable` panicSignal default.
- [x] T3. Intercept `*ast.MatchExpr` in `execStmt`'s ExprStmt case, before the
  CallExpr path.
- [x] T4. Add internal/interp/match_test.go: per-variant dispatch + payload bind,
  `_` rest arm, defensive `unreachable` panic, non-variant refusal.
- [x] T5. Run verify gates (go build/vet/test) + the US-022 dependency check.

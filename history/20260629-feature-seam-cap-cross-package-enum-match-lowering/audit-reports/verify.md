# Verification

Source of truth: business-spec.md acceptance criteria.

| Acceptance criterion | Result |
|---|---|
| Cross-package enum match transpiles without error | PASS — `unsupported statement-position match` gone; TestCrossPackageEnumMatchLowers green |
| Transpiled package compiles and links (go build) | PASS — corpus ModePackage case `cross-pkg-enum` (TestASTEnginePackageBehavioralTier) green |
| Lowered switch == hand-written type-switch at runtime | PASS — TestCrossPackageEnumMatchBehavesLikeSwitch runs the generated code against a reference switch |
| Same-package enum, Result, Option matches unchanged | PASS — full corpus behavioral tier green, additive change only |
| task check / build / fixpoint green; corpus tier unchanged | PASS — task check green, task build clean, task fixpoint = FIXPOINT OK |

FR coverage:
- FR-1 (qualified variant patterns recognized): matchQualifier SelectorExpr case.
- FR-2 (imported enums resolve): EnrichForeign reconstructs foreign enums into info.Enums.
- FR-3 (correct lowering): emits `case pkg.Enum_Variant:` (verified in generated Go).
- FR-4 (no regression): corpus behavioral tier green; both bootstrap stages agree.

Findings: none (no CRITICAL/MAJOR/MINOR). Feature works as specified.

## Assumptions validated
- Tag-only enum fixture is sufficient to prove the capability (real seam targets
  FuncMod/ChanDir/Mode/Severity are tag-only). reconstructForeignEnums also builds
  FieldSet, so payload variants are supported when needed.
- selfhost mirror kept in lockstep; fixpoint self-consistency preserved.

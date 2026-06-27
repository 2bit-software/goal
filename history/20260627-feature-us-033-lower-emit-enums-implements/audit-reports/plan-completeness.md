# Plan Audit — Coverage

Each acceptance criterion traces to a plan element:

- AC1 (sum encoding + implements marker/assertion): emit.go EnumDecl/
  SealedInterfaceDecl cases + lower.go genEnum/genInterface/implementsMarker +
  construction lowering (VariantLit + data-less SelectorExpr). Covered.
- AC2 (01-enums + 07-implements pass behavioral tier): backend_test
  TestASTEngineEnumsImplementsBehavioralTier via corpus.RunCompile. Covered.

No scope creep: only enum/sealed/implements + their construction are added;
match/Result/Option/?/derive remain unsupported-node errors (later stories).

No untested criteria.

# Plan Audit: Coverage — US-027

## Requirement → Plan trace

| Spec FR | Plan element |
|---------|--------------|
| FR-1 enums | `resolve.go` EnumDecl walker → Info.Enums; `Enum`/`Variant`/`Field` types |
| FR-2 structs | GenDecl(TYPE)+StructType walker → Info.Structs |
| FR-3 signatures | FuncDecl signature walker → Info.FuncSignatures; `FuncSig`/`Mode` |
| FR-4 from-registry | FuncDecl Mod=From/Derive walker → Info.FromRegistry; `ConvEntry` |
| FR-5 methods | FuncDecl Recv walker → Info.Methods; `Method` |
| FR-6 comma field | `typeString` over AST (no whitespace split) + `TestResolveStructCommaFieldType` |
| AC verify gates | unchanged build/vet/test surface; new test compiles in-module |

No plan element lacks a requirement. Sealed resolution is a small parity extra
(low risk, aids the FR-3/match stories later).

## Findings
- MINOR: `Sealed` is not in the named AC list but is cheap parity; keep.
- No CRITICAL/MAJOR.

## Assumptions
- Test compares specific symbols, not whole-map equality.
- Type strings compared modulo whitespace.

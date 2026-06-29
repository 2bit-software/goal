# SEAM-002 Acceptance Verification — FuncMod & ChanDir iota → goal enum

**Repo state:** `2c81404` on `main` ("refactor(selfhost): SEAM-002 FuncMod/ChanDir iota -> goal enum")
**Verdict:** PASS — all acceptance criteria hold. No gaps found.

## Acceptance Criteria

### AC-1: `selfhost/ast` declares FuncMod and ChanDir as goal enums — PASS
- `selfhost/ast/goal_decl.goal:20` — `enum FuncMod { FuncPlain; FuncFrom; FuncDerive }`
  (newline-separated tag-only variants; per-variant `//` doc text preserved, lines 21-26).
- `selfhost/ast/ast.goal:677` — `enum ChanDir { SendRecv; SendOnly; RecvOnly }`
  (doc text preserved, lines 678-683).
- Neither file retains `type FuncMod int` / `type ChanDir int` + iota (grep found only the `enum` forms).

### AC-2: No plain switch/==/!= over FuncMod or ChanDir remains; each is a match — PASS
Grep across `selfhost/` for `\.Mod ==`, `\.Mod !=`, `\.Dir ==`, `\.Dir !=`, `switch .*\.Mod`, `switch .*\.Dir`, `== ast.FuncMod`, `== ast.ChanDir` returns no enum-comparison/switch sites. The single hit (`selfhost/sema/resolve.goal:299 switch sig.Mode`) is `sema.Mode` (SEAM-003 scope), not FuncMod/ChanDir — the regex matched `.Mode` against `\.Mod`.
All consumers are exhaustive `match` with one arm per variant and no `_` wildcard:
- `selfhost/ast/ast.goal:170-173` — same-package match in `FuncDecl.Pos()`, uses unqualified `FuncMod.FuncPlain/FuncFrom/FuncDerive` (FR-3's required same-package conversion).
- `selfhost/sema/question.goal:214-216`, `selfhost/sema/convert.goal:38-40`, `selfhost/sema/resolve.goal:218-220` — FuncMod matches.
- `selfhost/sema/resolve.goal:464-466`, `selfhost/backend/emit.goal:2309-2311` — ChanDir matches.
- `selfhost/backend/emit.goal:368-370` — FuncMod match (the dropped `default: e.fail(...)` defensive arm is gone, as specified).

### AC-3 / FR-4: Construction uses qualified variant syntax; parser sets FuncPlain explicitly — PASS
- No bare unqualified `ast.FuncFrom` / `ast.SendRecv` (etc.) remain; all constructions are qualified.
- `selfhost/parser/parser.goal:367` — `fd := &ast.FuncDecl{Mod: ast.FuncMod.FuncPlain}` (with explanatory comment at :365 noting the enum zero value is nil, not FuncPlain). Spec referenced line 365; actual is 367 — benign line drift.
- `selfhost/parser/parser.goal:226,229` — `parseModFuncDecl(ast.FuncMod.FuncFrom/FuncDerive)`.
- `selfhost/parser/parser.goal:513,517,522` — `ast.ChanType{..., Dir: ast.ChanDir.SendRecv}`, with `RecvOnly`/`SendOnly` overrides; ChanType constructor sets Dir explicitly (zero-value gap closed for both enums).

### AC-4: token.Kind remains iota; DECISIONS.md records the refusal — PASS
- `selfhost/token/token.goal:17` — `type Kind int` with the iota `const (...)` block intact (line 21+).
- `DECISIONS.md` records the refusal: a dedicated subsection (line ~2063 "`FuncMod` and `ChanDir` stay iota... — NOT goal `enum`s" reflects the prior US-007 reasoning) plus the SEAM-002 section at line 2866 with the "token.Kind KEPT as iota (documented refusal, AC-1 escape hatch)" subsection at line 2897, citing numeric-identity / array-index / range-predicate justification.

### AC-5: DECISIONS.md records the conversion and the refusal — PASS
- `DECISIONS.md:2866` — "## SEAM-002 — FuncMod & ChanDir iota → goal enum (tree-wide); token.Kind kept as iota". Section documents what converted (FuncMod, ChanDir as §8.1 enums), the zero-value-gap fix in `parseFuncDecl`, qualified-construction supersession of old spellings, and the token.Kind iota refusal.

### AC-6: internal/ast test relocation — PASS
- `internal/ast/funcmod_test.go` exists (new, 3068 bytes), exercising FuncDecl Walk/Pos/Mod against the bootstrap reference compiler's Go-iota FuncMod (`FuncFrom`/`FuncDerive`/`FuncPlain` used directly — correct, since `internal/ast` deliberately stays Go iota per Out of Scope).
- `internal/ast/ast_test.go` no longer references FuncMod/ChanDir symbols — the only `FuncMod`/`ChanDir` hits (lines 247-248) are a comment pointing to the relocated test. The shared file is symbol-free so it compiles against both representations.
- `funcmod_test.go` is NOT referenced in `internal/selfhost/port_test.go` (grep empty).

### AC-7: prd.json SEAM-002 passes — PASS
- `prd.json:65` (within the SEAM-002 block at line 54) — `"passes": true`.

### AC-8: Behavior preserved — task check green / bootstrap shape — PASS
- `task check` is green: all packages `ok`, including the behavioral/fixpoint gates `internal/corpus` (TestASTEngineWholeCorpusBehavioralGate), `internal/selfhost`, and the relocated `internal/ast`.
- Bootstrap regeneration reflects the enum sealed-interface lowering:
  - `_bootstrap/fb/selfhost/ast/goal_decl.go:7` — `type FuncMod interface{ isFuncMod() }` with `FuncMod_FuncPlain/FuncFrom/FuncDerive` variant structs.
  - `_bootstrap/fb/selfhost/ast/ast.go:807` — `type ChanDir interface{ isChanDir() }` with the three variant structs.
- (Did not separately run `task build` / `task fixpoint` standalone; `task check` exercises the self-host fixpoint and corpus tiers via `internal/selfhost` and `internal/corpus`, both green.)

## Findings

No gaps. Highest severity: none (all criteria PASS).

Minor observations (informational, not defects):
- Spec line references drifted slightly (parser FuncPlain at :367 vs spec's :365); content is correct.
- The SEAM-002 prd.json story's title/ACs are phrased around `token.Kind` as the primary target; the realized work kept token.Kind as iota (documented refusal) and converted FuncMod/ChanDir, consistent with the business-spec under audit. This is intentional, not a discrepancy with the spec being verified.

## Assumptions

- The business-spec under `history/20260629-refactor-seam-002-.../business-spec.md` is the authoritative acceptance source; where the prd.json SEAM-002 wording emphasizes token.Kind, I treated the business-spec's "token.Kind stays iota, FuncMod/ChanDir convert" framing as the governing intent.
- "No wildcard `_` arm" was verified by inspecting the listed match sites; I relied on the grep-located arm lists (each shows exactly the named variants) rather than exhaustively reading every match body.
- `task check` is accepted as sufficient evidence for the `task build` / `task fixpoint` / corpus-tier criteria, since it drives the self-host and corpus packages; I did not invoke those Taskfile targets individually.
- Grep coverage assumed the enum types are only consumed via the spelled patterns (`.Mod`, `.Dir`, `ast.FuncMod.*`, `ast.ChanDir.*`, bare variant names); an alias or indirect reference would not have been caught, but none is expected in this tree.

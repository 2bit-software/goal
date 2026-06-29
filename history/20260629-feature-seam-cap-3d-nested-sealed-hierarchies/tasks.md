# Implementation Tasks

## Task 1: sema cascade
**Status**: completed
**Files**: internal/sema/resolve.go, selfhost/sema/resolve.goal
**Depends on**: none
**Spec coverage**: FR-2, FR-5
**Verify**: `go build ./internal/sema/...`

### Instructions
Add `func (info *Info) cascadeSealedImpls()`: for each sealed iface in
SealedImpls, walk info.EmbeddedIfaces transitively; for each transitively-
embedded iface that is also Sealed, addImplementor(embedded, impl) for each of
the iface's implementors. Snapshot SealedImpls keys before iterating. Call at the
tail of `Resolve(f)` (before return) and at the tail of `ResolvePackage` (after
the merge loop). Mirror line-for-line into selfhost/sema/resolve.goal.

## Task 2: backend cascade markers
**Status**: completed
**Files**: internal/backend/lower.go, internal/backend/emit.go, selfhost/backend/lower.goal, selfhost/backend/emit.goal
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-4, FR-5
**Verify**: `go build ./internal/backend/...`

### Instructions
Add `func sealedEmbeds(info *sema.Info, iface string) []string` in lower.go:
recursively collect transitively-embedded interfaces that are Sealed, source
order, deduped (nil-safe). In emit.go `implementsMarker`, after emitting the
sealed marker for `iface`, loop sealedEmbeds and emit `\n\n` + genMarkerMethod for
each. Mirror into selfhost/backend/{lower,emit}.goal.

## Task 3: regression tests + docs
**Status**: completed
**Files**: internal/backend/nested_sealed_test.go, internal/sema/nested_sealed_test.go, DECISIONS.md
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-1..FR-5, all acceptance criteria
**Verify**: `task check && task build && task fixpoint`

### Instructions
backend test (model on sealed_match_test.go): sealed A, sealed B embeds A,
concrete implementors at both levels (e.g. *Leaf implements A directly; *Node
implements B). Assert emitted Go contains markers `func (*Node) isB()` AND
`func (*Node) isNode()` style for the embedded level; transpile to temp module,
go test match-over-A and match-over-B vs reference type-switch. sema test: assert
SealedImpls contains the impl under both ifaces; non-exhaustive match over each
level is `non-exhaustive-match`. DECISIONS.md: record cascade design choice.

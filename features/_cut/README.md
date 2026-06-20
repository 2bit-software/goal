# `_cut` — audited but not used

Features in this directory were fully designed, spiked, and audited, then **cut before
v1**. They are kept here for the record — the spec, the reference transpiler spike, and the
worked examples are preserved intact — but they are **not** wired into the build, the unified
pipeline, the test suites, the docs, or the playground. Nothing here is reachable from
`go build ./...` (each spike carries its own `go.mod`), and no feature number is reused: a cut
feature's slot stays vacant in the active numbering.

If a cut feature is ever revived, move its directory back under `features/NN-name/` and re-wire
it; until then, treat everything here as a frozen artifact.

---

## 09-pure — lightweight `pure` annotation

**Cut: the value is real only with a piece we deliberately deferred, and the cheap version is a
guarantee that lies.**

`pure func` was a modifier marking a function side-effect-free. The reference transpiler only ever
**erased** it (`pure func` → `func`) and never verified anything; the actual effect-checking was
deferred to a "checker workstream." Auditing how that checker would have to work showed the feature
doesn't pay for itself in v1:

- **No definition existed.** Spec §4.2 punted the entire definition of "effect" to a checker that
  was never built — there was no algorithm, denylist, or rule to implement against.
- **The architecture can't check it soundly.** The transpiler is a token recognizer with no type
  information, no scope/escape analysis, and no cross-package resolution. It cannot tell local
  mutation from mutation through an aliased pointer/slice/map, cannot resolve a call's target
  package, and cannot see through interface dispatch. Sound purity checking is not buildable on it.
- **"Cheap" and "sound" are mutually exclusive.** A cheap syntactic denylist (`fmt`/`os`/…) is
  *unsound* — anything off the list passes, so a `pure` function that secretly does I/O through a
  third-party wrapper still "type-checks." An unenforced or unsound `pure` is *worse than nothing*:
  it's a guarantee-shaped marker that readers (human or model) will trust and that lies. Making it
  sound requires a real parse pass, an escape-analysis dodge (e.g. forbidding pointer/slice/map
  parameters), **and** a curated, per-Go-release FFI purity manifest for the stdlib boundary —
  none of which is "lightweight."
- **The transitivity / FFI boundary is the crux.** Purity is transitive: a `pure` function is pure
  only if every callee is. For goal code that means the marker is viral; for Go-library calls
  (`fmt.Println`, `strings.ToUpper`, `w.Write`) there is no marker at all, so the only sound option
  is a centrally-trusted purity manifest — not per-call-site "trust me" assertions, which would be a
  backdoor that defeats the guarantee.
- **The one concrete payoff is deferred.** The sole machine-usable benefit — auto-parallelization /
  memoization / reordering (§8.5) — was explicitly "not v1." So the cost and the benefit were both
  in the future and both bound to the expensive, sound version. The reasoning/documentation value
  that remained is marginal: an LLM already infers purity of a small leaf function as well as any
  conservative checker could.

**Decision:** remove `pure` from the v1 surface entirely rather than ship an unenforced, misleading
marker. Because the marker compiled to a no-op, removal costs nothing and retracts a promise we
couldn't keep. Reconsider it **together with** the §8.5 optimization backend — the only consumer
that turns purity into real value — at which point the sound-checker investment has something to pay
for it.

What lives here (`09-pure/`): `SYNTAX.md`, `TRANSPILE.md`, the standalone `transpiler/` spike (own
`go.mod`), and `examples/`.

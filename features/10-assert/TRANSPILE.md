# 10-assert — transpile to Go

Governing contract: §8.6 (runtime-preserved features) and §8.0.

## Erasure vs preservation (§8.0)

- **Runtime-preserved:** the assert check itself. Unlike the erased static guarantees (exhaustiveness,
  `implements`, field-completeness), an `assert` has **runtime semantics** — it must still
  panic on failure in the generated Go. So it lowers to a real `if !(cond) { panic(...) }` (§8.6),
  not to nothing.
- **Located feedback preserved into runtime:** the panic message carries the **source expression
  text**, so a failure says *which* invariant broke, verbatim — the located-feedback principle
  applied to a runtime failure (§8.6).
- **No defensive panic / no erasure** beyond surfacing: nothing here is proven unreachable; the panic
  is the assert's own semantics, not an unreachable-branch guard.

## Input → output pairs

### 1. Bare assert

```goal
assert amount > 0
```
```go
if !(amount > 0) {
    panic("assertion failed: amount > 0")
}
```

### 2. Printf-style message

```goal
assert age >= 0, "age must be non-negative, got %d", age
```
```go
if !(age >= 0) {
    panic("assertion failed: age >= 0: " + fmt.Sprintf("age must be non-negative, got %d", age))
}
```

…and the transpiler injects `import "fmt"` if the file does not already import it.

### 3. Mixed / internal commas / `%` in condition

```goal
assert lo <= hi
assert clamp(lo, hi, n) == n, "n=%d not in [%d, %d]", n, lo, hi
assert n%2 == 0
```
```go
if !(lo <= hi) {
    panic("assertion failed: lo <= hi")
}
if !(clamp(lo, hi, n) == n) {
    panic("assertion failed: clamp(lo, hi, n) == n: " + fmt.Sprintf("n=%d not in [%d, %d]", n, lo, hi))
}
if !(n%2 == 0) {
    panic("assertion failed: n%2 == 0")
}
```

## Lowering rules

The transpiler is a focused, statement-bounded recognizer (`text/scanner`, span-splice,
`go/format`); it does not parse full Go.

1. **Find each assert.** A token `assert` that is the **first token on its line** (statement
   position) is the keyword; `assert` elsewhere is an ordinary identifier and is left alone.
2. **Bound the statement.** Asserts are single-line: the statement runs from `assert` to the next
   newline (a trailing `;` is trimmed). Newlines are not tokens, so line bounds come from the source
   string.
3. **Split condition / message.** Scan to the **first top-level comma** (bracket depth 0, so commas
   inside calls/literals are ignored). Left of it = `cond`; right of it (if present) = the
   printf-style `msg` (`"fmt", args...`).
4. **Emit.**
   - Bare: `if !(cond) { panic(Q) }` where `Q = strconv.Quote("assertion failed: " + cond)`.
   - Message: `if !(cond) { panic(Q2 + fmt.Sprintf(msg)) }` where
     `Q2 = strconv.Quote("assertion failed: " + cond + ": ")`.
   - The expression text is always a **quoted string literal** (via `strconv.Quote`), never a format
     string — so a `%` in `cond` is literal and cannot accidentally consume a Sprintf verb.
5. **Inject `fmt`.** If any message-bearing assert was emitted and the file does not already import
   `"fmt"`, insert `import "fmt"` after the `package` clause. (gofmt keeps it as a valid separate
   import declaration.)

## Strategy forks

- **Bare vs message:** selected by the presence of a top-level comma (rule 3). Only the message form
  pulls in `fmt`.
- **Reserved — build-tag strip (NOT v1).** §8.6 wants asserts strippable from release builds via a
  build tag. The intended strategy: emit the assert bodies into a form a `noassert` build tag can
  compile out (e.g. asserts behind a `const assertsEnabled = true` the tag flips, or generated into
  a tagged companion). v1 **always emits** the check; this fork is documented, not implemented
  (NEXT-SESSION confirms stripping is not v1-critical).

## Hygiene

No `__goal_` temporaries are introduced — the lowering reuses the condition expression directly and
synthesizes only the literal message.

## Scope / not built

- **Statically-checkable subset** (§4.3) is reserved syntax, not built — every v1 assert is runtime
  runtime-checked. No static discharge, no Dafny-style proving.
- **Single-line conditions / statement position** assumed: a multi-line condition or an `assert`
  not at line start is out of scope (UB), consistent with the prior statement-bounded features
  (`?`).
- **Import injection is best-effort:** detected by scanning for a `"fmt"` import token. A file that
  imports `fmt` under a dot/named alias, or only inside a grouped block with unusual formatting,
  could double-import — acceptable for the reference (gofmt-valid), tightened by the real frontend.

# 11-doctests — transpile to Go

Governing contract: §8.6 (runtime-preserved / test-emitting features) and §4.1.

## Erasure vs preservation (§8.0)

- **Runtime-preserved, as a generated test:** the doctest is not erased and not inlined into the
  code path — it is **extracted into a separate `_test.go`** that executes under `go test` (§8.6).
  Turning the doctest into a real test is precisely what enforces §4.1's hard rule: **no way to
  silently not-run.**
- **Source comments preserved verbatim:** the original code file is unchanged. `///` is a valid Go
  line comment, so the doctest comments stay as documentation in the compiled code; a real build
  emits the source as-is **plus** the generated `<base>_doctest_test.go`.
- **No defensive panic:** failures surface as `t.Errorf`, the idiomatic located test failure.

## Two-output model

This feature emits **two** files from one `.goal` input:

1. the **code file** — the original source, unchanged (doc comments included); and
2. the **generated `_test.go`** — the extracted doctests.

The reference `transpile()` **returns file (2)** — the generated test file is this feature's actual
product, and the only part worth asserting. (File (1) is the input verbatim.) Accordingly each
example's `*.go.expected` holds the **generated test file**, and `main.go` writing `transpile()` to
stdout yields the `_test.go` (`go run . add.goal > add_doctest_test.go`).

## Input → output pairs

### 1. One doctest

```goal
/// Adds two ints.
/// >>> add(2, 3)
/// 5
func add(a int, b int) int { return a + b }
```
```go
package mathx

import "testing"

func TestDoctest_add_1(t *testing.T) {
	got := add(2, 3)
	want := 5
	if got != want {
		t.Errorf("doctest add: got %v, want %v", got, want)
	}
}
```

### 2. Multiple doctests → indexed test names

```goal
/// >>> repeat("ab", 2)
/// "abab"
/// >>> repeat("x", 3)
/// "xxx"
func repeat(s string, n int) string { … }
```
```go
func TestDoctest_repeat_1(t *testing.T) { got := repeat("ab", 2); want := "abab"; … }
func TestDoctest_repeat_2(t *testing.T) { got := repeat("x", 3);  want := "xxx";  … }
```

### 3. Prose-only doc → no test emitted

A `///` block with no `>>>` line generates nothing; only doctest-bearing functions produce tests.

## Lowering rules

The transpiler reads doc comments **straight from the source string** — the `text/scanner` lexer
skips comments, so token-based recognition can't see them.

1. **Package name.** Scan for the `package <name>` line; the generated file shares the package so it
   can call the functions under test.
2. **Find `///` blocks.** Walk lines; a maximal run of `///` lines is a doc block. Its content lines
   are the text after the leading whitespace and `///`.
3. **Attach to a function.** The next non-blank line after the block must be `func NAME(` (a free
   function); `NAME` names the tests. A block over a method (`func (r R) m(`) or a non-function decl
   attaches to nothing and is skipped (v1 scope).
4. **Parse pairs.** Within the block, each `>>> <expr>` line pairs with the **next** content line as
   the expected output. Lines without `>>>` are prose and ignored.
5. **Emit tests.** For the n-th doctest of `NAME`, emit:
   ```go
   func TestDoctest_NAME_n(t *testing.T) {
       got := <expr>
       want := <expected>
       if got != want { t.Errorf("doctest NAME: got %v, want %v", got, want) }
   }
   ```
   `import "testing"` is emitted only when at least one test exists (so a doctest-free file produces
   just a `package` clause, with no unused import).

## Strategy forks

None — a single extraction → `got`/`want` lowering. (Had the user chosen the inline-equality form,
the body would instead be `if !(<expr-with-==>) { t.Errorf(...) }`; the chosen next-line form maps to
the §8.6 `got`/`want` shape.)

## Hygiene

`got` / `want` are local to each generated test function, so no `__goal_` prefix is needed; test names
are namespaced by function name + index, so they never collide.

## Scope / not built

- **Expected output is a Go expression** (`5`, `"abab"`), not free-form printed text — it lowers to
  `want := <expected>`. No value parsing/printing layer (that would be needed for Python-style bare
  output) is built.
- **Comparison is `!=`.** Works for comparable types (the doctest examples). A doctest returning a
  non-comparable value (slice/map/func) won't compile under `got != want`; such cases need a
  `reflect.DeepEqual` variant — deferred.
- **Free functions only;** methods, multi-line expected output, doctests on non-function decls, and
  goscript's own runner (no `go test` there, §4.1) are out of scope for this Go-path audit.

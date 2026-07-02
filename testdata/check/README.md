# testdata/check — checker golden cases

One subdirectory per checker guarantee:

```
testdata/check/
  00-lex/             lexer-error cases
  01-enums/           enum-declaration cases
  08-no-zero-value/   field-completeness cases
  02-match/           match-exhaustiveness cases
  05-question-prop/   ?-propagation cases
  07-implements/      interface-satisfaction cases
  06-error-e/         closedness & From-totality cases
  12-derive-convert/  conversion-totality cases
  03-result/          must-use cases
  10-assert/          static-assert cases
```

The corpus manifest (built by `internal/corpus/generate.go`, which walks this
tree recursively) drives `RunCheck` (`internal/corpus/check_runner.go`), which
matches diagnostics against the `// want` markers below. Adding a `*.goal` case
here is auto-discovered — **it never touches the runner.**

## Inline expectation markers

Annotate the line a diagnostic is expected on with a `// want` marker:

```goal
return match s {            // want "missing variant Shape.Rect"
    Shape.Circle(c) => c.r
}
```

Rules:

- `// want "sub"` — some diagnostic **on that line** must contain the substring `sub`.
  Multiple markers per line are allowed.
- A `.goal` file with **no markers** is a clean/negative case: any `Error`-severity
  diagnostic anywhere fails it.
- Every `Error` diagnostic must be claimed by a marker on its line; an unclaimed
  `Error` fails the case. `Warning`s (located deferrals) may go unclaimed, so a
  "checker can't resolve this yet, but it's valid" case stays easy to write.

Write at least one positive case (a violation, with `// want`) and one negative case
(valid, no markers) per guarantee.

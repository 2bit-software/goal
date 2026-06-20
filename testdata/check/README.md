# testdata/check — checker golden cases

One subdirectory per guarantee, matching the `internal/check` slot:

```
testdata/check/
  08-no-zero-value/   field-completeness cases
  02-match/           match-exhaustiveness cases
  07-implements/      interface-satisfaction cases
  06-error-e/         closedness & From-totality cases
  12-derive-convert/  conversion-totality cases
  03-result/          must-use cases
  10-assert/          static-assert cases
```

The harness (`internal/check/check_test.go`, `TestCases`) auto-discovers every
`*.goal` file under here — **adding a case never touches the test file.**

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

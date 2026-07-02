This is the operating contract for the `goal` toolchain — the guarantees you can
rely on when driving the commands programmatically, without reading their output
as prose.

### Exit codes

Every `goal` command classifies its failure so you can triage without parsing the
message:

- **0** — success.
- **1** — a user-code diagnostic: checker findings, a syntax error, a `go build`
  failure of the transpiled Go, an interpreter runtime failure, or the program's
  own non-zero exit under `goal run`. Fix your `.goal` source.
- **2** — a usage error: an unknown subcommand, an unknown or malformed flag, or
  bad arguments. Fix your command line.
- **3** — an internal failure: a transpiler defect (generated Go that does not
  parse) or a build-overlay/toolchain setup failure not attributable to your
  code. Report it.

`goal run` propagates the running program's own exit code unchanged — it is never
reclassified into the 2/3 space.

### stdout vs stderr

The streams are split so machine output is never mixed with prose:

- **Diagnostics and prose** (checker findings, syntax errors, build-error lines,
  the `ok` line, notes) go to **stderr**.
- **`goal check --json`** writes its JSON array to **stdout** and nothing else.
- **`goalc`** writes the transpiled Go to **stdout**.
- **`goal run`** and **`goal test`** pass the program's / `go test`'s own output
  through unchanged.

So: read stdout for the artifact you asked for, read stderr for what went wrong.

### `goal check --json`

`goal check --json <pkg>` emits one JSON **array** to stdout — one object per
diagnostic, or `[]` when the package is clean. Each object has:

| field      | type   | meaning                                        |
|------------|--------|------------------------------------------------|
| `file`     | string | source file path as you passed it              |
| `line`     | int    | 1-based line                                   |
| `col`      | int    | 1-based column                                 |
| `severity` | string | `error` or `warning`                           |
| `code`     | string | stable bracketed code, e.g. `non-exhaustive-match` |
| `message`  | string | human-readable description                     |

A repairable diagnostic may also carry `suggestedFix` `{line, col, newText}` — a
pure insertion at that position. The field is omitted when no repair is known.
Both syntax errors and checker findings appear in the stream; the exit code is
unchanged by `--json` (1 when any error is present, 0 otherwise). `--json` is
rejected for `goal build` and `goal test`.

### Located text format

Without `--json`, every diagnostic — checker or syntax — renders as one line:

```
file:line:col: severity: [code] message
```

Syntax errors use the `[syntax]` code, e.g.
`main.goal:6:1: error: [syntax] expected ';'`. One regex captures every error
class, so you never special-case parse errors against checker errors.

### Source-line accuracy

Build errors and runtime panics report the **exact `.goal` statement line**, not
the enclosing declaration — the backend emits a per-statement `/*line*/` directive
before each source statement, so a lowered construct (`?`, `match`, a `...defaults`
expansion) never shifts the reported line off the statement that caused it.
Line-addressed edits from a build error or panic are exact.

### The doctest workflow

`goal test <pkg>` transpiles the package and runs its doctests via `go test`
through a build overlay — **ephemerally**, writing nothing to your source tree.
It exits 0 when all doctests pass, 1 when any fails. A failing doctest names the
`.goal` file and doctest line in its message, so you fix the source line directly.
(If you need the generated `_test.go` on disk — e.g. to run `go test` yourself —
use `goal build --emit` first.)

### Compact grammar of goal-only syntax

Everything below is goal's addition over Go; the rest of the language is Go.

```ebnf
EnumDecl     = "enum" Name "{" { Variant } "}" .
Variant      = Name [ "{" FieldList "}" ] .              (* e.g. Active { since: Time } *)

MatchExpr    = "match" Expr "{" { Arm } "}" .
Arm          = Pattern "=>" Body .                       (* newline-separated, no trailing comma *)
Pattern      = Enum "." Variant [ "(" bind ")" ] | "_" . (* "_" is the rest-arm *)

Question     = Expr "?" .                                (* postfix propagate on Result/Option *)

ResultType   = "Result" "[" T "," E "]" .                (* Result.Ok(x) / Result.Err(e) *)
OptionType   = "Option" "[" T "]" .                      (* Option.Some(x) / Option.None *)

DefaultsForm = "..." "defaults" .                        (* trailing rest in a composite literal *)
                                                         (* T{ a: x, ...defaults } *)

Implements   = "type" Name "struct" "implements" Iface "{" ... "}" .

FromFunc     = "from" "func" Signature Body .            (* registered total conversion *)
DeriveFunc   = "derive" "func" Signature [ Body ] .      (* bodyless = derive all fields *)
DeriveSpread = "..." "derive" "(" src ")" .              (* fill remaining target fields *)
                                                         (* a field set to _ is left zero *)

Doctest      = "///" ">>>" Expr NL "///" Expected .      (* doc-comment lines above a func *)
```

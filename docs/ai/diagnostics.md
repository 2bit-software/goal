The checker enforces each guarantee *before* lowering and emits located diagnostics in the form
`file:line:col: severity: [code] message`. An **Error** rejects the program before lowering
(unless `-nocheck`); a **Warning** is advisory — typically a lexical-stage deferral that the
typed "depth" stage resolves. Each diagnostic carries a stable `code` (greppable) and a remedy
in the message. The checker runs two stages: a lexical stage on the original source, and a typed
depth stage on the lowered Go via `go/types`; the typed finding wins when both flag the same
construct.

The full set of stable codes the checker can emit:

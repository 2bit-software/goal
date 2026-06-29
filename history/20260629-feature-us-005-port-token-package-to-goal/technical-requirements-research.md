# Technical Requirements / Research — US-005

- The goal compiler source is a Go superset, so the token package source is
  already valid goal. Porting = copying internal/token/token.go to
  selfhost/token/token.goal verbatim (it has zero internal deps).
- Reserved-word check: goal reserves the EXACT words `match`, `enum`, `assert`.
  token uses `ENUM` (uppercase constant) and `"enum"` (string/comment) only —
  no bare lowercase `enum` identifier — so no rename is needed.
- token exercises the US-001 iota const-block fix directly (Kind is an iota
  enum with bare continuation names and unexported beg/end range markers).
- Verification reuses the US-002 harness (internal/selfhost.BuildTranspiled).
  For criterion 3, transpile the ported package, write the generated Go plus
  the existing internal/token test file into a throwaway `module goal` temp
  module at internal/token, and run `go test`.
- Adding selfhost/token/*.goal is picked up by project.Discover during the
  bootstrap/fixpoint targets; both stages emit it identically so the fixpoint
  stays byte-identical, and the selfhost main package does not import it so
  `go build ./selfhost` is unaffected.

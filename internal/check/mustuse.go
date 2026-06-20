package check

import "goal/internal/analyze"

// checkMustUse enforces feature 03 (Result): the value of a Result-returning call must
// be used — consumed by `?`, matched, assigned and inspected, or explicitly discarded
// through the sanctioned discard surface. A Result dropped on the floor (a bare
// expression statement, or `_`-discarded without the explicit form) is an Error.
//
// Slot — not yet implemented. Returns no diagnostics until filled. See
// CHECKER-TODO.md (03-result) and prompt.md for the loop.
//
// Reuse, don't reinvent:
//   - analyze.Tables.FuncSignatures identifies which callees return a Result
//     (ModeResult / ModeResultClosed) — the must-use obligation attaches to their
//     call sites.
//   - The question pass (internal/pass/question.go) shows how a `?` consumes a Result
//     and how call sites are located.
//
// Defer-boundary: must-use is the most dataflow-shaped guarantee. Cover the local,
// statement-level cases (bare call statement; `:=` then unused) and defer anything
// needing real flow analysis (a Result stored, passed onward, then dropped) with a
// located Warning. This is the natural first candidate to graduate onto go/types if a
// lexical model proves too weak — note that boundary in DECISIONS.md if you hit it.
func checkMustUse(src string, t *analyze.Tables) ([]Diagnostic, error) {
	return nil, nil
}

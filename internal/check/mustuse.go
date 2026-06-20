package check

import (
	"fmt"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// checkMustUse enforces feature 03 (Result): the value of a Result-returning call must
// be used — consumed by `?`, matched, assigned and inspected, or explicitly discarded
// through the sanctioned discard surface. A Result dropped on the floor (a bare
// expression statement, or `_`-discarded without the explicit form) is an Error.
//
// Reuse, not reinvention:
//   - analyze.Tables.FuncSignatures identifies which callees return a Result
//     (ModeResult / ModeResultClosed): the must-use obligation attaches to a direct
//     call `f(...)` whose f has one of those modes. Option (ModeOption) and plain
//     (ModeNone) callees carry no obligation here.
//   - The question pass (internal/pass/question.go) shows how a `?` immediately follows
//     a Result call to consume it, and how a `match` / `:=` consume it instead. This
//     check mirrors that call-site walk but reads the *surrounding* tokens to decide
//     whether the value is consumed, instead of lowering.
//   - Must run pre-lowering: lowering rewrites a Result return into a native `(T,
//     error)` tuple and erases the `match`/`?`/return shapes this check reads.
//
// What it proves (the local, statement-level case — rock-solid lexically):
//   - A Result-returning direct call standing alone as an expression statement
//     (`parse(x)` on its own line, the close paren ending the statement) drops the
//     Result on the floor → Error `dropped-result`. The value is unreachable and the
//     error branch is silently discarded — exactly the silent-failure class Result
//     exists to kill (spec §3.2).
//
// What it recognizes as consumed (no diagnostic):
//   - `name := f(...)` / `name, … := f(...)` — bound to a named local (inspected-assign).
//   - `match f(...) { … }` — consumed by a match.
//   - `f(...)?` — consumed by `?` propagation.
//   - `return f(...)` — propagated as this function's own return value.
//   - `g(f(...))` / `xs[f(...)]` / any nested (non-statement-start) position — the call
//     is an argument/sub-expression, i.e. used in place.
//
// Defer-boundary (located Warning, never a false Error):
//   - `_ := f(...)` — a whole-Result discard. The sanctioned explicit-discard surface is
//     not yet defined (feature 03 SYNTAX.md §5 defers it together with this check), and
//     its lowering is unsettled, so this check will not yet rule on it →
//     `unresolved-result-discard`.
//   - A statement-leading call followed by a continuation (`f(...).x`, `f(...) + …`) this
//     lexical check does not resolve → `unresolved-result-use`.
//   - `name := f(...)` whose binding is then never read (a stored-then-dropped Result):
//     proving "never used" needs intra-function flow analysis the lexical model cannot do
//     soundly (closures, shadowing, reassignment), so the assigned-then-unused class is
//     deferred wholesale — this check rules only on the unconsumed *call site*, not on the
//     later fate of a bound value. Recorded as the go/types graduation boundary in
//     DECISIONS.md §03.
//
// No analyze.Tables extension was needed: FuncSignatures carries every fact this
// guarantee reads.
func checkMustUse(src string, t *analyze.Tables) ([]Diagnostic, error) {
	toks := scan.Lex(src)
	var diags []Diagnostic
	for i := 0; i+1 < len(toks); i++ {
		// A direct call is `IDENT (`; the callee must be an in-file Result-returning func.
		if !scan.IsIdent(toks[i].Text) || toks[i+1].Text != "(" {
			continue
		}
		sig, known := t.FuncSignatures[toks[i].Text]
		if !known || (sig.Mode != analyze.ModeResult && sig.Mode != analyze.ModeResultClosed) {
			continue
		}
		closeIdx := scan.MatchParen(toks, i+1)

		// A call standing alone as a statement is the rock-solid drop case.
		if isStatementStart(toks, i) {
			switch d, kind := classifyStatementCall(toks, i, closeIdx, toks[i].Text); kind {
			case stmtDropped, stmtDeferred:
				diags = append(diags, d)
			}
			// stmtConsumed (e.g. trailing `?`) adds nothing.
			continue
		}

		// A call on the RHS of a whole-Result discard `_ := f(...)` / `_ = f(...)`: the
		// sanctioned explicit-discard surface is not yet defined (feature 03 SYNTAX.md §5),
		// so this check defers rather than ruling it a use or a drop.
		if isUnderscoreDiscardRHS(toks, i) {
			diags = append(diags, Diagnostic{
				Pos:      toks[i].Start,
				Severity: Warning,
				Feature:  "03-result",
				Code:     "unresolved-result-discard",
				Message: fmt.Sprintf("cannot verify the `Result` from `%s(…)` is handled: it is discarded with `_ :=`, but the sanctioned explicit-discard surface for a `Result` is not yet defined — must-use deferred",
					toks[i].Text),
			})
		}
		// Any other nested / RHS / consuming position (named bind, `match` scrutinee,
		// `return` operand, argument) is a use in place — no obligation here.
	}
	return diags, nil
}

// statementCallKind classifies how a statement-leading Result call disposes of its value.
type statementCallKind int

const (
	stmtConsumed statementCallKind = iota // `?` — the value is propagated
	stmtDropped                           // nothing follows — dropped on the floor (Error)
	stmtDeferred                          // a shape this check won't yet rule on (Warning)
)

// isStatementStart reports whether the callee identifier at toks[i] begins a statement:
// the previous meaningful token is a statement boundary (`{`, `}`, `;`) or it is the
// first token. Every other predecessor (`return`, `match`, `=`, `,`, `(`, `[`, an
// operator, …) places the call in a consuming or nested position, which is a use, not a
// drop, so those calls carry no statement-level must-use obligation.
func isStatementStart(toks []scan.Token, i int) bool {
	if i == 0 {
		return true
	}
	switch toks[i-1].Text {
	case "{", "}", ";":
		return true
	default:
		return false
	}
}

// isUnderscoreDiscardRHS reports whether the call whose callee is toks[i] is the entire
// right-hand side of a whole-Result discard `_ := f(...)` or `_ = f(...)` — the LHS is a
// single `_` and the operator is `:=`/`=` immediately before the callee. (`:=` lexes as
// the two tokens `:` `=`.) An LHS with more than the bare `_` (e.g. `x, _ := …`) is an
// ordinary inspected-assign, not a whole-Result discard, and is left alone.
func isUnderscoreDiscardRHS(toks []scan.Token, i int) bool {
	if i < 2 || toks[i-1].Text != "=" {
		return false
	}
	switch {
	case toks[i-2].Text == "_":
		return true // `_ = f(...)`
	case i >= 3 && toks[i-2].Text == ":" && toks[i-3].Text == "_":
		return true // `_ := f(...)`
	}
	return false
}

// classifyStatementCall decides the fate of a statement-leading Result call whose callee
// is toks[i] and whose matching ")" is toks[closeIdx]. It reads only the token that
// follows the close paren:
//   - `?`            → consumed by propagation (no diagnostic).
//   - statement end  → dropped on the floor (Error `dropped-result`).
//   - anything else  → a chained / continued expression this lexical check won't follow
//     (`.method`, an operator, …); defer with a located Warning rather than risk a false
//     Error.
func classifyStatementCall(toks []scan.Token, i, closeIdx int, callee string) (Diagnostic, statementCallKind) {
	next := closeIdx + 1
	if next < len(toks) && toks[next].Text == "?" {
		return Diagnostic{}, stmtConsumed
	}
	// End of statement: the call is the whole statement (next is a closing brace, a
	// statement separator, or the source ends).
	if next >= len(toks) || toks[next].Text == "}" || toks[next].Text == ";" {
		return Diagnostic{
			Pos:      toks[i].Start,
			Severity: Error,
			Feature:  "03-result",
			Code:     "dropped-result",
			Message: fmt.Sprintf("the `Result` returned by `%s(…)` is dropped: a `Result` must be used — consume it with `match %s(…) { Result.Ok(v) => … Result.Err(e) => … }`, propagate it with `%s(…)?`, or bind it with `x := %s(…)`",
				callee, callee, callee, callee),
		}, stmtDropped
	}
	// The call is followed by something that continues the expression (a selector, an
	// operator, …). A Result has no usable surface besides match/?/bind, so this is
	// unusual; rather than guess whether the continuation consumes the value, defer.
	return Diagnostic{
		Pos:      toks[i].Start,
		Severity: Warning,
		Feature:  "03-result",
		Code:     "unresolved-result-use",
		Message: fmt.Sprintf("cannot verify the `Result` returned by `%s(…)` is used: it is followed by `%s`, a continuation this check does not resolve lexically — must-use deferred",
			callee, toks[next].Text),
	}, stmtDeferred
}

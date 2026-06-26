// Package fix rewrites plain-Go patterns inside .goal source into idiomatic goal — the
// inverse direction of the lowering passes. It is the engine behind `goal fix`: a function
// written as a Go `(T, error)` tuple with manual `if err != nil { return zero, err }`
// propagation becomes a `Result[T, error]` returning function whose body propagates with
// `?`; a `switch` over an in-file enum becomes a `match`.
//
// Like the lowering passes, fix is lexical: it lexes with scan, matches token patterns,
// and splices minimal edits — it never reflows untouched code. Every rule is conservative,
// when a candidate cannot be transformed safely the rule leaves it untouched and records a
// Report instead, so `goal fix` never emits incorrect code (the transpiler's `goal check`
// remains the correctness authority).
package fix

import (
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// Level classifies a Report: a Suggest is an opportunity fix chose not to apply, a Warn is
// an applied change with a caveat the caller should know about, a Skip is a detected
// candidate fix refused to touch because it could not prove the rewrite safe.
type Level int

const (
	Suggest Level = iota
	Warn
	Skip
)

func (l Level) String() string {
	switch l {
	case Warn:
		return "warning"
	case Skip:
		return "skipped"
	default:
		return "suggestion"
	}
}

// Change records one rewrite fix applied, located by 1-based line in the pre-edit source.
type Change struct {
	Line int
	Rule string
}

// Report records a candidate fix detected but did not (fully) rewrite, with a human
// message and the rule that produced it, located by 1-based line.
type Report struct {
	Line  int
	Level Level
	Rule  string
	Msg   string
}

// maxIters bounds the fixed-point loop. A rewrite in one pass (e.g. a signature becoming
// Result) can expose a rewrite in the next (its body's propagation becoming collapsible);
// the loop re-runs until nothing changes. The bound is a runaway guard, never reached in
// practice — the rules are monotone (each removes a pattern, none reintroduces one).
const maxIters = 8

// File rewrites one .goal file's source into idiomatic goal and returns the new source
// (== src when nothing changed), the changes applied, and reports for candidates that were
// detected but not safely rewritten. It runs every fixer to a fixed point so interdependent
// rewrites (signature conversion then body propagation) settle in one call.
func File(src string) (out string, changes []Change, reports []Report) {
	out = src
	seen := map[string]bool{}
	// add records a rule's reports, deduplicating by rule+message (not line, which shifts
	// as edits splice) so a stable condition reported on several passes appears once.
	add := func(rs []Report) {
		for _, r := range rs {
			key := r.Rule + "\x00" + r.Msg
			if !seen[key] {
				seen[key] = true
				reports = append(reports, r)
			}
		}
	}
	for range maxIters {
		t := analyze.Build(out)
		toks := scan.Lex(out)
		spans := analyze.FuncSpans(toks, t)

		var reps []scan.Replacement
		var iterReports []Report
		reps = append(reps, fixPropagate(out, toks, spans, t, &changes, &iterReports)...)
		reps = append(reps, fixResultSig(out, toks, t, &changes, &iterReports)...)
		reps = append(reps, fixSwitchToMatch(out, toks, t, &changes, &iterReports)...)
		add(iterReports)

		if len(reps) == 0 {
			// Converged. Every (T, error) function fix could convert is now Result-
			// returning, so the post-hoc call-site analysis sees only the genuinely
			// manual sites; run it once here, against the final source.
			var cs []Report
			reportCallSites(out, toks, spans, &cs)
			add(cs)
			break
		}
		out = scan.Splice(out, 0, len(out), reps)
	}
	return out, changes, reports
}

// lineOf returns the 1-based line number of byte offset off in src.
func lineOf(src string, off int) int {
	if off > len(src) {
		off = len(src)
	}
	return strings.Count(src[:off], "\n") + 1
}

// indentOf returns the leading run of spaces/tabs of the line starting at lineStart.
func indentOf(src string, lineStart int) string {
	end := lineStart
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	return src[lineStart:end]
}

// lineStartBefore returns the byte offset that begins the line containing off.
func lineStartBefore(src string, off int) int {
	return strings.LastIndexByte(src[:off], '\n') + 1
}

// spanHasComment reports whether src[lo:hi] contains a // or /* comment marker, so a rule
// removing a multi-statement region can refuse rather than silently drop the comment (the
// lexer skips comments, so they are invisible in the token stream).
func spanHasComment(src string, lo, hi int) bool {
	if lo < 0 || hi > len(src) || lo > hi {
		return false
	}
	s := src[lo:hi]
	return strings.Contains(s, "//") || strings.Contains(s, "/*")
}

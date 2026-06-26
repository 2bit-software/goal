package check

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// checkQuestion validates each `?` propagation site so a callee that cannot be propagated is
// rejected at check time with a located diagnostic, rather than silently lowered to Go that
// fails to compile. The `?` lowering (internal/pass/question.go) refuses the same provably
// broken callees, but a transpile failure only surfaces as a non-fatal `goal check` note —
// this check is what makes `goal check` itself reject them, with `file:line:col`.
//
// In an open-E (`Result[_, error]`) function a resolved plain/foreign `?` callee must return a
// trailing `error`: a void callee (nothing to propagate) and a non-error callee (its last
// result is not `error`) are Errors, and binding a value from a callee that does not return
// exactly `(value, error)` is an Error. When the callee cannot be resolved — a method whose
// receiver type is not inferred, or an import that did not load — a discarding `?` is a
// Warning (deferral): the lowering assumes the two-value form, which is wrong for an
// error-only callee, so the site is surfaced rather than silently guessed.
//
// A `?` whose callee is itself a `Result` function is fine (it ends in error by construction).
// Option `?` (the nil-check path) and closed-E `?` (feature 06, validated by checkClosed) are
// left to their own lowering/check.
func checkQuestion(src string, t *analyze.Tables) ([]Diagnostic, error) {
	toks := scan.Lex(src)
	spans := analyze.FuncSpans(toks, t)
	var diags []Diagnostic
	for q := range toks {
		if toks[q].Text != "?" {
			continue
		}
		p := toks[q].Start
		enc, ok := analyze.SigAt(spans, p)
		if !ok || enc.Mode != analyze.ModeResult {
			continue // only open-E Result context; Option/closed/non-Result handled elsewhere
		}
		lineStart := strings.LastIndexByte(src[:p], '\n') + 1
		name, rhs, isAssign := scan.SplitAssign(src[lineStart:p])
		if !isAssign && !scan.IsBareQuestionStmt(src, toks, q, lineStart) {
			diags = append(diags, Diagnostic{Pos: p, Severity: Error, Feature: "05-question-prop",
				Code:    "question-not-statement",
				Message: "`?` must be the right-hand side of an assignment (`name := expr?`) or a standalone `expr?` statement"})
			continue
		}
		discard := !isAssign || name == "_"
		key := scan.CalleeKey(rhs)
		csig, known := analyze.FuncSig{}, false
		if key != "" {
			csig, known = t.FuncSignatures[key]
		}

		// A resolved plain/foreign callee (no Result/Option mode of its own) is `?`-able here
		// only if it returns a trailing error. Result-mode callees end in error by construction.
		if known && csig.Mode == analyze.ModeNone {
			switch {
			case csig.Arity == 0:
				diags = append(diags, Diagnostic{Pos: p, Severity: Error, Feature: "05-question-prop",
					Code:    "question-callee-no-error",
					Message: fmt.Sprintf("`?` callee `%s` returns nothing; `?` needs a callee that returns a trailing `error` (or a `Result`)", key)})
			case !csig.EndsInError:
				diags = append(diags, Diagnostic{Pos: p, Severity: Error, Feature: "05-question-prop",
					Code:    "question-callee-no-error",
					Message: fmt.Sprintf("`?` callee `%s` does not return an `error` as its last result; `?` propagates an error", key)})
			case !discard && csig.Arity != 2:
				diags = append(diags, Diagnostic{Pos: p, Severity: Error, Feature: "05-question-prop",
					Code:    "question-binds-nonvalue",
					Message: fmt.Sprintf("`?` binds a value but `%s` returns %d value(s); write a bare `…?` to propagate only the error", key, csig.Arity)})
			}
			continue
		}

		// Unresolved callee in the discard form: the lowering falls back to the two-value
		// destructure, which is wrong for an error-only callee. Surface it rather than guess.
		if !known && discard {
			label := key
			if label == "" {
				label = strings.TrimSpace(rhs)
			}
			diags = append(diags, Diagnostic{Pos: p, Severity: Warning, Feature: "05-question-prop",
				Code:    "question-callee-unresolved",
				Message: fmt.Sprintf("cannot confirm the arity of `?` callee `%s` (an uninferred method receiver or an import that did not load); `?` assumes a two-value `(T, error)` callee — if it returns only `error`, the generated Go will not compile until its arity is resolved", label)})
		}
	}
	return diags, nil
}

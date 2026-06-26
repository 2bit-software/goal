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
	imports := importAliases(src)
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

		// In an open-E `Result[_, error]` function `?` propagates a plain `error`, so a resolved
		// callee is valid only if it yields a trailing error: a `Result[T, error]`, or a
		// plain/foreign function ending in `error`. An `Option`, a closed-E `Result`, a void,
		// or a non-error callee has no `error` to propagate.
		if known {
			switch {
			case csig.Mode == analyze.ModeResult:
				// ok: lowers to a trailing (T, error).
			case csig.Mode == analyze.ModeOption:
				diags = append(diags, Diagnostic{Pos: p, Severity: Error, Feature: "05-question-prop",
					Code:    "question-callee-no-error",
					Message: fmt.Sprintf("`?` in a `Result[_, error]` function propagates an `error`, but `%s` returns an `Option`; map its `None` to an error first", key)})
			case csig.Mode == analyze.ModeResultClosed:
				diags = append(diags, Diagnostic{Pos: p, Severity: Error, Feature: "05-question-prop",
					Code:    "question-callee-no-error",
					Message: fmt.Sprintf("`?` in an open-E `Result[_, error]` function propagates `error`, but `%s` returns a closed-E `Result`; convert its error to `error` first", key)})
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
		// destructure, which is wrong for an error-only callee. Surface only a genuinely opaque
		// callee — a method on a value whose receiver type isn't inferred. A call into an
		// imported package (`state.Write`, `pkg.Func`) is either resolved already or a sibling
		// goal package that lowers to `(T, error)`; warning on those is noise.
		if !known && discard && !isImportedCall(key, imports) {
			label := key
			if label == "" {
				label = strings.TrimSpace(rhs)
			}
			diags = append(diags, Diagnostic{Pos: p, Severity: Warning, Feature: "05-question-prop",
				Code:    "question-callee-unresolved",
				Message: fmt.Sprintf("cannot confirm the arity of `?` callee `%s` (its receiver type isn't inferred); `?` assumes a two-value `(T, error)` callee — if it returns only `error`, the generated Go will not compile until its arity is resolved", label)})
		}
	}
	return diags, nil
}

// importAliases returns the set of package qualifiers a file's imports bind (the explicit
// alias, or the import path's last segment when unaliased).
func importAliases(src string) map[string]bool {
	out := map[string]bool{}
	for _, imp := range analyze.ParseImports(src) {
		alias := imp.Alias
		if alias == "" {
			alias = imp.Path
			if i := strings.LastIndexByte(alias, '/'); i >= 0 {
				alias = alias[i+1:]
			}
		}
		out[alias] = true
	}
	return out
}

// isImportedCall reports whether key is a package-qualified call (`pkg.Func`) whose qualifier
// is one of the file's imports — as opposed to a method on a value (`recv.Method`).
func isImportedCall(key string, imports map[string]bool) bool {
	pkg, _, ok := strings.Cut(key, ".")
	return ok && imports[pkg]
}

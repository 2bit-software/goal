package guide

import "sort"

// diagDoc documents one checker diagnostic code for the guide. Code is the stable,
// greppable identifier the checker emits; Feature is the guarantee that produces it;
// Severity is "error" (rejects the program) or "warning" (advisory, usually a lexical
// deferral the typed stage resolves); Meaning is a one-line explanation.
type diagDoc struct {
	Code     string
	Feature  string
	Severity string
	Meaning  string
}

// diagnosticCatalog documents every stable diagnostic code the checker can emit. The
// set is held exactly to the codes present in internal/check and internal/typecheck —
// a test (TestDiagnosticCatalogMatchesSource) fails if the two diverge, so this catalog
// cannot silently fall out of sync with the checker.
var diagnosticCatalog = []diagDoc{
	// 02-match
	{"non-exhaustive-match", "02-match", "error", "a `match` on an enum omits a variant and has no `_` rest-arm."},
	{"unresolved-match-enum", "02-match", "warning", "the matched value's enum type can't be resolved lexically; deferred to the typed stage."},
	// 03-result
	{"dropped-result", "03-result", "error", "a `Result`-returning call's value is discarded instead of handled or propagated."},
	{"discarded-result-error", "03-result", "error", "the error arm of a `Result` is dropped (typed stage)."},
	{"dropped-stored-result", "03-result", "error", "a `Result` is stored as a plain value, defeating must-use handling."},
	{"unresolved-result-use", "03-result", "warning", "a `Result` use can't be resolved lexically; deferred to the typed stage."},
	{"unresolved-result-discard", "03-result", "warning", "a possible `Result` discard can't be resolved lexically; deferred."},
	{"unresolved-dropped-field", "03-result", "warning", "a possibly-dropped stored `Result` field can't be resolved lexically; deferred."},
	// 05-question-prop
	{"question-not-statement", "05-question-prop", "error", "`?` is neither the RHS of an assignment nor a standalone `expr?` statement."},
	{"question-callee-no-error", "05-question-prop", "error", "a `?` callee returns nothing or its last result isn't `error`, so there's no failure to propagate."},
	{"question-binds-nonvalue", "05-question-prop", "error", "`name := expr?` binds a value but the callee doesn't return exactly `(value, error)`."},
	{"question-callee-unresolved", "05-question-prop", "warning", "a discarding `?` callee's arity can't be resolved lexically; the two-value form is assumed."},
	// 06-error-e (closed-E Result)
	{"missing-from-conversion", "06-error-e", "error", "`?` bridges two error enums but no `from func` conversion is registered."},
	{"err-outside-closed-enum", "06-error-e", "error", "`Result.Err(...)` is used with a value outside the declared error enum."},
	{"unknown-error-variant", "06-error-e", "error", "a closed-E `Result` references an error variant the enum doesn't declare."},
	{"unresolved-error-enum", "06-error-e", "warning", "the error enum of a closed-E `Result` can't be resolved lexically; deferred."},
	{"unresolved-err-value", "06-error-e", "warning", "an `Err` value's type can't be resolved lexically; deferred."},
	{"unresolved-question-error", "06-error-e", "warning", "a `?` site's error type can't be resolved lexically; deferred."},
	// 07-implements
	{"unimplemented-method", "07-implements", "error", "a `struct implements I` is missing a method required by interface `I`."},
	{"method-signature-mismatch", "07-implements", "error", "a method exists but its signature doesn't satisfy the declared interface."},
	{"unresolved-interface", "07-implements", "warning", "the named interface of an `implements` clause can't be resolved lexically; deferred."},
	// 08-no-zero-value
	{"missing-field", "08-no-zero-value", "error", "a struct literal omits a required field and has no `...defaults`."},
	{"generic-missing-field", "08-no-zero-value", "error", "a generic struct literal omits required fields (typed stage)."},
	{"elided-missing-field", "08-no-zero-value", "error", "an elided literal (type inferred from a collection) omits required fields (typed stage)."},
	{"unresolved-literal-type", "08-no-zero-value", "warning", "a struct literal's type can't be resolved lexically; deferred to the typed stage."},
	// 10-assert
	{"assert-always-false", "10-assert", "error", "an `assert` condition is constantly false — it would always panic."},
	{"assert-always-true", "10-assert", "warning", "an `assert` condition is constantly true — the assert is dead."},
	// 12-derive-convert
	{"unsourced-field", "12-derive-convert", "error", "a `derive func` target field has no source (registry leaf, recursion, expr, or `_`)."},
	{"unbridged-field", "12-derive-convert", "error", "a `derive func` field needs a conversion that isn't in the `from func` registry."},
	{"fallible-in-total-derive", "12-derive-convert", "error", "a total `derive func` sources a field from a fallible conversion."},
	{"unresolved-derive-type", "12-derive-convert", "warning", "a `derive func` type pair can't be resolved lexically; deferred to the typed stage."},
	{"unresolved-derive-field", "12-derive-convert", "warning", "a `derive func` target field can't be resolved lexically; deferred."},
}

// catalogByFeature returns the diagnostic codes grouped by feature, with features and
// codes in a stable order, so the rendered catalog is deterministic.
func catalogByFeature() []struct {
	Feature string
	Codes   []diagDoc
} {
	order := []string{}
	seen := map[string]bool{}
	byFeat := map[string][]diagDoc{}
	for _, d := range diagnosticCatalog {
		if !seen[d.Feature] {
			seen[d.Feature] = true
			order = append(order, d.Feature)
		}
		byFeat[d.Feature] = append(byFeat[d.Feature], d)
	}
	sort.Strings(order)
	var out []struct {
		Feature string
		Codes   []diagDoc
	}
	for _, f := range order {
		codes := byFeat[f]
		sort.Slice(codes, func(i, j int) bool { return codes[i].Code < codes[j].Code })
		out = append(out, struct {
			Feature string
			Codes   []diagDoc
		}{f, codes})
	}
	return out
}

// catalogCodes returns the set of diagnostic codes the catalog documents.
func catalogCodes() map[string]bool {
	out := make(map[string]bool, len(diagnosticCatalog))
	for _, d := range diagnosticCatalog {
		out[d.Code] = true
	}
	return out
}

package corpus

import (
	"goal/internal/check"
	"goal/internal/parser"
	"goal/internal/sema"
)

// SemaCheck is the AST-based checker the corpus judges against the same inline
// // want markers as the lexical check.Analyze. It parses src with the new
// front-end, resolves its semantic facts, runs the sema checks, and converts the
// sema diagnostics to check.Diagnostic so the shared [Checker] seam (and the
// corpus check runner) can consume it unchanged.
//
// As later sema stories (US-030/031) extend sema.Check, this adapter automatically
// carries their diagnostics too — it is the single seam where the AST checker plugs
// into the corpus. SemaCheck satisfies [Checker] via [CheckerFunc].
func SemaCheck(src string) ([]check.Diagnostic, error) {
	file, err := parser.ParseFile(src)
	if err != nil {
		return nil, err
	}
	info := sema.Resolve(file)
	sdiags := sema.Check(file, info)

	diags := make([]check.Diagnostic, len(sdiags))
	for i, d := range sdiags {
		diags[i] = check.Diagnostic{
			Pos:      d.Pos.Offset,
			Severity: toCheckSeverity(d.Severity),
			Feature:  d.Feature,
			Code:     d.Code,
			Message:  d.Message,
		}
	}
	return diags, nil
}

// toCheckSeverity maps a sema severity onto the corresponding check severity. The
// two enums share constant order, but the mapping is explicit so a future reorder
// of either cannot silently mistranslate a rejection into an advisory.
func toCheckSeverity(s sema.Severity) check.Severity {
	if s == sema.Warning {
		return check.Warning
	}
	return check.Error
}

// Package pass holds the per-construct lowering passes. Each pass has the signature
//
//	func(src string, t *analyze.Tables) (string, error)
//
// re-lexes the current source, splices its construct, and returns the new source
// WITHOUT formatting. The pipeline driver formats once after the last pass; an
// intermediate source need not be gofmt-parseable, only lexable.
//
// Passes are ordered so that signature/type lowering precedes the control-flow
// lowering that depends on it (see package pipeline). A pass reads cross-pass facts
// it can no longer see in the lowered source from the name-keyed analyze.Tables.
package pass

import (
	"goal/internal/analyze"
	"goal/internal/scan"
)

// Hygienic names for synthesized temporaries and named returns (spec §8 prefix).
// Shared across passes so the signature rewrite (result) and the `?` lowering
// (question) agree on the same identifiers.
const (
	okName   = "__goal_ok"   // named success return / Ok binding target
	errName  = "__goal_err"  // named error return / Err binding target
	valName  = "__goal_v"    // Ok value captured at a match site
	optBase  = "__goal_o"    // base for `?`-Option temporaries (suffixed with a counter)
	someName = "__goal_some" // boxed Some value when the payload isn't addressable
	evar     = "__goal_e"    // type-switch guard at a closed-E match/? site
)

// funcSpan, funcSpans, and sigAt now live in package analyze (FuncSpan/FuncSpans/SigAt)
// so the `goal fix` rewriter can share them; these thin aliases keep the passes' existing
// call sites unchanged.
type funcSpan = analyze.FuncSpan

func funcSpans(toks []scan.Token, t *analyze.Tables) []analyze.FuncSpan {
	return analyze.FuncSpans(toks, t)
}

func sigAt(spans []analyze.FuncSpan, off int) (analyze.FuncSig, bool) {
	return analyze.SigAt(spans, off)
}

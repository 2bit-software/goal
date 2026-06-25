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

// funcSpan pairs a function body's current byte span with its analyzed signature, so
// a pass that has lost the original return type from the lowered source can recover
// the enclosing function's mode and T/E types by re-scanning and looking up the name.
// Several passes need this (open `?`, closed `?`/match/construction), so centralizing
// it keeps the open/closed-E discrimination in one place.
type funcSpan struct {
	lo, hi int
	sig    analyze.FuncSig
}

// funcSpans returns one span per function in the current source, carrying its
// analyzed signature from the name-keyed tables (functions without a recorded
// signature are omitted).
func funcSpans(toks []scan.Token, t *analyze.Tables) []funcSpan {
	var spans []funcSpan
	for _, f := range scan.ScanFuncs(toks) {
		if sig, ok := t.FuncSignatures[f.Name]; ok {
			spans = append(spans, funcSpan{lo: toks[f.BodyOpen].Start, hi: toks[f.BodyClose].End, sig: sig})
		}
	}
	return spans
}

// sigAt returns the signature of the function whose body contains byte offset off.
func sigAt(spans []funcSpan, off int) (analyze.FuncSig, bool) {
	for _, s := range spans {
		if off >= s.lo && off < s.hi {
			return s.sig, true
		}
	}
	return analyze.FuncSig{}, false
}

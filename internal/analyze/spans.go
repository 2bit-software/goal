package analyze

import (
	"goal/internal/scan"
)

// FuncSpan pairs a function body's current byte span with its analyzed signature, so a
// caller that has lost the original return type from lowered (or plain-Go) source can
// recover the enclosing function's mode and T/E types by re-scanning and looking up the
// name. The lowering passes and the `goal fix` rewriter both consult it, so it lives here
// in the name-keyed analysis package rather than in any one pass.
type FuncSpan struct {
	Lo, Hi int
	Sig    FuncSig
}

// FuncSpans returns one span per function in the current source, carrying its analyzed
// signature from the name-keyed tables (functions without a recorded signature are
// omitted).
func FuncSpans(toks []scan.Token, t *Tables) []FuncSpan {
	var spans []FuncSpan
	for _, f := range scan.ScanFuncs(toks) {
		if sig, ok := t.FuncSignatures[f.Name]; ok {
			spans = append(spans, FuncSpan{Lo: toks[f.BodyOpen].Start, Hi: toks[f.BodyClose].End, Sig: sig})
		}
	}
	return spans
}

// SigAt returns the signature of the function whose body contains byte offset off.
func SigAt(spans []FuncSpan, off int) (FuncSig, bool) {
	for _, s := range spans {
		if off >= s.Lo && off < s.Hi {
			return s.Sig, true
		}
	}
	return FuncSig{}, false
}

// Package textedittest holds Go-authored tests for internal/textedit that the
// golden corpus cannot express — chiefly the Splice conflict model: when two
// replacements overlap, which one wins (deterministic (Start, End) order) and
// that the loser is returned in the dropped slice rather than silently lost.
//
// It lives in its own package (not internal/textedit) because internal/textedit
// is generated from textedit.goal and guarded by the verify-generated drift
// gate, which a hand-written _test.go beside the generated files would trip.
package textedittest

// Package analyze builds the name-keyed symbol tables that survive re-lexing.
//
// The pipeline splices bytes, so byte offsets shift between passes and any table
// keyed by offset would be invalid after the first edit. Tables here are therefore
// keyed by symbol name (function name, type name): lowering preserves those names —
// a Result signature rewrite keeps the function name, an enum decl keeps its type
// name — so a table built once from the original source stays valid for every pass.
//
// Tables are built once, before any pass runs, and are read-only to passes.
package analyze

import (
	"strings"

	"goal/internal/scan"
)

// Mode is a function's error-propagation shape, read from its return type.
type Mode int

const (
	// ModeNone is a function that returns neither Result nor Option.
	ModeNone Mode = iota
	// ModeResult is a function returning Result[T, E].
	ModeResult
	// ModeOption is a function returning Option[T].
	ModeOption
)

// FuncSig is the analyzed return signature of one function.
type FuncSig struct {
	Name string
	Mode Mode
	T    string // success type (the T in Result[T, E] or Option[T])
	E    string // error type (the E in Result[T, E]); "" for Option/none
}

// Tables holds every name-keyed table the passes consult. It is built once from the
// original source and never mutated by a pass.
type Tables struct {
	// FuncSignatures maps a function name to its analyzed return signature. Passes
	// that have already lowered a signature (Result -> named returns, Option -> *T)
	// can no longer read the original mode from the source, so they recover it here.
	FuncSignatures map[string]FuncSig
}

// Build analyzes the original source and returns the populated tables.
func Build(src string) *Tables {
	toks := scan.Lex(src)
	t := &Tables{FuncSignatures: map[string]FuncSig{}}
	for _, f := range scan.ScanFuncs(toks) {
		if f.Name == "" {
			continue
		}
		t.FuncSignatures[f.Name] = analyzeSig(src, toks, f)
	}
	return t
}

// analyzeSig reads the return mode and type parameters of one function from its
// (un-lowered) signature.
func analyzeSig(src string, toks []scan.Token, f scan.Func) FuncSig {
	sig := FuncSig{Name: f.Name, Mode: ModeNone}
	pc := f.ParamsClose
	if pc < 0 || pc+2 >= f.BodyOpen {
		return sig
	}
	switch {
	case toks[pc+1].Text == "Result" && toks[pc+2].Text == "[":
		rb := scan.MatchBracket(toks, pc+2)
		comma := scan.TopLevelComma(toks, pc+2, rb)
		if comma < 0 {
			return sig
		}
		sig.Mode = ModeResult
		sig.T = strings.TrimSpace(src[toks[pc+3].Start:toks[comma].Start])
		sig.E = strings.TrimSpace(src[toks[comma+1].Start:toks[rb].Start])
	case toks[pc+1].Text == "Option" && toks[pc+2].Text == "[":
		rb := scan.MatchBracket(toks, pc+2)
		sig.Mode = ModeOption
		sig.T = strings.TrimSpace(src[toks[pc+3].Start:toks[rb].Start])
	}
	return sig
}

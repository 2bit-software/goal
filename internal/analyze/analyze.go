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
	// ModeResult is a function returning the open-E Result[T, error]: E is the
	// builtin error, lowered to the native (T, error) tuple (feature 03).
	ModeResult
	// ModeResultClosed is a function returning a closed-E Result[T, E] where E is not
	// error, lowered to the generic sum encoding Ok[T,E]/Err[T,E] (feature 06).
	ModeResultClosed
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

// Field is one variant field: the goal field name (lowercase) and its type
// expression captured verbatim from source.
type Field struct {
	Name string
	Type string
}

// Variant is one enum variant; Fields is empty for a data-less variant.
type Variant struct {
	Name   string
	Fields []Field
}

// Enum is a closed sum type: its variants plus name/field membership sets the match
// and construction lowering consult.
type Enum struct {
	Name     string
	Variants []Variant
	VSet     map[string]bool            // variant-name set
	FieldSet map[string]map[string]bool // variant -> its field-name set
}

// Tables holds every name-keyed table the passes consult. It is built once from the
// original source and never mutated by a pass.
type Tables struct {
	// FuncSignatures maps a function name to its analyzed return signature. Passes
	// that have already lowered a signature (Result -> named returns, Option -> *T)
	// can no longer read the original mode from the source, so they recover it here.
	FuncSignatures map[string]FuncSig
	// Enums maps an enum type name to its analyzed variants. Read by the enums pass
	// (encoding + construction) and the match pass (field-access exporting).
	Enums map[string]*Enum
	// Sealed is the set of interface names declared `sealed interface`. It
	// disambiguates `implements I for T`: a marker method when I is sealed (feature
	// 01), versus a compile-time assertion otherwise (feature 07).
	Sealed map[string]bool
	// Structs maps a `type X struct {…}` name to its ordered fields. Read by the
	// defaults pass (`...defaults` expansion) and feature 12's derive.
	Structs map[string][]Field
	// TypeDecls maps a type name to its underlying form: "struct", "interface", or a
	// type expression (alias target / defined-type underlying). Read by the defaults
	// pass to recover a field's zero value through alias chains.
	TypeDecls map[string]string
	// FromRegistry maps a (source error type, destination error type) pair to the
	// `from func` that converts between them. Read by the closed-E pass to insert the
	// From-conversion when `?` crosses error types, and by feature 12's derive.
	FromRegistry map[[2]string]string
}

// Build analyzes the original source and returns the populated tables.
func Build(src string) *Tables {
	toks := scan.Lex(src)
	t := &Tables{
		FuncSignatures: map[string]FuncSig{},
		Enums:          map[string]*Enum{},
		Sealed:         map[string]bool{},
		Structs:        map[string][]Field{},
		TypeDecls:      map[string]string{},
		FromRegistry:   map[[2]string]string{},
	}
	for _, f := range scan.ScanFuncs(toks) {
		if f.Name == "" {
			continue
		}
		t.FuncSignatures[f.Name] = analyzeSig(src, toks, f)
	}
	analyzeFromFuncs(src, toks, t)
	for i := 0; i+1 < len(toks); i++ {
		switch {
		case toks[i].Text == "enum":
			e := analyzeEnum(src, toks, i)
			t.Enums[e.Name] = e
		case toks[i].Text == "sealed" && toks[i+1].Text == "interface" && i+2 < len(toks):
			t.Sealed[toks[i+2].Text] = true
		}
	}
	analyzeTypeDecls(src, toks, t)
	return t
}

// analyzeTypeDecls scans top-level `type` declarations, populating Structs (ordered
// fields of every `type X struct {…}`) and TypeDecls (name -> underlying form).
func analyzeTypeDecls(src string, toks []scan.Token, t *Tables) {
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "type" || !scan.IsIdent(toks[i+1].Text) {
			continue
		}
		name := toks[i+1].Text
		switch toks[i+2].Text {
		case "=":
			t.TypeDecls[name] = restOfLine(src, toks[i+3].Start)
		case "struct":
			t.TypeDecls[name] = "struct"
			open := indexOf(toks, i+2, "{")
			if open >= 0 {
				closeIdx := scan.MatchBrace(toks, open)
				t.Structs[name] = parseStructBody(src[toks[open].End:toks[closeIdx].Start])
			}
		case "interface":
			t.TypeDecls[name] = "interface"
		default:
			t.TypeDecls[name] = restOfLine(src, toks[i+2].Start)
		}
	}
}

// parseStructBody parses the text between a struct's braces into ordered fields. One
// field per line (or `;`-separated); `a, b int` yields two fields.
func parseStructBody(body string) []Field {
	var fields []Field
	for _, raw := range strings.FieldsFunc(body, func(r rune) bool { return r == '\n' || r == ';' }) {
		line := raw
		if c := strings.Index(line, "//"); c >= 0 {
			line = line[:c]
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // blank, or an embedded type (unsupported here) — skip
		}
		typ := parts[len(parts)-1]
		for _, nm := range parts[:len(parts)-1] {
			nm = strings.TrimSuffix(nm, ",")
			if nm != "" {
				fields = append(fields, Field{Name: nm, Type: typ})
			}
		}
	}
	return fields
}

// restOfLine returns the source from offset to the next newline, trimmed and with a
// leading "=" stripped so an alias `type X = Y` yields just `Y`.
func restOfLine(src string, offset int) string {
	end := len(src)
	if nl := strings.IndexByte(src[offset:], '\n'); nl >= 0 {
		end = offset + nl
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(src[offset:end]), "="))
}

// indexOf returns the index of the first token with text txt at or after `from`.
func indexOf(toks []scan.Token, from int, txt string) int {
	for k := from; k < len(toks); k++ {
		if toks[k].Text == txt {
			return k
		}
	}
	return -1
}

// analyzeFromFuncs records each `from func NAME(p SRC) DST` conversion in the From
// registry, keyed by (SRC, DST). The `from` modifier is stripped by the closed-E pass.
func analyzeFromFuncs(src string, toks []scan.Token, t *Tables) {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "from" || toks[i+1].Text != "func" {
			continue
		}
		fi := i + 1
		bo := scan.FirstBodyBrace(toks, fi)
		if bo < 0 {
			continue
		}
		pc := scan.ParamsClose(toks, bo)
		if pc < 0 || fi+3 >= pc {
			continue
		}
		name := toks[fi+1].Text
		srcType := strings.TrimSpace(src[toks[fi+3].End:toks[pc].Start])
		dstType := strings.TrimSpace(src[toks[pc].End:toks[bo].Start])
		t.FromRegistry[[2]string{srcType, dstType}] = name
	}
}

// analyzeEnum parses `enum NAME { variant... }` starting at toks[i] == "enum".
func analyzeEnum(src string, toks []scan.Token, i int) *Enum {
	e := &Enum{
		Name:     toks[i+1].Text,
		VSet:     map[string]bool{},
		FieldSet: map[string]map[string]bool{},
	}
	k := i + 3 // step past `enum NAME {`
	for k < len(toks) && toks[k].Text != "}" {
		vname := toks[k].Text
		k++
		var fields []Field
		if k < len(toks) && toks[k].Text == "{" {
			fields, k = parseFields(src, toks, k+1)
			k++ // consume the variant's closing "}"
		}
		e.Variants = append(e.Variants, Variant{Name: vname, Fields: fields})
		e.VSet[vname] = true
		set := map[string]bool{}
		for _, f := range fields {
			set[f.Name] = true
		}
		e.FieldSet[vname] = set
	}
	return e
}

// parseFields parses `name: Type, name: Type` up to the closing "}", starting at the
// first field name. Type expressions are captured verbatim, honoring nested
// () [] {} so map/func/struct/slice types survive intact.
func parseFields(src string, toks []scan.Token, k int) ([]Field, int) {
	var fields []Field
	for k < len(toks) && toks[k].Text != "}" {
		name := toks[k].Text
		k += 2 // skip name and ":"
		typeStart := toks[k].Start
		typeEnd := toks[k].End
		depth := 0
		for k < len(toks) {
			t := toks[k]
			if depth == 0 && (t.Text == "," || t.Text == "}") {
				break
			}
			switch t.Text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			typeEnd = t.End
			k++
		}
		fields = append(fields, Field{Name: name, Type: strings.TrimSpace(src[typeStart:typeEnd])})
		if k < len(toks) && toks[k].Text == "," {
			k++
		}
	}
	return fields, k
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
		sig.T = strings.TrimSpace(src[toks[pc+3].Start:toks[comma].Start])
		sig.E = strings.TrimSpace(src[toks[comma+1].Start:toks[rb].Start])
		if sig.E == "error" {
			sig.Mode = ModeResult // open-E: native (T, error)
		} else {
			sig.Mode = ModeResultClosed // closed-E: Ok[T,E]/Err[T,E] sum
		}
	case toks[pc+1].Text == "Option" && toks[pc+2].Text == "[":
		rb := scan.MatchBracket(toks, pc+2)
		sig.Mode = ModeOption
		sig.T = strings.TrimSpace(src[toks[pc+3].Start:toks[rb].Start])
	}
	return sig
}

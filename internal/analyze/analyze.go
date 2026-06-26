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
	"maps"
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
	Name  string
	Mode  Mode
	T     string // success type (the T in Result[T, E] or Option[T])
	E     string // error type (the E in Result[T, E]); "" for Option/none
	Arity int    // number of values the function returns at `?`-lowering time; 0 if unknown/void
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
	// FromRegistry maps a (source type, target type) pair to the `from func` that
	// converts between them. Read by the closed-E pass for the `?` From-conversion
	// (error types) and by the derive pass for field-by-field conversion (any types).
	FromRegistry map[[2]string]ConvEntry
	// Interfaces maps an in-file `type I interface { … }` name to its declared method
	// set (name + normalized signature). Read by the implements check (feature 07) to
	// know what an `implements I` clause obligates. Empty for a marker (sealed)
	// interface. Embedded interfaces are recorded under EmbeddedIfaces.
	Interfaces map[string][]Method
	// EmbeddedIfaces maps an in-file interface name to the interface names it embeds
	// (a bare type name on its own line inside the body, e.g. `io.Reader`). The
	// implements check folds an in-file embedded interface's methods into the
	// obligation and defers when an embedded interface is not resolvable in-file.
	EmbeddedIfaces map[string][]string
	// Methods maps a concrete type name to the methods declared on it in this file
	// (value- and pointer-receiver alike), name + normalized signature. Read by the
	// implements check to decide whether a type satisfies an interface's obligation.
	Methods map[string][]Method

	// SuppressResultPrelude tells the closed-E Result pass NOT to inject the generic
	// Ok[T,E]/Err[T,E] preamble inline. The package transpile driver sets it so the
	// prelude is emitted once per package (a single goal_prelude.go) instead of once per
	// file; single-file Transpile leaves it false and the pass injects inline exactly as
	// before. This is a deliberate driver directive — the one field on Tables that is not
	// a name-keyed source fact — kept here so it threads through the existing pass
	// signature without widening it.
	SuppressResultPrelude bool
}

// ConvEntry is one `from func` conversion: its name and whether it is fallible (the
// return type is `(T, error)` rather than a bare T).
type ConvEntry struct {
	Name     string
	Fallible bool
}

// Method is one method of an interface or a concrete type: its name plus a normalized
// signature (the parameter and result *type* lists with parameter names and whitespace
// stripped) so an interface obligation and a concrete method can be compared for
// equality. Sig is the canonical text `(paramTypes) resultTypes`; Raw keeps the
// original parameter+result text for diagnostics.
type Method struct {
	Name string
	Sig  string
	Raw  string
}

// Build analyzes the original source and returns the populated tables.
// newTables returns an empty Tables with every name-keyed map initialized.
func newTables() *Tables {
	return &Tables{
		FuncSignatures: map[string]FuncSig{},
		Enums:          map[string]*Enum{},
		Sealed:         map[string]bool{},
		Structs:        map[string][]Field{},
		TypeDecls:      map[string]string{},
		FromRegistry:   map[[2]string]ConvEntry{},
		Interfaces:     map[string][]Method{},
		EmbeddedIfaces: map[string][]string{},
		Methods:        map[string][]Method{},
	}
}

func Build(src string) *Tables {
	toks := scan.Lex(src)
	t := newTables()
	for _, f := range scan.ScanFuncs(toks) {
		if f.Name == "" {
			continue
		}
		t.FuncSignatures[f.Name] = analyzeSig(src, toks, f)
	}
	analyzeFromFuncs(src, toks, t)
	analyzeMethods(src, toks, t)
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

// BuildPackage builds one set of name-keyed tables for a whole package by analyzing
// each file's source independently and unioning the results. Because the tables are
// position-free and keyed by symbol name, the union lets a pass running over one file
// resolve symbols — enums, structs, `from func`s, signatures — declared in any sibling
// file, with no file needing another's text. Pass sources in a stable order (e.g.
// project.Discover's path-sorted files) so the merge is deterministic.
func BuildPackage(srcs []string) *Tables {
	t := newTables()
	for _, src := range srcs {
		t.Merge(Build(src))
	}
	return t
}

// Merge unions o into t: every name-keyed entry from o is copied in. On a key present
// in both, o wins (last-merged-wins) — callers supply sources in a stable order, so
// this is deterministic. A genuine duplicate declaration (the same function or type
// name in two files) is a Go redeclaration error the Go compiler reports; analyze does
// not re-implement that check, it just keeps the last definition so lowering can
// proceed and the real error surfaces at `go build`.
func (t *Tables) Merge(o *Tables) {
	maps.Copy(t.FuncSignatures, o.FuncSignatures)
	maps.Copy(t.Enums, o.Enums)
	maps.Copy(t.Sealed, o.Sealed)
	maps.Copy(t.Structs, o.Structs)
	maps.Copy(t.TypeDecls, o.TypeDecls)
	maps.Copy(t.FromRegistry, o.FromRegistry)
	maps.Copy(t.Interfaces, o.Interfaces)
	maps.Copy(t.EmbeddedIfaces, o.EmbeddedIfaces)
	maps.Copy(t.Methods, o.Methods)
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
			open := indexOf(toks, i+2, "{")
			if open >= 0 {
				closeIdx := scan.MatchBrace(toks, open)
				methods, embedded := parseInterfaceBody(src, toks, open, closeIdx)
				t.Interfaces[name] = methods
				if len(embedded) > 0 {
					t.EmbeddedIfaces[name] = embedded
				}
			}
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
		line := stripFieldTag(raw)
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

// stripFieldTag removes a struct field's tag — a back-quoted or double-quoted string
// literal following the type (`Name string ` + "`json:\"n\"`") — and anything after it,
// so the tag is not mistaken for the field type. A Go field type never contains a quote,
// so the first back-quote (or, for the rare double-quoted tag, the first double-quote)
// opens the tag and ends the name/type portion. Any trailing line comment goes with it.
func stripFieldTag(line string) string {
	cut := -1
	for i := 0; i < len(line); i++ {
		if line[i] == '`' || line[i] == '"' {
			cut = i
			break
		}
	}
	if cut < 0 {
		return line
	}
	return line[:cut]
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

// analyzeFromFuncs records each `from func NAME(p SRC) RET` conversion in the From
// registry, keyed by (SRC, target). The target and fallibility are read from RET via
// parseReturn, so a fallible `(T, error)` leaf keys by T. A forward scan from the
// parameter list handles both a bare return type (feature 06) and a parenthesized
// `(T, error)` (feature 12). The `from` modifier is stripped by the derive pass.
func analyzeFromFuncs(src string, toks []scan.Token, t *Tables) {
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "from" || toks[i+1].Text != "func" {
			continue
		}
		open := indexOf(toks, i+2, "(")
		if open < 0 {
			continue
		}
		closeP := scan.MatchParen(toks, open)
		srcType := strings.TrimSpace(src[toks[open+1].End:toks[closeP].Start])
		retType := strings.TrimSpace(src[toks[closeP].End:firstBraceAfter(src, toks[closeP].End)])
		tgt, fallible := parseReturn(retType)
		t.FromRegistry[[2]string{srcType, tgt}] = ConvEntry{Name: toks[i+2].Text, Fallible: fallible}
	}
}

// parseReturn splits a function return type into its target type and whether it is
// fallible: `(T, error)` -> (T, true); a bare `T` -> (T, false).
func parseReturn(ret string) (tgt string, fallible bool) {
	ret = strings.TrimSpace(ret)
	if strings.HasPrefix(ret, "(") && strings.HasSuffix(ret, ")") {
		first, _, _ := strings.Cut(ret[1:len(ret)-1], ",")
		return strings.TrimSpace(first), true
	}
	return ret, false
}

// firstBraceAfter returns the offset of the first "{" at or after offset, or len(src).
func firstBraceAfter(src string, offset int) int {
	if b := strings.IndexByte(src[offset:], '{'); b >= 0 {
		return offset + b
	}
	return len(src)
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

// parseInterfaceBody parses the body of `type I interface { … }` between the braces at
// open and close, returning its declared methods and the names of any embedded
// interfaces. A method is a `Name(params) results` entry whose token after the name is
// "(". A bare type name on its own (no following "(") is an embedded interface.
func parseInterfaceBody(src string, toks []scan.Token, open, close int) (methods []Method, embedded []string) {
	for k := open + 1; k < close; k++ {
		if !scan.IsIdent(toks[k].Text) {
			continue
		}
		if k+1 < close && toks[k+1].Text == "(" {
			name := toks[k].Text
			pc := scan.MatchParen(toks, k+1)
			m := methodFrom(src, toks, name, k+1, pc, close)
			methods = append(methods, m)
			k = endOfSignature(toks, pc, close)
			continue
		}
		// A bare identifier (possibly qualified `io.Reader`) with no "(" is an embedded
		// interface. Capture the dotted name and skip to the line's end.
		var name strings.Builder
		name.WriteString(toks[k].Text)
		for k+2 < close && toks[k+1].Text == "." && scan.IsIdent(toks[k+2].Text) {
			name.WriteString(".")
			name.WriteString(toks[k+2].Text)
			k += 2
		}
		embedded = append(embedded, name.String())
	}
	return methods, embedded
}

// analyzeMethods records every concrete-type method `func (r T) Name(params) results`
// under its receiver type name, keyed minus any pointer star (value- and pointer-
// receiver methods both contribute to *T's method set, which is what an interface
// assertion `var _ I = (*T)(nil)` sees).
func analyzeMethods(src string, toks []scan.Token, t *Tables) {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "func" || toks[i+1].Text != "(" {
			continue
		}
		rc := scan.MatchParen(toks, i+1)
		recv := receiverType(toks, i+1, rc)
		if recv == "" {
			continue
		}
		// The method name is the identifier just after the receiver's ")".
		ni := rc + 1
		if ni >= len(toks) || !scan.IsIdent(toks[ni].Text) {
			continue
		}
		po := ni + 1
		if po >= len(toks) || toks[po].Text != "(" {
			continue
		}
		pc := scan.MatchParen(toks, po)
		body := scan.FirstBodyBrace(toks, i)
		end := len(toks)
		if body >= 0 {
			end = body
		}
		m := methodFrom(src, toks, toks[ni].Text, po, pc, end)
		t.Methods[recv] = append(t.Methods[recv], m)
	}
}

// receiverType returns the bare receiver type name of a method receiver `(r T)` or
// `(r *T)` between the parens at open and close, stripped of a leading "*". It returns
// "" when no type identifier is present.
func receiverType(toks []scan.Token, open, close int) string {
	name := ""
	for k := open + 1; k < close; k++ {
		if scan.IsIdent(toks[k].Text) {
			name = toks[k].Text // last identifier in the receiver = the type
		}
	}
	return name
}

// methodFrom builds a Method from a signature whose parameter list opens at po (the
// "(") and closes at pc (the ")"). The result list is the text from pc to the next
// signature terminator (a "{" body, a newline at interface-body depth, or limit).
func methodFrom(src string, toks []scan.Token, name string, po, pc, limit int) Method {
	params := strings.TrimSpace(src[toks[po].Start:toks[pc].End])
	resEnd := endOfSignature(toks, pc, limit)
	results := ""
	if resEnd > pc {
		results = strings.TrimSpace(src[toks[pc].End:toks[resEnd].End])
	}
	raw := strings.TrimSpace(params + " " + results)
	return Method{Name: name, Sig: normalizeSig(params, results), Raw: raw}
}

// endOfSignature returns the index of the last token of a method signature whose
// parameter list closes at pc: it consumes a balanced result list (a parenthesized
// `(T, error)` or a single bare type) up to limit. It returns pc when there is no
// result list (the next token starts a new method or closes the body).
func endOfSignature(toks []scan.Token, pc, limit int) int {
	k := pc + 1
	if k >= limit {
		return pc
	}
	switch {
	case toks[k].Text == "(":
		return scan.MatchParen(toks, k)
	case toks[k].Text == "{" || toks[k].Text == "}":
		return pc
	case scan.IsIdent(toks[k].Text) || toks[k].Text == "*" || toks[k].Text == "[":
		// A single bare result type, possibly `*T`, `[]T`, `pkg.T`. Walk the balanced
		// type expression to its end.
		end := k
		depth := 0
		for ; k < limit; k++ {
			switch toks[k].Text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			if depth < 0 {
				break
			}
			end = k
			// A bare type ends at the next ident that starts a new method: ident "(".
			if depth == 0 && k+1 < limit && scan.IsIdent(toks[k+1].Text) &&
				k+2 < limit && toks[k+2].Text == "(" {
				break
			}
		}
		return end
	default:
		return pc
	}
}

// normalizeSig produces a canonical signature string from a method's parameter and
// result text by reducing each to its type sequence: parameter names are dropped and
// all whitespace is collapsed, so `(p []byte) (int, error)` and `([]byte) (int,error)`
// compare equal. The result is `paramTypes|resultTypes`.
func normalizeSig(params, results string) string {
	return normalizeTypeList(stripOuterParens(params)) + "|" + normalizeTypeList(stripOuterParens(results))
}

// stripOuterParens removes one layer of surrounding parentheses from s, if present.
func stripOuterParens(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return s[1 : len(s)-1]
	}
	return s
}

// normalizeTypeList canonicalizes a comma-separated parameter/result list to its type
// sequence: each entry's leading parameter name (an identifier followed by a type) is
// dropped, leaving the type; whitespace is collapsed. A lone type with no name is kept
// as-is. Entries are split at top-level commas only (nested generics/funcs survive).
func normalizeTypeList(list string) string {
	list = strings.TrimSpace(list)
	if list == "" {
		return ""
	}
	parts := splitTopLevel(list)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, normalizeParam(p))
	}
	return strings.Join(out, ",")
}

// normalizeParam reduces one `name Type` (or bare `Type`) entry to its collapsed type.
// A leading bare identifier that is followed by more tokens is treated as the parameter
// name and dropped; the remainder is the type. A single token is a bare type kept as-is.
func normalizeParam(p string) string {
	fields := strings.Fields(p)
	if len(fields) >= 2 {
		// `name Type…` — drop the name, keep the rest as the type. (A leading `...` or
		// `[]`/`*` would attach to the type, never standing alone as a "name".)
		if isBareName(fields[0]) {
			return collapse(strings.Join(fields[1:], " "))
		}
	}
	return collapse(p)
}

// isBareName reports whether s is a plain identifier (no type punctuation), i.e. a
// candidate parameter name rather than a type.
func isBareName(s string) bool {
	for _, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return s != ""
}

// collapse trims and squeezes internal whitespace runs in a type expression to a single
// space, then removes spaces adjacent to punctuation so spacing differences don't matter.
func collapse(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	for _, p := range []string{"[", "]", "*", ".", ","} {
		s = strings.ReplaceAll(s, " "+p, p)
		s = strings.ReplaceAll(s, p+" ", p)
	}
	return s
}

// splitTopLevel splits a comma-separated list at commas that sit at delimiter depth 0,
// so a generic `map[K]V` or a func type's inner commas do not split the entry.
func splitTopLevel(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i, r := range s {
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if last := strings.TrimSpace(s[start:]); last != "" {
		parts = append(parts, last)
	}
	return parts
}

// analyzeSig reads the return mode, type parameters, and return arity of one function from
// its (un-lowered) signature.
func analyzeSig(src string, toks []scan.Token, f scan.Func) FuncSig {
	sig := FuncSig{Name: f.Name, Mode: ModeNone}
	pc := f.ParamsClose
	if pc < 0 || f.BodyOpen < 0 {
		return sig
	}
	sig.Arity = inFileArity(src, toks, f)
	if pc+2 < f.BodyOpen {
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
	}
	// A Result/Option signature lowers to a fixed call shape — (T, error) or *T — so its
	// `?`-time arity is the lowered count, not the single syntactic return read above.
	switch sig.Mode {
	case ModeResult:
		sig.Arity = 2
	case ModeOption, ModeResultClosed:
		sig.Arity = 1
	}
	return sig
}

// inFileArity reports how many values f returns, read from its un-lowered return clause (the
// text between the parameter list's close and the body brace). The parameter-list close is
// derived from the name token because scan.ParamsClose can point at a parenthesized return
// type's own ")" instead.
func inFileArity(src string, toks []scan.Token, f scan.Func) int {
	open := f.NameTok + 1
	if open >= len(toks) || toks[open].Text != "(" {
		return 0
	}
	cl := scan.MatchParen(toks, open)
	if cl < 0 || cl >= f.BodyOpen {
		return 0
	}
	return countReturns(strings.TrimSpace(src[toks[cl].End:toks[f.BodyOpen].Start]))
}

// countReturns reports how many values a function's return clause yields: an empty clause is
// 0, a single bare type is 1, and a parenthesized list is its count of top-level entries
// (`(a, b int, err error)` is 3, `(int, error)` is 2).
func countReturns(ret string) int {
	ret = strings.TrimSpace(ret)
	if ret == "" {
		return 0
	}
	if strings.HasPrefix(ret, "(") && strings.HasSuffix(ret, ")") {
		inner := strings.TrimSpace(ret[1 : len(ret)-1])
		if inner == "" {
			return 0
		}
		return len(splitTopLevel(inner))
	}
	return 1
}

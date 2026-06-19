// Package main is a standalone reference transpiler for goal feature 06-error-e:
// closed error type E. A `Result[T, E]` whose E is a closed error enum lowers to
// the §8.1 sum encoding (a sealed Ok/Err sum), NOT the open-E native tuple. `match`
// on it is a type switch; `?` is type-switch-and-return, with a `From`-conversion
// (declared `from func`) auto-invoked in the Err arm when the caller's error type
// differs from the callee's (§3.3, §8.3 closed-E fork, resolving the §9 From shape).
//
// It composes feature 01 (enum encoding + construction), 02 (match -> type switch),
// 03 (Result.Ok/Err construction), and 05 (`?`). Scope: closed-E only; open-E
// (error) Results are feature 03. No error checking, no type inference. The match/`?`
// scrutinee must be a direct call so the callee's Result type is resolvable. Nested
// Err patterns and value-position match are out of scope. Malformed input is UB.
package main

import (
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/scanner"
	"unicode"
)

type token struct {
	text  string
	start int
	end   int
}

func lex(src string) []token {
	var s scanner.Scanner
	s.Init(strings.NewReader(src))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanRawStrings |
		scanner.ScanInts | scanner.ScanFloats | scanner.ScanChars | scanner.ScanComments | scanner.SkipComments
	s.Whitespace = 1<<'\t' | 1<<'\n' | 1<<'\r' | 1<<' '
	var toks []token
	for tk := s.Scan(); tk != scanner.EOF; tk = s.Scan() {
		txt := s.TokenText()
		start := s.Position.Offset
		toks = append(toks, token{text: txt, start: start, end: start + len(txt)})
	}
	return toks
}

type replacement struct {
	start, end int
	text       string
}

type span struct{ start, end int }

// resType is the (T, E) of a Result[T, E].
type resType struct{ t, e string }

const (
	evar = "__gop_e" // type-switch guard at a match/? site
)

// the injected generic sum encoding for closed-E Results.
const resultPreamble = `type Result[T, E any] interface{ isResult() }
type Ok[T, E any] struct{ Value T }
type Err[T, E any] struct{ Value E }

func (Ok[T, E]) isResult()  {}
func (Err[T, E]) isResult() {}`

type tstate struct {
	src        string
	toks       []token
	enums      map[string]*enumInfo  // enum name -> info (variant fields)
	funcRes    map[string]resType    // func name -> Result[T,E] it returns (closed-E)
	fromConv   map[[2]string]string  // (srcErr, dstErr) -> conversion func name
	funcRanges []frange              // body span -> enclosing Result type
	matchSpans []span                // match regions (skip for enum-construction)
	hasClosed  bool
}

type frange struct {
	start, end int
	res        resType
	hasRes     bool
}

func transpile(src string) (string, error) {
	st := &tstate{
		src:      src,
		toks:     lex(src),
		enums:    map[string]*enumInfo{},
		funcRes:  map[string]resType{},
		fromConv: map[[2]string]string{},
	}
	var reps []replacement

	reps = append(reps, st.scanEnums()...)
	reps = append(reps, st.scanFuncs()...)
	if st.hasClosed {
		reps = append(reps, st.injectPreamble())
	}
	reps = append(reps, st.lowerMatches()...)
	qReps, err := st.lowerQuestions()
	if err != nil {
		return "", err
	}
	reps = append(reps, qReps...)
	reps = append(reps, st.lowerResultCtors()...)
	reps = append(reps, st.lowerEnumCtors()...)

	out := splice(src, 0, len(src), reps)
	formatted, ferr := format.Source([]byte(out))
	if ferr != nil {
		return "", fmt.Errorf("generated Go did not parse: %w\n--- generated ---\n%s", ferr, out)
	}
	return string(formatted), nil
}

// ----- enums (feature 01 encoding) -----

type field struct{ name, typ string }
type variant struct {
	name   string
	fields []field
}
type enumInfo struct {
	name     string
	variants []variant
	fieldSet map[string]map[string]bool
}

func (st *tstate) scanEnums() []replacement {
	var reps []replacement
	toks := st.toks
	for i := 0; i < len(toks); i++ {
		if toks[i].text != "enum" {
			continue
		}
		rep, info, next := st.parseEnum(i)
		reps = append(reps, rep)
		st.enums[info.name] = info
		i = next - 1
	}
	return reps
}

func (st *tstate) parseEnum(i int) (replacement, *enumInfo, int) {
	toks, src := st.toks, st.src
	name := toks[i+1].text
	k := i + 3
	info := &enumInfo{name: name, fieldSet: map[string]map[string]bool{}}
	for toks[k].text != "}" {
		vname := toks[k].text
		k++
		var fields []field
		if toks[k].text == "{" {
			fields, k = parseFields(src, toks, k+1)
			k++
		}
		info.variants = append(info.variants, variant{name: vname, fields: fields})
		set := map[string]bool{}
		for _, f := range fields {
			set[f.name] = true
		}
		info.fieldSet[vname] = set
	}
	return replacement{toks[i].start, toks[k].end, genEnum(info)}, info, k + 1
}

func parseFields(src string, toks []token, k int) ([]field, int) {
	var fields []field
	for toks[k].text != "}" {
		name := toks[k].text
		k += 2
		typeStart := toks[k].start
		typeEnd := toks[k].end
		depth := 0
		for {
			t := toks[k]
			if depth == 0 && (t.text == "," || t.text == "}") {
				break
			}
			switch t.text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			typeEnd = t.end
			k++
		}
		fields = append(fields, field{name: name, typ: strings.TrimSpace(src[typeStart:typeEnd])})
		if toks[k].text == "," {
			k++
		}
	}
	return fields, k
}

func genEnum(info *enumInfo) string {
	marker := "is" + info.name
	var b strings.Builder
	fmt.Fprintf(&b, "type %s interface{ %s() }\n\n", info.name, marker)
	for _, v := range info.variants {
		if len(v.fields) == 0 {
			fmt.Fprintf(&b, "type %s_%s struct{}\n", info.name, v.name)
			continue
		}
		fmt.Fprintf(&b, "type %s_%s struct {\n", info.name, v.name)
		for _, f := range v.fields {
			fmt.Fprintf(&b, "\t%s %s\n", exported(f.name), f.typ)
		}
		b.WriteString("}\n")
	}
	b.WriteString("\n")
	for _, v := range info.variants {
		fmt.Fprintf(&b, "func (%s_%s) %s() {}\n", info.name, v.name, marker)
	}
	return b.String()
}

// ----- functions: record Result modes and From-conversions, strip `from` -----

func (st *tstate) scanFuncs() []replacement {
	var reps []replacement
	toks := st.toks
	for i := 0; i < len(toks); i++ {
		isFrom := toks[i].text == "from" && i+1 < len(toks) && toks[i+1].text == "func"
		if toks[i].text != "func" && !isFrom {
			continue
		}
		fi := i
		if isFrom {
			fi = i + 1
		}
		bo := firstBodyBrace(toks, fi)
		if bo < 0 {
			continue
		}
		bc := matchBrace(toks, bo)
		pc := paramsClose(toks, bo)
		name := toks[fi+1].text

		if isFrom {
			// from func NAME ( PNAME SRC ) DST { ... }
			srcType := strings.TrimSpace(st.src[toks[fi+3].end:toks[pc].start])
			dstType := strings.TrimSpace(st.src[toks[pc].end:toks[bo].start])
			st.fromConv[[2]string{srcType, dstType}] = name
			reps = append(reps, replacement{toks[i].start, toks[fi].start, ""}) // strip `from `
			st.funcRanges = append(st.funcRanges, frange{toks[bo].start, toks[bc].end, resType{}, false})
			i = bc
			continue
		}

		info := frange{start: toks[bo].start, end: toks[bc].end}
		if pc >= 0 && pc+2 < bo && toks[pc+1].text == "Result" && toks[pc+2].text == "[" {
			t, e, _ := parseResult(st.src, toks, pc+1)
			if e != "error" {
				st.funcRes[name] = resType{t, e}
				info.res = resType{t, e}
				info.hasRes = true
				st.hasClosed = true
			}
		}
		st.funcRanges = append(st.funcRanges, info)
		i = bc
	}
	return reps
}

// parseResult parses Result[T, E] starting at toks[ri] == "Result". Returns T, E
// (verbatim) and the index of the closing "]".
func parseResult(src string, toks []token, ri int) (t, e string, rb int) {
	rb = matchBracket(toks, ri+1)
	comma := topLevelComma(toks, ri+1, rb)
	t = strings.TrimSpace(src[toks[ri+2].start:toks[comma].start])
	e = strings.TrimSpace(src[toks[comma].end:toks[rb].start])
	return t, e, rb
}

// ----- preamble injection -----

func (st *tstate) injectPreamble() replacement {
	off := st.injectOffset()
	return replacement{off, off, "\n" + resultPreamble + "\n"}
}

// injectOffset returns a byte offset just after the package clause / import block,
// where the generic Result encoding can be inserted (imports must precede decls).
func (st *tstate) injectOffset() int {
	toks, src := st.toks, st.src
	off := 0
	for i := range toks {
		switch toks[i].text {
		case "package":
			if nl := strings.IndexByte(src[toks[i+1].end:], '\n'); nl >= 0 {
				off = toks[i+1].end + nl + 1
			}
		case "import":
			if i+1 < len(toks) && toks[i+1].text == "(" {
				cl := matchParen(toks, i+1)
				if nl := strings.IndexByte(src[toks[cl].end:], '\n'); nl >= 0 {
					off = toks[cl].end + nl + 1
				}
			} else if i+1 < len(toks) {
				if nl := strings.IndexByte(src[toks[i+1].end:], '\n'); nl >= 0 {
					off = toks[i+1].end + nl + 1
				}
			}
		}
	}
	return off
}

// ----- match on a closed-E Result -> type switch -----

func (st *tstate) lowerMatches() []replacement {
	var reps []replacement
	toks := st.toks
	for i := 0; i < len(toks); i++ {
		if toks[i].text != "match" {
			continue
		}
		rep, next := st.lowerMatch(i)
		reps = append(reps, rep)
		i = next - 1
	}
	return reps
}

func (st *tstate) lowerMatch(mi int) (replacement, int) {
	toks, src := st.toks, st.src
	bo := mi + 1
	for depth := 0; ; bo++ {
		switch toks[bo].text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case "{":
			if depth == 0 {
				goto found
			}
		}
	}
found:
	scrut := strings.TrimSpace(src[toks[mi].end:toks[bo].start])
	bc := matchBrace(toks, bo)
	st.matchSpans = append(st.matchSpans, span{toks[mi].start, toks[bc].end})
	res := st.funcRes[leadIdent(scrut)]
	arms := parseResultArms(toks, bo+1, bc)

	var okArm, errArm *resultArm
	for j := range arms {
		switch arms[j].variant {
		case "Ok":
			okArm = &arms[j]
		case "Err":
			errArm = &arms[j]
		}
	}

	okBody, okUse := st.armBody(okArm, "Value")
	errBody, errUse := st.armBody(errArm, "Value")

	var b strings.Builder
	if okUse || errUse {
		fmt.Fprintf(&b, "switch %s := %s.(type) {\n", evar, scrut)
	} else {
		fmt.Fprintf(&b, "switch %s.(type) {\n", scrut)
	}
	fmt.Fprintf(&b, "case Ok[%s, %s]:\n%s\n", res.t, res.e, okBody)
	fmt.Fprintf(&b, "case Err[%s, %s]:\n%s\n", res.t, res.e, errBody)
	fmt.Fprintf(&b, "default:\npanic(%q)\n}", fmt.Sprintf("unreachable: non-exhaustive Result[%s, %s] (compiler invariant violated)", res.t, res.e))
	return replacement{toks[mi].start, toks[bc].end, b.String()}, bc + 1
}

// armBody returns the lowered arm body. If the arm binds and uses its value, a
// `name := __gop_e.<field>` alias is prepended. The bool reports guard-var use.
func (st *tstate) armBody(a *resultArm, field string) (string, bool) {
	if a == nil {
		return "", false
	}
	body := bodySrc(st.src, st.toks, a.bodyLo, a.bodyHi)
	if a.binding != "" && bodyUses(st.toks, a.bodyLo, a.bodyHi, a.binding) {
		return fmt.Sprintf("%s := %s.%s\n%s", a.binding, evar, field, body), true
	}
	return body, false
}

// ----- ? propagation over a closed-E Result -----

func (st *tstate) lowerQuestions() ([]replacement, error) {
	var reps []replacement
	toks, src := st.toks, st.src
	for q := range toks {
		if toks[q].text != "?" {
			continue
		}
		p := toks[q].start
		lineStart := strings.LastIndexByte(src[:p], '\n') + 1
		name, rhs, ok := splitAssign(src[lineStart:p])
		if !ok {
			return nil, fmt.Errorf("`?` must be the RHS of an assignment: `name := expr?`")
		}
		callee := st.funcRes[leadIdent(rhs)]
		caller, hasCaller := st.resultAt(p)
		if callee.e == "" || !hasCaller {
			return nil, fmt.Errorf("closed-E `?` needs a Result-returning callee and enclosing function")
		}
		errValue := evar + ".Value"
		if callee.e != caller.e {
			conv, found := st.fromConv[[2]string{callee.e, caller.e}]
			if !found {
				return nil, fmt.Errorf("no `from func` conversion declared for %s -> %s (required to `?` across closed error types)", callee.e, caller.e)
			}
			errValue = fmt.Sprintf("%s(%s.Value)", conv, evar)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "var %s %s\n", name, callee.t)
		fmt.Fprintf(&b, "switch %s := %s.(type) {\n", evar, rhs)
		fmt.Fprintf(&b, "case Ok[%s, %s]:\n%s = %s.Value\n", callee.t, callee.e, name, evar)
		fmt.Fprintf(&b, "case Err[%s, %s]:\nreturn Err[%s, %s]{Value: %s}\n", callee.t, callee.e, caller.t, caller.e, errValue)
		fmt.Fprintf(&b, "default:\npanic(%q)\n}", fmt.Sprintf("unreachable: non-exhaustive Result[%s, %s] (compiler invariant violated)", callee.t, callee.e))
		reps = append(reps, replacement{lineStart, toks[q].end, b.String()})
	}
	return reps, nil
}

func (st *tstate) resultAt(off int) (resType, bool) {
	for _, f := range st.funcRanges {
		if off >= f.start && off < f.end && f.hasRes {
			return f.res, true
		}
	}
	return resType{}, false
}

// ----- Result.Ok/Err construction (wrap only; inner expr lowered separately) -----

func (st *tstate) lowerResultCtors() []replacement {
	var reps []replacement
	toks := st.toks
	for i := 0; i+4 < len(toks); i++ {
		if toks[i].text != "Result" || toks[i+1].text != "." {
			continue
		}
		variant := toks[i+2].text
		if (variant != "Ok" && variant != "Err") || toks[i+3].text != "(" {
			continue
		}
		if st.inMatch(toks[i].start) {
			continue
		}
		res, ok := st.resultAt(toks[i].start)
		if !ok {
			continue
		}
		closeIdx := matchParen(toks, i+3)
		// wrap: `Result.Ok(`  ->  `Ok[T, E]{Value: `   and  `)` -> `}`
		reps = append(reps, replacement{toks[i].start, toks[i+3].end,
			fmt.Sprintf("%s[%s, %s]{Value: ", variant, res.t, res.e)})
		reps = append(reps, replacement{toks[closeIdx].start, toks[closeIdx].end, "}"})
		i = closeIdx
	}
	return reps
}

// ----- enum construction (feature 01): EnumName.Variant[(args)] -----

func (st *tstate) lowerEnumCtors() []replacement {
	var reps []replacement
	toks := st.toks
	for j := 0; j+2 < len(toks); j++ {
		e, ok := st.enums[toks[j].text]
		if !ok || toks[j+1].text != "." {
			continue
		}
		vname := toks[j+2].text
		if _, isV := e.fieldSet[vname]; !isV {
			continue
		}
		if st.inMatch(toks[j].start) {
			continue
		}
		if j+3 < len(toks) && toks[j+3].text == "(" {
			closeIdx := matchParen(toks, j+3)
			args := parseArgs(st.src, toks, j+4, closeIdx)
			reps = append(reps, replacement{toks[j].start, toks[closeIdx].end, enumCtor(e.name, vname, args)})
			j = closeIdx
			continue
		}
		reps = append(reps, replacement{toks[j].start, toks[j+2].end, enumCtor(e.name, vname, nil)})
		j += 2
	}
	return reps
}

type kv struct{ label, expr string }

func parseArgs(src string, toks []token, start, closeIdx int) []kv {
	var args []kv
	k := start
	for k < closeIdx {
		label := toks[k].text
		k += 2
		exprStart := toks[k].start
		exprEnd := toks[k].end
		depth := 0
		for k < closeIdx {
			t := toks[k]
			if depth == 0 && t.text == "," {
				break
			}
			switch t.text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			exprEnd = t.end
			k++
		}
		args = append(args, kv{label, strings.TrimSpace(src[exprStart:exprEnd])})
		if k < closeIdx && toks[k].text == "," {
			k++
		}
	}
	return args
}

func enumCtor(enum, variant string, args []kv) string {
	if len(args) == 0 {
		return fmt.Sprintf("%s(%s_%s{})", enum, enum, variant)
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprintf("%s: %s", exported(a.label), a.expr)
	}
	return fmt.Sprintf("%s(%s_%s{%s})", enum, enum, variant, strings.Join(parts, ", "))
}

// ----- arm parsing (Result.Ok/Err patterns), shared with 02/03 -----

type resultArm struct {
	variant        string
	binding        string
	bodyLo, bodyHi int
}

func parseResultArms(toks []token, lo, hi int) []resultArm {
	var arrows []int
	for j, depth := lo, 0; j < hi; j++ {
		switch toks[j].text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
		if depth == 0 && toks[j].text == "=" && j+1 < hi && toks[j+1].text == ">" {
			arrows = append(arrows, j)
		}
	}
	arms := make([]resultArm, len(arrows))
	for i, eq := range arrows {
		patStart := lo
		if i > 0 {
			patStart = patternStart(toks, arrows[i])
		}
		a := resultArm{variant: toks[patStart+2].text}
		if patStart+3 < eq && toks[patStart+3].text == "(" {
			a.binding = toks[patStart+4].text
		}
		a.bodyLo = eq + 2
		if i+1 < len(arrows) {
			a.bodyHi = patternStart(toks, arrows[i+1])
		} else {
			a.bodyHi = hi
		}
		arms[i] = a
	}
	return arms
}

func patternStart(toks []token, eqIdx int) int {
	j := eqIdx - 1
	if toks[j].text == ")" {
		depth := 0
		k := j
		for ; k >= 0; k-- {
			switch toks[k].text {
			case ")":
				depth++
			case "(":
				depth--
			}
			if depth == 0 {
				break
			}
		}
		return k - 3
	}
	return j - 2
}

// ----- shared helpers -----

func (st *tstate) inMatch(off int) bool {
	for _, s := range st.matchSpans {
		if off >= s.start && off < s.end {
			return true
		}
	}
	return false
}

func bodySrc(src string, toks []token, lo, hi int) string {
	if lo >= hi {
		return ""
	}
	return strings.TrimSpace(src[toks[lo].start:toks[hi-1].end])
}

func bodyUses(toks []token, lo, hi int, name string) bool {
	for j := lo; j < hi; j++ {
		if toks[j].text == name {
			return true
		}
	}
	return false
}

func splitAssign(s string) (name, rhs string, ok bool) {
	if lhs, after, found := strings.Cut(s, ":="); found {
		return strings.TrimSpace(lhs), strings.TrimSpace(after), true
	}
	return "", strings.TrimSpace(s), false
}

func leadIdent(s string) string {
	end := 0
	for end < len(s) {
		r := rune(s[end])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			end++
			continue
		}
		break
	}
	return s[:end]
}

func exported(name string) string {
	if name == "" {
		return name
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func firstBodyBrace(toks []token, fi int) int {
	depth := 0
	for k := fi + 1; k < len(toks); k++ {
		switch toks[k].text {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case "{":
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

func paramsClose(toks []token, body int) int {
	depth := 0
	for k := body - 1; k >= 0; k-- {
		t := toks[k].text
		if depth == 0 && t == ")" {
			return k
		}
		switch t {
		case ")", "]":
			depth++
		case "(", "[":
			depth--
		}
	}
	return -1
}

func topLevelComma(toks []token, openIdx, closeIdx int) int {
	depth := 0
	for k := openIdx + 1; k < closeIdx; k++ {
		switch toks[k].text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
		if depth == 0 && toks[k].text == "," {
			return k
		}
	}
	return -1
}

func matchParen(toks []token, openIdx int) int   { return matchPair(toks, openIdx, "(", ")") }
func matchBracket(toks []token, openIdx int) int { return matchPair(toks, openIdx, "[", "]") }
func matchBrace(toks []token, openIdx int) int   { return matchPair(toks, openIdx, "{", "}") }

func matchPair(toks []token, openIdx int, open, close string) int {
	depth := 0
	for k := openIdx; k < len(toks); k++ {
		switch toks[k].text {
		case open:
			depth++
		case close:
			depth--
		}
		if depth == 0 {
			return k
		}
	}
	return len(toks) - 1
}

func splice(src string, lo, hi int, reps []replacement) string {
	sort.SliceStable(reps, func(a, b int) bool {
		if reps[a].start != reps[b].start {
			return reps[a].start < reps[b].start
		}
		return reps[a].end < reps[b].end
	})
	var b strings.Builder
	prev := lo
	for _, r := range reps {
		if r.start < prev {
			continue
		}
		b.WriteString(src[prev:r.start])
		b.WriteString(r.text)
		prev = r.end
	}
	b.WriteString(src[prev:hi])
	return b.String()
}

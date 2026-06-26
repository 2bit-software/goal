package analyze

import (
	"strings"

	"goal/internal/scan"
)

// Receiver-type resolution for `?` callees. A `recv.Method()?` site cannot be resolved by name
// alone — `recv` is a value, so its method's signature lives behind its TYPE. This file does the
// one lexical step that recovers it: read the receiver variable's declared type from the
// enclosing function's receiver/parameter list, then look the method up by `Type.Method` (an
// in-file type via Tables.Methods, an imported type via Tables.ForeignMethods). A receiver whose
// type is not lexically declared — a local bound by `:=`, the result of a call chain — stays
// unresolved, and the `?` lowering keeps its safe two-value form.

// ResolveCallee resolves the function or method that rhs (a `?` operand) calls, returning its
// `?`-relevant signature. off is the byte offset of the `?`, used to find the enclosing
// function for receiver/parameter lookup. A plain or package-qualified call resolves through
// FuncSignatures; a `recv.Method()` call resolves through the receiver's declared type. ok is
// false when the callee cannot be resolved lexically.
func ResolveCallee(t *Tables, toks []scan.Token, off int, rhs string) (FuncSig, bool) {
	key := scan.CalleeKey(rhs)
	if key == "" {
		return FuncSig{}, false
	}
	if sig, ok := t.FuncSignatures[key]; ok {
		return sig, true
	}
	recv, method, isMethod := strings.Cut(key, ".")
	if !isMethod {
		return FuncSig{}, false
	}
	typ, ok := scopeVarType(toks, off, recv)
	if !ok {
		return FuncSig{}, false
	}
	base := baseType(typ)
	if strings.Contains(base, ".") {
		sig, ok := t.ForeignMethods[base+"."+method]
		return sig, ok
	}
	for _, m := range t.Methods[base] {
		if m.Name == method {
			return FuncSig{Arity: m.Arity, EndsInError: m.EndsInError}, true
		}
	}
	return FuncSig{}, false
}

// QuestionMethodReceiverPkgs returns the import qualifiers of the receiver types of every
// `recv.Method()?` site in src (e.g. "os" for a `w.Close()?` whose `w` is `*os.File`), so the
// foreign loader knows which packages to parse for their methods.
func QuestionMethodReceiverPkgs(src string) map[string]bool {
	toks := scan.Lex(src)
	out := map[string]bool{}
	for q := range toks {
		if toks[q].Text != "?" {
			continue
		}
		p := toks[q].Start
		lineStart := strings.LastIndexByte(src[:p], '\n') + 1
		_, rhs, _ := scan.SplitAssign(src[lineStart:p])
		recv, _, isMethod := strings.Cut(scan.CalleeKey(rhs), ".")
		if !isMethod {
			continue
		}
		typ, ok := scopeVarType(toks, p, recv)
		if !ok {
			continue
		}
		if pkg, _, ok := strings.Cut(baseType(typ), "."); ok {
			out[pkg] = true
		}
	}
	return out
}

// scopeVarType returns the declared type expression of variable `name` in the scope of the
// function whose body contains byte offset off — its receiver or a parameter. ok is false when
// `name` is not a receiver/parameter (a local, a field, or a package qualifier).
func scopeVarType(toks []scan.Token, off int, name string) (string, bool) {
	f, ok := enclosingFunc(toks, off)
	if !ok {
		return "", false
	}
	if rn, rt, isMethod := receiverNameType(toks, f); isMethod && rn == name {
		return rt, true
	}
	return paramType(toks, f, name)
}

// enclosingFunc returns the function whose body span contains byte offset off.
func enclosingFunc(toks []scan.Token, off int) (scan.Func, bool) {
	for _, f := range scan.ScanFuncs(toks) {
		if f.BodyOpen >= 0 && f.BodyClose > f.BodyOpen && f.BodyClose < len(toks) &&
			off >= toks[f.BodyOpen].Start && off < toks[f.BodyClose].End {
			return f, true
		}
	}
	return scan.Func{}, false
}

// receiverNameType returns the receiver variable name and type of a method `func (r T) Name…`.
// isMethod is false for a plain function (no receiver).
func receiverNameType(toks []scan.Token, f scan.Func) (name, typ string, isMethod bool) {
	if f.NameTok < 1 || toks[f.NameTok-1].Text != ")" {
		return "", "", false
	}
	open := matchOpenParenBack(toks, f.NameTok-1)
	if open < 0 || open+1 > f.NameTok-1 {
		return "", "", false
	}
	lo, hi := open+1, f.NameTok-1
	if hi-lo >= 2 && scan.IsIdent(toks[lo].Text) {
		return toks[lo].Text, concatTokens(toks, lo+1, hi), true
	}
	return "", concatTokens(toks, lo, hi), true // unnamed receiver — no name to match
}

// paramType returns the declared type of parameter `name` in f's parameter list. It handles the
// grouped form `(a, b int)` by carrying the type back to the bare names that share it.
func paramType(toks []scan.Token, f scan.Func, name string) (string, bool) {
	open := paramOpen(toks, f)
	if open < 0 {
		return "", false
	}
	close := scan.MatchParen(toks, open)
	if close <= open {
		return "", false
	}
	segs := commaSegments(toks, open+1, close)
	named := false
	for _, s := range segs {
		if isNameTypeSeg(toks, s[0], s[1]) {
			named = true
			break
		}
	}
	if !named {
		return "", false // an unnamed parameter list has no names to resolve
	}
	cur := ""
	for i := len(segs) - 1; i >= 0; i-- {
		lo, hi := segs[i][0], segs[i][1]
		switch {
		case isNameTypeSeg(toks, lo, hi):
			cur = concatTokens(toks, lo+1, hi)
			if toks[lo].Text == name {
				return cur, true
			}
		case hi-lo == 1 && scan.IsIdent(toks[lo].Text): // grouped bare name, inherits cur
			if toks[lo].Text == name {
				return cur, true
			}
		}
	}
	return "", false
}

// isNameTypeSeg reports whether tokens [lo, hi) are a clear `name type` parameter — a leading
// identifier followed by a type — as opposed to a bare grouped name or an unnamed `pkg.Type`.
func isNameTypeSeg(toks []scan.Token, lo, hi int) bool {
	return hi-lo >= 2 && scan.IsIdent(toks[lo].Text) && toks[lo+1].Text != "."
}

// paramOpen returns the index of the "(" opening f's parameter list (after the name and an
// optional type-parameter list), or -1.
func paramOpen(toks []scan.Token, f scan.Func) int {
	k := f.NameTok + 1
	if k < len(toks) && toks[k].Text == "[" {
		k = scan.MatchBracket(toks, k) + 1
	}
	if k < len(toks) && toks[k].Text == "(" {
		return k
	}
	return -1
}

// matchOpenParenBack returns the index of the "(" matching the ")" at closeIdx, scanning back.
func matchOpenParenBack(toks []scan.Token, closeIdx int) int {
	depth := 0
	for k := closeIdx; k >= 0; k-- {
		switch toks[k].Text {
		case ")", "]", "}":
			depth++
		case "(", "[", "{":
			depth--
		}
		if depth == 0 {
			return k
		}
	}
	return -1
}

// commaSegments splits toks[lo:hi] into [start,end) ranges at top-level commas.
func commaSegments(toks []scan.Token, lo, hi int) [][2]int {
	var segs [][2]int
	depth, start := 0, lo
	for k := lo; k < hi; k++ {
		switch toks[k].Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case ",":
			if depth == 0 {
				segs = append(segs, [2]int{start, k})
				start = k + 1
			}
		}
	}
	if start < hi {
		segs = append(segs, [2]int{start, hi})
	}
	return segs
}

// concatTokens joins the text of toks[lo:hi] (type expressions need no spaces).
func concatTokens(toks []scan.Token, lo, hi int) string {
	var b strings.Builder
	for k := lo; k < hi && k < len(toks); k++ {
		b.WriteString(toks[k].Text)
	}
	return b.String()
}

// baseType strips a leading "*" and a trailing "[…]" instantiation, yielding the named type a
// method table is keyed by ("*os.File" -> "os.File", "*Tree[T]" -> "Tree").
func baseType(typ string) string {
	typ = strings.TrimSpace(typ)
	for strings.HasPrefix(typ, "*") {
		typ = strings.TrimSpace(typ[1:])
	}
	if i := strings.IndexByte(typ, '['); i >= 0 {
		typ = typ[:i]
	}
	return strings.TrimSpace(typ)
}

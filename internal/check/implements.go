package check

import (
	"fmt"
	"strings"

	"goal/internal/analyze"
	"goal/internal/scan"
)

// checkImplements enforces feature 07 (implements): for `type T struct implements I`,
// the type T must actually satisfy interface I — every method I declares must exist on
// T with a matching signature. A missing or mis-signed method is an Error, located at
// the struct's `implements` clause (mirroring the declaration-site error goal promises:
// the contract is checked where it is asserted, not at a distant call site).
//
// Reuse, not reinvention:
//   - The implements pass (internal/pass/implements.go) locates the inline
//     `implements I, J` clause between `struct` and the body `{`, splits the
//     comma-separated interface list, and scans pointer receivers. This check lifts the
//     same clause locator and interface split.
//   - analyze.Tables.Sealed marks a sealed interface (feature 01): its obligation is a
//     single unexported marker method the implements pass synthesizes, so it is
//     trivially satisfied by construction — skipped here, never flagged.
//   - analyze.Tables (extended this iteration) carries the method index the obligation
//     needs: Interfaces (an in-file interface's declared method set, name + normalized
//     signature), EmbeddedIfaces (embedded interface names to fold in), and Methods (a
//     concrete type's declared methods). The implements check reads these by name.
//   - Must run pre-lowering: the clause is stripped to plain Go and the assertion
//     `var _ I = T{}` is emitted, erasing the structure this check inspects.
//
// Defer-boundary (emit a located Warning, never a false Error):
//   - A qualified interface (`io.Writer`) or any interface not declared in this file:
//     its method set is out-of-package and unreadable lexically — deferred.
//   - An interface that embeds an interface not resolvable in-file: the full obligation
//     is unknown — deferred.
//   - The clause subject T is only checked when it is an in-file struct whose methods
//     this file declares; nothing here invents methods T might gain elsewhere.
//
// Signature equality is by normalized type sequence (parameter and result types, names
// and whitespace stripped — see analyze.normalizeSig). Equality across type aliases or
// differently-spelled-but-equal types is *not* resolved lexically; a name mismatch that
// is really an alias would surface as a (false) mismatch, so the check only fires a
// mismatch Error when the method name is present but its normalized signature differs —
// the common, lexically-decidable case — and never tries to prove two differently-named
// types unequal beyond textual normalization.
func checkImplements(src string, t *analyze.Tables) ([]Diagnostic, error) {
	toks := scan.Lex(src)
	var diags []Diagnostic
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "type" || !scan.IsIdent(toks[i+1].Text) || toks[i+2].Text != "struct" {
			continue
		}
		typeName := toks[i+1].Text

		// The struct body's "{" is the first brace after `struct`; the clause sits
		// between `struct` and that brace (implements pass locator).
		open := -1
		for k := i + 3; k < len(toks); k++ {
			if toks[k].Text == "{" {
				open = k
				break
			}
		}
		if open < 0 {
			continue
		}
		imp := -1
		for k := i + 3; k < open; k++ {
			if toks[k].Text == "implements" {
				imp = k
				break
			}
		}
		if imp < 0 {
			continue // a plain struct with no implements clause
		}
		ifaces := splitInterfaces(src[toks[imp].End:toks[open].Start])
		impPos := toks[imp].Start
		for _, iface := range ifaces {
			diags = append(diags, checkOneImplements(t, typeName, iface, impPos)...)
		}
	}
	return diags, nil
}

// splitInterfaces splits the clause's comma-separated interface list into trimmed
// names, dropping empties (mirrors the implements pass's splitInterfaces). A qualified
// name (`io.Writer`) survives intact — it carries no comma.
func splitInterfaces(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// checkOneImplements verifies that type typeName satisfies the single interface iface
// asserted by the `implements` clause at impPos. It returns an Error per missing or
// mis-signed method, a single Warning when iface's obligation cannot be read lexically,
// or nothing when the obligation is met (or the interface is sealed, hence trivial).
func checkOneImplements(t *analyze.Tables, typeName, iface string, impPos int) []Diagnostic {
	// A sealed interface (feature 01) is satisfied by the unexported marker method the
	// implements pass itself synthesizes — trivially met, never an obligation here.
	if t.Sealed[iface] {
		return nil
	}

	// A qualified interface (`io.Writer`) is out-of-package: its method set is not
	// readable in-file. Defer.
	if strings.Contains(iface, ".") {
		return deferImplements(typeName, iface, impPos,
			fmt.Sprintf("interface `%s` is from another package", iface))
	}

	required, resolved := requiredMethods(t, iface, map[string]bool{})
	if !resolved {
		return deferImplements(typeName, iface, impPos,
			fmt.Sprintf("interface `%s` is not declared in this file (or embeds an interface that isn't)", iface))
	}

	have := map[string]analyze.Method{}
	for _, m := range t.Methods[typeName] {
		have[m.Name] = m
	}

	var diags []Diagnostic
	for _, want := range required {
		got, ok := have[want.Name]
		switch {
		case !ok:
			diags = append(diags, Diagnostic{
				Pos:      impPos,
				Severity: Error,
				Feature:  "07-implements",
				Code:     "unimplemented-method",
				Message: fmt.Sprintf("type `%s` does not implement `%s`: missing method `%s%s` — declare `func (%s) %s%s`",
					typeName, iface, want.Name, sigForMsg(want), recvHint(typeName), want.Name, sigForMsg(want)),
			})
		case got.Sig != want.Sig:
			diags = append(diags, Diagnostic{
				Pos:      impPos,
				Severity: Error,
				Feature:  "07-implements",
				Code:     "method-signature-mismatch",
				Message: fmt.Sprintf("type `%s` does not implement `%s`: method `%s` has signature `%s` but `%s` requires `%s`",
					typeName, iface, want.Name, sigText(got), iface, sigText(want)),
			})
		}
	}
	return diags
}

// requiredMethods returns interface iface's full method obligation — its own declared
// methods plus every method of each interface it embeds — and whether the obligation
// was fully resolvable in-file. An embedded interface that is qualified or undeclared
// makes the obligation unresolvable (resolved=false), so the caller defers rather than
// asserts an incomplete set. The seen set guards against an embedding cycle.
func requiredMethods(t *analyze.Tables, iface string, seen map[string]bool) (methods []analyze.Method, resolved bool) {
	if seen[iface] {
		return nil, true // already folded in (cycle guard) — contributes nothing new
	}
	seen[iface] = true
	own, ok := t.Interfaces[iface]
	if !ok {
		return nil, false // not an in-file interface — unreadable
	}
	methods = append(methods, own...)
	for _, emb := range t.EmbeddedIfaces[iface] {
		if strings.Contains(emb, ".") {
			return nil, false // embeds a qualified (out-of-package) interface
		}
		sub, subOK := requiredMethods(t, emb, seen)
		if !subOK {
			return nil, false
		}
		methods = append(methods, sub...)
	}
	return methods, true
}

// deferImplements builds the single located Warning emitted when an `implements`
// obligation cannot be proven lexically. The Warning names the type, the interface, and
// the reason it could not be resolved — never an Error, so a real (out-of-package)
// satisfaction is not falsely rejected.
func deferImplements(typeName, iface string, impPos int, reason string) []Diagnostic {
	return []Diagnostic{{
		Pos:      impPos,
		Severity: Warning,
		Feature:  "07-implements",
		Code:     "unresolved-interface",
		Message: fmt.Sprintf("cannot verify `%s implements %s`: %s — interface-satisfaction deferred",
			typeName, iface, reason),
	}}
}

// recvHint renders a receiver placeholder for the "declare …" suggestion, e.g.
// `r MyType`, using a lowercased first letter of the type as the receiver name.
func recvHint(typeName string) string {
	r := "r"
	if typeName != "" {
		r = strings.ToLower(typeName[:1])
	}
	return r + " " + typeName
}

// sigForMsg renders a method's parameter+result text for a suggestion, falling back to
// the normalized signature when the raw text is empty.
func sigForMsg(m analyze.Method) string {
	if m.Raw != "" {
		return m.Raw
	}
	return sigText(m)
}

// sigText renders a normalized signature `params|results` back into readable
// `(params) results` form for a diagnostic message.
func sigText(m analyze.Method) string {
	params, results, _ := strings.Cut(m.Sig, "|")
	out := "(" + params + ")"
	if results != "" {
		out += " " + results
	}
	return out
}

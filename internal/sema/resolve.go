package sema

// resolve.go derives the name-keyed semantic facts (Info) by walking the parsed
// goal AST. It is the structural replacement for internal/analyze's token
// scanning: because it reads enums, structs, signatures, the from-registry, and
// methods off the parse tree, it is correct by construction where the token
// scanner is not — most visibly a struct field whose type contains an embedded
// (top-level) comma, which analyze.parseStructBody splits on whitespace and
// mangles. Here a field's type is rendered from its AST type node (typeString),
// so the comma never confuses the resolver.

import (
	"maps"
	"strings"

	"goal/internal/ast"
	"goal/internal/token"
)

// ResolvePackage walks every file of a goal package and returns one merged Info,
// so a file resolves enums/structs/from-funcs/signatures/methods declared in a
// sibling. It is the AST analogue of analyze.BuildPackage: the package driver
// threads the merged facts through every file's emit, so a cross-file reference
// (e.g. a `match` over an enum declared in another file) lowers correctly. Files
// are merged in the given order; on a duplicate key the last file wins (a genuine
// redeclaration is a Go error the toolchain reports), mirroring analyze.Merge.
func ResolvePackage(files []*ast.File) *Info {
	merged := Resolve(nil) // an empty, fully-initialized Info
	for _, f := range files {
		merged.Merge(Resolve(f))
	}
	return merged
}

// Merge unions o's name-keyed facts into info (last-merged-wins on a shared key),
// mirroring analyze.Tables.Merge so the AST package driver coheres a package the
// same way the splice driver does.
func (info *Info) Merge(o *Info) {
	maps.Copy(info.FuncSignatures, o.FuncSignatures)
	maps.Copy(info.Enums, o.Enums)
	maps.Copy(info.Sealed, o.Sealed)
	maps.Copy(info.Structs, o.Structs)
	maps.Copy(info.FromRegistry, o.FromRegistry)
	maps.Copy(info.Methods, o.Methods)
	maps.Copy(info.Interfaces, o.Interfaces)
	maps.Copy(info.EmbeddedIfaces, o.EmbeddedIfaces)
}

// Resolve walks a parsed file and returns its name-keyed semantic facts. The
// returned Info has every map initialized (safe to read even when empty). An
// unrecognized or not-yet-modeled declaration is skipped, mirroring analyze's
// tolerance.
func Resolve(f *ast.File) *Info {
	info := &Info{
		FuncSignatures: map[string]FuncSig{},
		Enums:          map[string]*Enum{},
		Sealed:         map[string]bool{},
		Structs:        map[string][]Field{},
		FromRegistry:   map[[2]string]ConvEntry{},
		Methods:        map[string][]Method{},
		Interfaces:     map[string][]Method{},
		EmbeddedIfaces: map[string][]string{},
	}
	if f == nil {
		return info
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.EnumDecl:
			info.resolveEnum(d)
		case *ast.SealedInterfaceDecl:
			if d.Name != nil {
				info.Sealed[d.Name.Name] = true
			}
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				info.resolveTypeDecl(d)
			}
		case *ast.FuncDecl:
			info.resolveFunc(d)
		}
	}
	return info
}

// resolveEnum records an enum and its variants (with payload fields) plus the
// variant-name and per-variant field-name membership sets.
func (info *Info) resolveEnum(d *ast.EnumDecl) {
	if d.Name == nil {
		return
	}
	e := &Enum{
		Name:     d.Name.Name,
		VSet:     map[string]bool{},
		FieldSet: map[string]map[string]bool{},
	}
	for _, v := range d.Variants {
		if v == nil || v.Name == nil {
			continue
		}
		var fields []Field
		fset := map[string]bool{}
		for _, pf := range v.Payload {
			if pf == nil || pf.Name == nil {
				continue
			}
			fields = append(fields, Field{Name: pf.Name.Name, Type: typeString(pf.Type)})
			fset[pf.Name.Name] = true
		}
		e.Variants = append(e.Variants, Variant{Name: v.Name.Name, Fields: fields})
		e.VSet[v.Name.Name] = true
		e.FieldSet[v.Name.Name] = fset
	}
	info.Enums[e.Name] = e
}

// resolveTypeDecl records every `type X struct {…}` in a (possibly grouped) type
// declaration as ordered struct fields, and every `type X interface {…}` as its
// declared method set plus embedded-interface names. Sealed interfaces are their
// own decl node (handled in Resolve); aliases are not modeled here.
func (info *Info) resolveTypeDecl(d *ast.GenDecl) {
	for _, s := range d.Specs {
		ts, ok := s.(*ast.TypeSpec)
		if !ok || ts.Name == nil {
			continue
		}
		switch t := ts.Type.(type) {
		case *ast.StructType:
			if t.Fields != nil {
				info.Structs[ts.Name.Name] = structFields(t.Fields)
			}
		case *ast.InterfaceType:
			info.resolveInterface(ts.Name.Name, t)
		}
	}
}

// resolveInterface records an in-file interface's directly declared methods (into
// Interfaces) and its embedded interface names (into EmbeddedIfaces). A method spec
// is a Field carrying a name and a *FuncType; an embedded interface is a Field with
// no names whose type renders to a (possibly qualified) interface name. The method
// Sig is normalized with the same helpers as a concrete method (resolveMethod), so
// the implements check compares interface and concrete signatures apples-to-apples.
func (info *Info) resolveInterface(name string, it *ast.InterfaceType) {
	// Register the interface even when empty, so a declared-but-method-less
	// interface resolves (resolved=true) instead of looking out-of-file.
	if _, seen := info.Interfaces[name]; !seen {
		info.Interfaces[name] = nil
	}
	if it.Methods == nil {
		return
	}
	for _, f := range it.Methods.List {
		if f == nil {
			continue
		}
		if len(f.Names) == 0 {
			// Embedded interface: its type is the embedded interface name.
			if emb := typeString(f.Type); emb != "" {
				info.EmbeddedIfaces[name] = append(info.EmbeddedIfaces[name], emb)
			}
			continue
		}
		ft, ok := f.Type.(*ast.FuncType)
		if !ok {
			continue
		}
		params := paramTypeListFL(ft.Params)
		results := paramTypeListFL(ft.Results)
		for _, n := range f.Names {
			info.Interfaces[name] = append(info.Interfaces[name], Method{
				Name:        n.Name,
				Sig:         joinTypes(params) + "|" + joinTypes(results),
				Arity:       resultArity(ft.Results),
				EndsInError: resultEndsInError(ft.Results),
			})
		}
	}
}

// structFields flattens a struct field list to ordered (name, type) entries,
// expanding a shared-type group `a, b int` into one entry per name. An embedded
// field (no names) is skipped, matching analyze.
func structFields(fl *ast.FieldList) []Field {
	var fields []Field
	for _, f := range fl.List {
		if f == nil || len(f.Names) == 0 {
			continue
		}
		typ := typeString(f.Type)
		for _, n := range f.Names {
			fields = append(fields, Field{Name: n.Name, Type: typ})
		}
	}
	return fields
}

// resolveFunc routes a func declaration to the right table: a from/derive
// modifier registers a conversion; a receiver records a method; otherwise it is a
// plain function signature.
func (info *Info) resolveFunc(d *ast.FuncDecl) {
	switch {
	case d.Mod == ast.FuncFrom || d.Mod == ast.FuncDerive:
		info.resolveConversion(d)
	case d.Recv != nil:
		info.resolveMethod(d)
	default:
		if d.Name != nil {
			info.FuncSignatures[d.Name.Name] = funcSig(d.Name.Name, d.Type)
		}
	}
}

// resolveConversion registers a `from func`/`derive func` in the from-registry,
// keyed by (source type, target type). Source is the first parameter's type;
// target and fallibility come from the result list.
func (info *Info) resolveConversion(d *ast.FuncDecl) {
	if d.Name == nil || d.Type == nil {
		return
	}
	src := firstParamType(d.Type.Params)
	if src == "" {
		return
	}
	tgt, fallible := resultTarget(d.Type.Results)
	info.FromRegistry[[2]string{src, tgt}] = ConvEntry{Name: d.Name.Name, Fallible: fallible}
}

// resolveMethod records a method under its receiver type name (star-stripped, so
// value- and pointer-receiver methods share a key).
func (info *Info) resolveMethod(d *ast.FuncDecl) {
	if d.Name == nil || d.Type == nil {
		return
	}
	recv := receiverType(d.Recv)
	if recv == "" {
		return
	}
	params := paramTypeListFL(d.Type.Params)
	results := paramTypeList(resultFields(d.Type.Results))
	raw := strings.TrimSpace(joinTypes(params) + " " + joinTypes(results))
	info.Methods[recv] = append(info.Methods[recv], Method{
		Name:        d.Name.Name,
		Sig:         joinTypes(params) + "|" + joinTypes(results),
		Raw:         raw,
		Arity:       resultArity(d.Type.Results),
		EndsInError: resultEndsInError(d.Type.Results),
	})
}

// funcSig reads the return mode, success/error types, and `?`-time arity of one
// function from its signature, mirroring analyze.analyzeSig: a Result/Option
// result is detected off the AST result type, and the Result/Option lowering
// fixes the lowered arity and ends-in-error.
func funcSig(name string, t *ast.FuncType) FuncSig {
	sig := FuncSig{Name: name, Mode: ModeNone}
	if t == nil {
		return sig
	}
	sig.Arity = resultArity(t.Results)
	sig.EndsInError = resultEndsInError(t.Results)

	if head, args := resultGeneric(t.Results); head != "" {
		switch {
		case head == "Result" && len(args) == 2:
			sig.T, sig.E = args[0], args[1]
			if sig.E == "error" {
				sig.Mode = ModeResult
			} else {
				sig.Mode = ModeResultClosed
			}
		case head == "Option" && len(args) == 1:
			sig.Mode = ModeOption
			sig.T = args[0]
		}
	}
	switch sig.Mode {
	case ModeResult:
		sig.Arity, sig.EndsInError = 2, true
	case ModeOption, ModeResultClosed:
		sig.Arity, sig.EndsInError = 1, false
	}
	return sig
}

// resultGeneric returns the head type name and its type-argument strings when the
// result list is a single unnamed generic result (Result[...]/Option[...]); it
// returns "" otherwise.
func resultGeneric(fl *ast.FieldList) (head string, args []string) {
	if fl == nil || len(fl.List) != 1 {
		return "", nil
	}
	f := fl.List[0]
	if len(f.Names) != 0 {
		return "", nil
	}
	switch x := f.Type.(type) {
	case *ast.IndexExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			return id.Name, []string{typeString(x.Index)}
		}
	case *ast.IndexListExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			out := make([]string, 0, len(x.Indices))
			for _, idx := range x.Indices {
				out = append(out, typeString(idx))
			}
			return id.Name, out
		}
	}
	return "", nil
}

// resultTarget mirrors analyze.parseReturn over the AST: a multi-result (or
// single named) list is the fallible `(T, error)` form whose target is the first
// result type; a single unnamed result is the bare target type.
func resultTarget(fl *ast.FieldList) (tgt string, fallible bool) {
	fields := resultFields(fl)
	if len(fields) == 0 {
		return "", false
	}
	if len(fields) > 1 {
		return typeString(fields[0].Type), true
	}
	return typeString(fields[0].Type), false
}

// resultArity counts the values a result list yields, expanding shared-type
// groups (an empty list is 0; `(int, error)` is 2; `(a, b int, err error)` is 3).
func resultArity(fl *ast.FieldList) int {
	n := 0
	for _, f := range resultFields(fl) {
		if c := len(f.Names); c > 0 {
			n += c
		} else {
			n++
		}
	}
	return n
}

// resultEndsInError reports whether the last result's type is `error`.
func resultEndsInError(fl *ast.FieldList) bool {
	fields := resultFields(fl)
	if len(fields) == 0 {
		return false
	}
	return typeString(fields[len(fields)-1].Type) == "error"
}

// resultFields returns the result field list (nil-safe).
func resultFields(fl *ast.FieldList) []*ast.Field {
	if fl == nil {
		return nil
	}
	return fl.List
}

// firstParamType returns the type of the first parameter (expanded so a shared
// group's first name still yields the type), or "" when there is none.
func firstParamType(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	return typeString(fl.List[0].Type)
}

// paramTypeList renders each field's type once per name (or once when unnamed),
// dropping parameter names — the canonical type sequence used for method sigs.
func paramTypeList(fields []*ast.Field) []string {
	var out []string
	for _, f := range fields {
		typ := typeString(f.Type)
		if c := len(f.Names); c > 0 {
			for range f.Names {
				out = append(out, typ)
			}
		} else {
			out = append(out, typ)
		}
	}
	return out
}

func paramTypeListFL(fl *ast.FieldList) []string {
	if fl == nil {
		return nil
	}
	return paramTypeList(fl.List)
}

func joinTypes(types []string) string { return strings.Join(types, ",") }

// receiverType returns the bare receiver type name of a method receiver
// `(r T)` / `(r *T)`, stripped of a leading "*"; "" when absent.
func receiverType(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	return strings.TrimPrefix(typeString(fl.List[0].Type), "*")
}

// typeString renders a type expression to source-equivalent, whitespace-canonical
// text. It is comma-safe (commas inside a func type or generic argument list stay
// inside the rendered type), which is the whole point of resolving off the AST.
func typeString(x ast.Expr) string {
	switch x := x.(type) {
	case nil:
		return ""
	case *ast.Ident:
		return x.Name
	case *ast.BasicLit:
		return x.Value
	case *ast.SelectorExpr:
		return typeString(x.X) + "." + selName(x.Sel)
	case *ast.StarExpr:
		return "*" + typeString(x.X)
	case *ast.ParenExpr:
		return "(" + typeString(x.X) + ")"
	case *ast.ArrayType:
		if x.Len != nil {
			return "[" + typeString(x.Len) + "]" + typeString(x.Elt)
		}
		return "[]" + typeString(x.Elt)
	case *ast.MapType:
		return "map[" + typeString(x.Key) + "]" + typeString(x.Value)
	case *ast.IndexExpr:
		return typeString(x.X) + "[" + typeString(x.Index) + "]"
	case *ast.IndexListExpr:
		parts := make([]string, 0, len(x.Indices))
		for _, idx := range x.Indices {
			parts = append(parts, typeString(idx))
		}
		return typeString(x.X) + "[" + strings.Join(parts, ", ") + "]"
	case *ast.Ellipsis:
		if x.Elt != nil {
			return "..." + typeString(x.Elt)
		}
		return "..."
	case *ast.ChanType:
		switch x.Dir {
		case ast.RecvOnly:
			return "<-chan " + typeString(x.Value)
		case ast.SendOnly:
			return "chan<- " + typeString(x.Value)
		default:
			return "chan " + typeString(x.Value)
		}
	case *ast.FuncType:
		return "func" + funcTypeString(x)
	case *ast.InterfaceType:
		if x.Methods == nil || len(x.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{ ... }"
	case *ast.StructType:
		if x.Fields == nil || len(x.Fields.List) == 0 {
			return "struct{}"
		}
		return "struct{ ... }"
	default:
		return ""
	}
}

// funcTypeString renders the "(params) results" tail of a func type.
func funcTypeString(t *ast.FuncType) string {
	var b strings.Builder
	b.WriteString("(")
	b.WriteString(strings.Join(fieldTypes(t.Params), ", "))
	b.WriteString(")")
	if t.Results != nil && len(t.Results.List) > 0 {
		res := fieldTypes(t.Results)
		if len(t.Results.List) > 1 || len(t.Results.List[0].Names) > 0 {
			b.WriteString(" (")
			b.WriteString(strings.Join(res, ", "))
			b.WriteString(")")
		} else {
			b.WriteString(" ")
			b.WriteString(res[0])
		}
	}
	return b.String()
}

// fieldTypes renders each field of a list as `name1, name2 Type` (or bare `Type`
// when unnamed) — used inside a func type's parameter/result rendering.
func fieldTypes(fl *ast.FieldList) []string {
	if fl == nil {
		return nil
	}
	out := make([]string, 0, len(fl.List))
	for _, f := range fl.List {
		typ := typeString(f.Type)
		if len(f.Names) > 0 {
			names := make([]string, 0, len(f.Names))
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
			out = append(out, strings.Join(names, ", ")+" "+typ)
		} else {
			out = append(out, typ)
		}
	}
	return out
}

func selName(id *ast.Ident) string {
	if id == nil {
		return ""
	}
	return id.Name
}

package ast

// This file gives the AST a deterministic textual rendering — an s-expression
// dump — used by the parser's snapshot tests (US-025) to pin the structural
// shape produced for each goal construct. It is intentionally structure-only:
// source positions (token.Pos) are omitted so a snapshot records the tree's
// shape, not byte offsets that shift whenever surrounding whitespace changes.
//
// The renderer is reflection-driven over each node's exported fields so it
// automatically covers every current and future node type without a parallel
// type switch (unlike Walk). Determinism comes for free: struct field order is
// fixed, and no map iteration is involved. reflect/strings/fmt are stdlib, so
// the zero-dependency rule still holds.

import (
	"fmt"
	"reflect"
	"strings"

	"goal/internal/token"
)

var (
	posType  = reflect.TypeOf(token.Pos{})
	kindType = reflect.TypeOf(token.Kind(0))
	modType  = reflect.TypeOf(FuncPlain)
	dirType  = reflect.TypeOf(SendRecv)
)

// Sexpr renders an AST node as a deterministic, indented s-expression. Source
// positions (token.Pos fields) are omitted so the form pins structure rather
// than byte offsets; token.Kind operator/keyword fields render via String(),
// and zero/empty fields are elided for compactness. The output is stable across
// runs for a given tree, which makes it suitable for snapshot tests. The result
// ends with a trailing newline.
func Sexpr(n Node) string {
	var b strings.Builder
	d := &sexprDumper{b: &b}
	if n == nil {
		b.WriteString("nil\n")
		return b.String()
	}
	d.dump(reflect.ValueOf(n), 0)
	b.WriteByte('\n')
	return b.String()
}

type sexprDumper struct{ b *strings.Builder }

func indent(n int) string { return strings.Repeat("  ", n) }

func (d *sexprDumper) dump(v reflect.Value, depth int) {
	switch v.Kind() {
	case reflect.Invalid:
		d.b.WriteString("nil")
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			d.b.WriteString("nil")
			return
		}
		d.dump(v.Elem(), depth)
	case reflect.Struct:
		d.dumpStruct(v, depth)
	case reflect.Slice:
		d.dumpSlice(v, depth)
	case reflect.String:
		fmt.Fprintf(d.b, "%q", v.String())
	case reflect.Bool:
		fmt.Fprintf(d.b, "%v", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		d.dumpInt(v)
	default:
		fmt.Fprintf(d.b, "%v", v.Interface())
	}
}

// dumpInt renders an integer-kinded value, naming the goal enums (token.Kind,
// FuncMod, ChanDir) so snapshots read meaningfully instead of showing raw ints.
func (d *sexprDumper) dumpInt(v reflect.Value) {
	switch v.Type() {
	case kindType:
		d.b.WriteString(v.Interface().(token.Kind).String())
	case modType:
		d.b.WriteString(funcModName(FuncMod(v.Int())))
	case dirType:
		d.b.WriteString(chanDirName(ChanDir(v.Int())))
	default:
		fmt.Fprintf(d.b, "%d", v.Int())
	}
}

func (d *sexprDumper) dumpStruct(v reflect.Value, depth int) {
	t := v.Type()
	d.b.WriteString("(" + t.Name())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Type == posType {
			continue // positions pin offsets, not structure — omit them
		}
		fv := v.Field(i)
		if isEmptyField(fv) {
			continue
		}
		d.b.WriteString("\n" + indent(depth+1) + "." + f.Name + " ")
		d.dump(fv, depth+1)
	}
	d.b.WriteString(")")
}

func (d *sexprDumper) dumpSlice(v reflect.Value, depth int) {
	if v.Len() == 0 {
		d.b.WriteString("[]")
		return
	}
	d.b.WriteString("[")
	for i := 0; i < v.Len(); i++ {
		d.b.WriteString("\n" + indent(depth+1))
		d.dump(v.Index(i), depth+1)
	}
	d.b.WriteString("\n" + indent(depth) + "]")
}

// isEmptyField reports whether a struct field should be elided from the dump:
// nil pointers/interfaces, empty slices/maps, and zero scalars (the zero
// token.Pos is already skipped by type). This keeps snapshots focused on the
// fields a node actually carries.
func isEmptyField(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

func funcModName(m FuncMod) string {
	switch m {
	case FuncPlain:
		return "FuncPlain"
	case FuncFrom:
		return "FuncFrom"
	case FuncDerive:
		return "FuncDerive"
	default:
		return fmt.Sprintf("FuncMod(%d)", int(m))
	}
}

func chanDirName(dir ChanDir) string {
	switch dir {
	case SendRecv:
		return "SendRecv"
	case SendOnly:
		return "SendOnly"
	case RecvOnly:
		return "RecvOnly"
	default:
		return fmt.Sprintf("ChanDir(%d)", int(dir))
	}
}

package interp

// These tests prove US-012 "Eval enum construction": the interpreter constructs
// enum variant values — data-less (`Status.Pending`) and payload-carrying
// (`Status.Active(since: now())`) — into the universal tagged-union Value, and
// reads their payload fields back by name. They drive a real parsed+resolved
// 01-enums-shaped program through the direct-evalExpr seam (newInterp + call,
// from call_test.go). Enums exist at runtime as tagged unions, NOT the Go
// backend's optimizations.

import (
	"strings"
	"testing"

	"goal/internal/ast"
)

const enumProgram = `package main

type Time = int64

func now() Time { return 7 }

enum Status {
	Pending
	Active { since: Time }
	Cancelled { reason: string, at: Time }
}

func pending() Status { return Status.Pending }
func active() Status { return Status.Active(since: now()) }
func cancelled() Status { return Status.Cancelled(reason: "timeout", at: now()) }
`

// wantVariant asserts a value is a variant with the expected type id and tag.
func wantVariant(t *testing.T, v Value, typeID, tag string) {
	t.Helper()
	if v.Kind != KindVariant || v.Variant == nil {
		t.Fatalf("value kind = %s, want variant", v.Kind)
	}
	if v.Variant.TypeID != typeID {
		t.Errorf("TypeID = %q, want %q", v.Variant.TypeID, typeID)
	}
	if v.Variant.Tag != tag {
		t.Errorf("Tag = %q, want %q", v.Variant.Tag, tag)
	}
}

func TestConstructDataLessVariant(t *testing.T) {
	ip := newInterp(t, enumProgram)
	v := evalFn(t, ip, "pending")
	wantVariant(t, v, "Status", "Pending")
	if len(v.Variant.Fields) != 0 {
		t.Errorf("data-less variant has %d fields, want 0", len(v.Variant.Fields))
	}
}

func TestConstructPayloadVariantSingleField(t *testing.T) {
	ip := newInterp(t, enumProgram)
	v := evalFn(t, ip, "active")
	wantVariant(t, v, "Status", "Active")
	since, ok := v.Field("since")
	if !ok {
		t.Fatalf("Active is missing the `since` field")
	}
	if !since.Equal(IntVal(7)) {
		t.Errorf("since = %s, want 7", since)
	}
}

func TestConstructPayloadVariantMultiField(t *testing.T) {
	ip := newInterp(t, enumProgram)
	v := evalFn(t, ip, "cancelled")
	wantVariant(t, v, "Status", "Cancelled")
	reason, ok := v.Field("reason")
	if !ok {
		t.Fatalf("Cancelled is missing the `reason` field")
	}
	if !reason.Equal(StrVal("timeout")) {
		t.Errorf("reason = %s, want \"timeout\"", reason)
	}
	at, ok := v.Field("at")
	if !ok {
		t.Fatalf("Cancelled is missing the `at` field")
	}
	if !at.Equal(IntVal(7)) {
		t.Errorf("at = %s, want 7", at)
	}
}

func TestConstructUnknownVariantIsRefused(t *testing.T) {
	ip := newInterp(t, enumProgram)
	// A payload construction of a non-existent variant of a known enum.
	lit := &ast.VariantLit{
		Enum:    &ast.Ident{Name: "Status"},
		Variant: &ast.Ident{Name: "Nope"},
		Args:    []ast.Expr{&ast.LabeledArg{Label: &ast.Ident{Name: "x"}, Value: intLit("1")}},
	}
	_, err := ip.evalExpr(lit, ip.root)
	if err == nil {
		t.Fatalf("constructing Status.Nope: expected an error, got none")
	}
	if !strings.Contains(err.Error(), "Status") || !strings.Contains(err.Error(), "Nope") {
		t.Errorf("error %q does not name the enum and bad variant", err)
	}
}

func TestConstructUnknownEnumIsRefused(t *testing.T) {
	ip := newInterp(t, enumProgram)
	lit := &ast.VariantLit{
		Enum:    &ast.Ident{Name: "Bogus"},
		Variant: &ast.Ident{Name: "X"},
		Args:    []ast.Expr{&ast.LabeledArg{Label: &ast.Ident{Name: "y"}, Value: intLit("1")}},
	}
	_, err := ip.evalExpr(lit, ip.root)
	if err == nil {
		t.Fatalf("constructing Bogus.X: expected an error, got none")
	}
	if !strings.Contains(err.Error(), "Bogus") {
		t.Errorf("error %q does not name the unknown enum", err)
	}
}

// Package sema holds the semantic information the AST back-ends consume: the
// name-keyed facts (enums, structs, signatures, the from-registry, methods)
// derived by walking the goal AST, plus the correctness checks layered on them.
//
// This is the Phase 2 replacement for the token-scanning internal/analyze: where
// analyze rebuilds structure from a flat token stream (and so mis-resolves any
// construct token scanning cannot model — e.g. a struct field whose type carries
// an embedded comma), sema reads structure off the parsed tree, where those facts
// are correct by construction. Resolve (resolve.go) populates Info by AST walk;
// the later sema stories (US-029..US-031) add the checks layered on these facts.
package sema

// Mode is a function's error-propagation shape, read from its return type. The
// constant order mirrors analyze.Mode so the two resolvers' modes line up.
type Mode int

const (
	// ModeNone returns neither Result nor Option.
	ModeNone Mode = iota
	// ModeResult is the open-E Result[T, error], lowered to native (T, error).
	ModeResult
	// ModeResultClosed is a closed-E Result[T, E] (E is not error), lowered to the
	// generic sum encoding Ok[T,E]/Err[T,E].
	ModeResultClosed
	// ModeOption is Option[T], lowered to *T.
	ModeOption
)

// FuncSig is the analyzed return signature of one function.
type FuncSig struct {
	Name string
	Mode Mode
	T    string // success type (the T in Result[T, E] or Option[T])
	E    string // error type (the E in Result[T, E]); "" for Option/none
	// Arity is the number of values the function returns at `?`-lowering time
	// (the lowered count for Result/Option), 0 for void/unknown.
	Arity int
	// EndsInError reports whether the function's last result is a failure a `?`
	// can propagate: a Result, a `func(…) error`, or a tuple ending in error.
	EndsInError bool
}

// Field is one variant or struct field: the goal field name and its type
// expression rendered from the AST (whitespace-canonical, comma-safe).
type Field struct {
	Name string
	Type string
}

// Variant is one enum variant; Fields is empty for a data-less variant.
type Variant struct {
	Name   string
	Fields []Field
}

// Enum is a closed sum type: its variants plus the name/field membership sets the
// match and construction lowering consult.
type Enum struct {
	Name     string
	Variants []Variant
	VSet     map[string]bool            // variant-name set
	FieldSet map[string]map[string]bool // variant -> its field-name set
}

// ConvEntry is one `from func`/`derive func` conversion: its name and whether it
// is fallible (returns `(T, error)` rather than a bare T).
type ConvEntry struct {
	Name     string
	Fallible bool
}

// Method is one method declared on a concrete type: its name, a normalized
// signature (param types | result types, names dropped), the raw param+result
// text, plus the `?`-relevant arity and ends-in-error facts.
type Method struct {
	Name        string
	Sig         string
	Raw         string
	Arity       int
	EndsInError bool
	// Return is the method's full analyzed return signature, including the
	// Result/Option mode (and success/error types). A `recv.Method()?` callee
	// resolves to this so the `?` lowering sees the real Result/error shape, not
	// just the raw arity. Arity/EndsInError above mirror Return's facts.
	Return FuncSig
}

// Info is the resolved semantic information for one goal file: the name-keyed
// symbol facts a back-end needs to lower goal constructs. Populated by Resolve
// (resolve.go); New returns an empty placeholder for the plain-Go path that does
// not yet read these facts.
type Info struct {
	// FuncSignatures maps a function name to its analyzed return signature.
	FuncSignatures map[string]FuncSig
	// Enums maps an enum type name to its analyzed variants.
	Enums map[string]*Enum
	// Sealed is the set of interface names declared `sealed interface`.
	Sealed map[string]bool
	// SealedImpls is the implementor registry: an interface name mapped to the
	// concrete types that name it in an `implements` clause (`type T struct
	// implements I`). It is kept distinct from Sealed because `implements` clauses
	// also target ordinary (non-sealed) interfaces, whose feature-07 satisfaction
	// CheckImplements verifies by short-circuiting on `Sealed[iface]`; folding the
	// two together would make an ordinary `implements` target look sealed and skip
	// that check. Only entries whose key is also in Sealed form the sealed-interface
	// implementor registry a sealed `match` is checked and lowered against; other
	// entries are inert. Unioned across a package's files by Merge.
	SealedImpls map[string][]string
	// Structs maps a `type X struct {…}` name to its ordered fields.
	Structs map[string][]Field
	// FromRegistry maps a (source type, target type) pair to the conversion func
	// (`from func`/`derive func`) between them.
	FromRegistry map[[2]string]ConvEntry
	// Methods maps a concrete type name to the methods declared on it (value- and
	// pointer-receiver alike, keyed by the star-stripped receiver type name).
	Methods map[string][]Method
	// ForeignMethods maps `pkg.Type.Method` (the goal-source spelling of an
	// imported type's method) to its return signature, so a check can resolve a
	// `recv.Method()?` whose receiver is an out-of-package type without
	// analyze.Tables. Populated by EnrichForeign (foreign.go); a foreign entry
	// carries only the `?`-relevant facts (Arity, EndsInError), with Mode left
	// ModeNone — mirroring analyze.Tables.ForeignMethods.
	ForeignMethods map[string]FuncSig
	// Interfaces maps an in-file interface type name to its directly declared
	// methods (name + normalized signature). Embedded interfaces are recorded
	// separately in EmbeddedIfaces and folded in by the implements check.
	Interfaces map[string][]Method
	// EmbeddedIfaces maps an in-file interface type name to the names of the
	// interfaces it embeds (a qualified name like "io.Reader" survives intact).
	EmbeddedIfaces map[string][]string
}

// New returns an empty *Info. Back-ends that do not yet need semantic facts (the
// plain-Go subset) call New to satisfy the Backend.Emit signature; its maps are
// nil (read-only callers range over them harmlessly).
func New() *Info {
	return &Info{}
}

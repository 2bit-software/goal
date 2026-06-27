package interp

// This file defines the interpreter's lexical environment: a parent-linked
// chain of name -> Value bindings modelling nested lexical scopes. Inner scopes
// shadow outer ones, and lookups fall through to enclosing scopes until a
// binding is found or the chain is exhausted. The evaluation stories
// (US-004 onward) read and write program state through this Env.

// NotFoundError reports that a name was not bound in any scope along the
// environment chain. It carries the missing name so callers can surface a
// located "undefined: x" diagnostic rather than a silent zero Value.
type NotFoundError struct {
	Name string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string { return "undefined: " + e.Name }

// Env is a single lexical scope: its own bindings plus a link to the enclosing
// scope. The root scope has a nil parent.
type Env struct {
	vars   map[string]Value
	parent *Env
}

// NewEnv constructs a fresh root scope with no parent.
func NewEnv() *Env {
	return &Env{vars: map[string]Value{}}
}

// NewChild opens an inner scope whose parent is the receiver. Bindings defined
// in the child shadow same-named bindings in the parent without mutating them;
// the parent binding is visible again once the child is discarded.
func (e *Env) NewChild() *Env {
	return &Env{vars: map[string]Value{}, parent: e}
}

// Define binds name to v in THIS scope, replacing any binding already present
// in this scope. It never writes to a parent scope.
func (e *Env) Define(name string, v Value) {
	if e.vars == nil {
		e.vars = map[string]Value{}
	}
	e.vars[name] = v
}

// Lookup resolves name by checking this scope first, then walking toward the
// root through parent scopes, returning the first binding found. If the name is
// bound in no scope along the chain it returns the zero Value and a
// *NotFoundError naming the missing symbol.
func (e *Env) Lookup(name string) (Value, error) {
	for s := e; s != nil; s = s.parent {
		if v, ok := s.vars[name]; ok {
			return v, nil
		}
	}
	return Value{}, &NotFoundError{Name: name}
}

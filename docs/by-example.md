# goal by Example

A feature-by-feature reference for programmers: each section shows **what a feature
is**, **what it unlocks** (the silent failure it converts into a located error),
**what we're after** (the design goal), a runnable `.goal` snippet, and **what it
transpiles to** — the exact, gofmt'd Go the front-end emits today.

goal is a correctness-oriented dialect of Go that transpiles to plain Go. It exists to
give AI coding agents (and humans) **fast, located, machine-checkable feedback** —
biasing every feature toward turning *silent runtime failures* and *human-judgment
calls* into *compiler or test errors* the model can act on. It stays as close to Go's
shape as possible so the model's existing fluency carries over; every divergence earns
its keep.

> **Two principles run through every feature** (see `goal-design-spec.md`):
> - **The feedback principle.** A feature's value = how much correctness signal it
>   produces, how cheaply, how early, how structured. The ranking is
>   **tests > compiler errors > human prose**. Bias every decision toward moving errors
>   out of "silent runtime failure" and into "fast, located, structured feedback."
> - **The familiarity principle.** Stay Go-shaped by default; when a feature Go *lacks*
>   forces divergence, land on another widely-seen idiom (Rust/Swift/Scala/TS) rather
>   than inventing novel syntax.

> **Front-end status: complete.** All 11 features compose, and every snippet here
> round-trips to independently-compiling Go (the `.go.expected` golden files in
> `features/*/examples/` and `testdata/` are generated *from* the tool and verified to
> compile). The **checker** — which turns each guarantee below into a static diagnostic
> — is the next workstream; today the front-end lowers proven-valid input and defers
> anything it can't resolve. See `NEXT-SESSION.md` and `TODO.md` for status.

This doc is the quick-ingest companion to the [design spec](../goal-design-spec.md)
(authoritative prose), the [decisions ledger](../DECISIONS.md), and the per-feature
reference transpilers in [`features/`](../features/). It is also the intended source
for a future website with a live playground — each section is a self-contained
goal→Go pair ready to drop into an editor.

---

## Contents

- **Sum types & control flow:** [Enums](#01-enums) · [Match](#02-match)
- **Errors & absence:** [Result (open-E)](#03-result-open-e) · [Option](#04-option) ·
  [`?` propagation](#05--propagation) · [Result (closed-E)](#06-result-closed-e)
- **Contracts on types:** [implements](#07-implements) ·
  [No zero value](#08-no-zero-value)
- **Runtime & test feedback:** [assert](#10-assert) · [Doctests](#11-doctests) ·
  [derive-convert](#12-derive-convert)
- **Composition:** [All eleven in one file](#composition-all-eleven-in-one-file)

| # | Feature | Surface | Lowers to |
|---|---------|---------|-----------|
| 01 | [enums](#01-enums) | `enum E { A; B { x: T } }` | sealed interface + per-variant struct + marker |
| 02 | [match](#02-match) | `match e { E.A => … }` | exhaustive type switch (panicking default) |
| 03 | [Result (open-E)](#03-result-open-e) | `Result[T, error]` | native `(T, error)` tuple |
| 04 | [Option](#04-option) | `Option[T]` | `*T` (nil = None) |
| 05 | [`?` propagation](#05--propagation) | `x := f()?` | unwrap-or-early-return |
| 06 | [Result (closed-E)](#06-result-closed-e) | `Result[T, MyErr]` | generic sum `Ok[T,E]`/`Err[T,E]` + From-conversion |
| 07 | [implements](#07-implements) | `type T struct implements I { … }` | compile-time assertion `var _ I = T{}` |
| 08 | [no-zero-value](#08-no-zero-value) | `T{a: x, ...defaults}` | explicit per-field zero expansion |
| 10 | [assert](#10-assert) | `assert cond, "msg", args` | runtime `if !(cond) { panic(...) }` |
| 11 | [doctests](#11-doctests) | `/// >>> f(2)` / `/// 4` | a generated `_test.go` |
| 12 | [derive-convert](#12-derive-convert) | `derive func g(s S) T` | field-by-field conversion via a `from func` registry |

---

# Sum types & control flow

## 01. Enums

A **closed** tagged union: `enum Name { ... }` lists every variant in one place,
optionally with brace-named payload fields. Construction is qualified —
`Name.Variant`, `Name.Variant(field: value)`.

```goal
enum Name {
    Variant1
    Variant2 { field: Type }
    Variant3 { f1: Type, f2: Type }
}
```

**Unlocks:** a variant set the compiler knows is *complete* and not externally
extensible — which is what makes `match` exhaustiveness (feature 02) provable. Adding a
variant later turns every now-incomplete `match` into a located error instead of a
silent fall-through.

**Goal:** a closed enum block is self-evidently a frozen set, co-located in one place,
so the compiler can enforce exhaustive handling and stop external code from secretly
breaking that completeness.

```goal name=traffic.goal
package traffic

enum Light {
    Red
    Yellow
    Green
}

func first() Light {
    return Light.Red
}
```

Transpiles to:

```go
package traffic

type Light interface{ isLight() }

type Light_Red struct{}
type Light_Yellow struct{}
type Light_Green struct{}

func (Light_Red) isLight()    {}
func (Light_Yellow) isLight() {}
func (Light_Green) isLight()  {}

func first() Light {
	return Light(Light_Red{})
}
```

**Lowers to:** a sealed interface (`interface{ isLight() }`) with one per-variant struct
and an unexported marker method each — closing the set in Go's own type system.

## 02. Match

A dedicated `match` construct (deliberately *not* `switch`) with `=>` arms. Arms bind
the variant payload (`E.Variant(b) => use(b)`); an explicit `_` arm is the conscious
opt-out for partial handling. `match` can be a statement or yield a value.

```goal
match value {
    Enum.Variant => stmt
    Enum.Variant(binding) => expr_or_stmt
    _ => expr_or_stmt   // optional, explicit rest-arm
}
```

**Unlocks:** exhaustiveness checking — the checker proves every variant is handled, so a
forgotten case is a located compile error, not silent wrong behavior. `_` is a
deliberate marker for "I really do mean to handle the rest," not a reflexive default.

**Goal:** a distinct `match` syntax (so the model doesn't reflexively reach for Go's
`default:` and silently defeat exhaustiveness) that converts case-omission into
compile-time feedback.

```goal name=status_rest.goal
package status

type Time = int64

func render(t Time)    {}
func showPlaceholder() {}

enum Status {
    Pending
    Active { since: Time }
    Cancelled { reason: string, at: Time }
}

func handle2(s Status) {
    match s {
        Status.Active(a) => render(a.since)
        _ => showPlaceholder()
    }
}
```

Transpiles to:

```go
package status

type Time = int64

func render(t Time)    {}
func showPlaceholder() {}

type Status interface{ isStatus() }

type Status_Pending struct{}
type Status_Active struct {
	Since Time
}
type Status_Cancelled struct {
	Reason string
	At     Time
}

func (Status_Pending) isStatus()   {}
func (Status_Active) isStatus()    {}
func (Status_Cancelled) isStatus() {}

func handle2(s Status) {
	switch __gop_v := s.(type) {
	case Status_Active:
		render(__gop_v.Since)
	default:
		showPlaceholder()
	}
}
```

**Lowers to:** a Go type switch over the variant structs. An exhaustive `match` emits a
defensive panicking `default`; an explicit `_` arm becomes a real `default:` block.

---

# Errors & absence

## 03. Result (open-E)

A fallible function returns a `Result[T, E]` as its **whole** return value — a sum of
`Result.Ok(T)` / `Result.Err(E)`. When `E` is the `error` interface (the *open* case),
this is the everyday error path.

```goal
func name(...) Result[T, error] {
    if cond {
        return Result.Err(err)
    }
    return Result.Ok(value)
}
```

**Unlocks:** Go's single biggest silent-failure pattern — `cfg, _ := parse(s)` — goes
away. The error is a *branch of the value you hold*, not a separate return you can
forget. You reach the success value only by handling both branches (or with `?`,
feature 05).

**Goal:** make the error and success path a single sum so the success value is
unreachable without going through error handling — turning accidental error discards
into compile errors.

```goal name=result_parse.goal
package config

import "errors"

type Config struct {
    Raw string
}

func parse(s string) Result[Config, error] {
    if s == "" {
        return Result.Err(errors.New("empty input"))
    }
    return Result.Ok(Config{Raw: s})
}
```

Transpiles to:

```go
package config

import "errors"

type Config struct {
	Raw string
}

func parse(s string) (__gop_ok Config, __gop_err error) {
	if s == "" {
		return __gop_ok, errors.New("empty input")
	}
	return Config{Raw: s}, nil
}
```

**Lowers to:** Go's native `(T, error)` pair. `Result.Ok(v)` → `(v, nil)`,
`Result.Err(e)` → `(zero, e)`, and a `match` on the result becomes an `err != nil`
branch. Zero runtime cost over hand-written Go.

## 04. Option

`Option[T]` is a sum of `Option.Some(T)` / `Option.None` — the same machinery as
`Result`, used for *absence*. Its inner value is reachable only by destructuring through
a `match`.

```goal
func find(id ID) Option[User] {
    u, ok := lookup(id)
    if !ok {
        return Option.None
    }
    return Option.Some(u)
}
```

**Unlocks:** nil-dereference — Go's *other* great silent-failure class — is eliminated.
The value is unreachable without passing through a `match` arm, so every access is
provably guarded.

**Goal:** for nearly free (it reuses the sum-type machinery from features 01–03), the
checker converts a potential nil-dereference into a forced, visible `match`.

```goal name=option_find.goal
package users

type ID int

type User struct {
    Name string
}

func lookup(id ID) (User, bool) {
    return User{}, false
}

func find(id ID) Option[User] {
    u, ok := lookup(id)
    if !ok {
        return Option.None
    }
    return Option.Some(u)
}

func greet(u User) {}
func prompt()      {}

func handle(id ID) {
    match find(id) {
        Option.Some(u) => greet(u)
        Option.None => prompt()
    }
}
```

Transpiles to:

```go
package users

type ID int

type User struct {
	Name string
}

func lookup(id ID) (User, bool) {
	return User{}, false
}

func find(id ID) *User {
	u, ok := lookup(id)
	if !ok {
		return nil
	}
	return &u
}

func greet(u User) {}
func prompt()      {}

func handle(id ID) {
	if __gop_o := find(id); __gop_o != nil {
		u := *__gop_o
		greet(u)
	} else {
		prompt()
	}
}
```

**Lowers to:** `*T`. `Option.None` → `nil`, `Option.Some(v)` → `&v`, and `match` becomes
the nil-check `if p != nil { … } else { … }`.

## 05. `?` propagation

Postfix `?` on a `Result` or `Option` value: if it's `Err`/`None`, early-return it from
the enclosing function; otherwise unwrap to the `Ok`/`Some` value. It makes the safe
path the *short* path.

```goal
func loadConfig(p string) Result[Config, error] {
    raw := readFile(p)?   // Err → return it; else unwrap to []byte
    cfg := parse(raw)?
    return Result.Ok(cfg)
}
```

**Unlocks:** error/absence handling becomes a straight line through the function instead
of a pyramid of matches — so `Result`/`Option` are *cheaper* than `if err != nil`, and
the model reaches for them instead of routing around the safety.

**Goal:** `?` is the affordance that makes the safe types the path of least resistance.
The glyph is the conventional Rust/Swift one — non-negotiable for recognition.

```goal name=qprop_result.goal
package config

type Config struct {
    Raw string
}

func readFile(p string) Result[[]byte, error] {
    return Result.Ok([]byte(p))
}

func parse(raw []byte) Result[Config, error] {
    return Result.Ok(Config{Raw: string(raw)})
}

func loadConfig(p string) Result[Config, error] {
    raw := readFile(p)?
    cfg := parse(raw)?
    return Result.Ok(cfg)
}
```

Transpiles to:

```go
package config

type Config struct {
	Raw string
}

func readFile(p string) (__gop_ok []byte, __gop_err error) {
	return []byte(p), nil
}

func parse(raw []byte) (__gop_ok Config, __gop_err error) {
	return Config{Raw: string(raw)}, nil
}

func loadConfig(p string) (__gop_ok Config, __gop_err error) {
	raw, __gop_err := readFile(p)
	if __gop_err != nil {
		return __gop_ok, __gop_err
	}
	cfg, __gop_err := parse(raw)
	if __gop_err != nil {
		return __gop_ok, __gop_err
	}
	return cfg, nil
}
```

**Lowers to:** in a `Result[_, error]` function, `name := expr?` becomes
`name, __gop_err := expr; if __gop_err != nil { return __gop_ok, __gop_err }`. In an
`Option[_]` function it becomes a nil-check-and-return, then a deref.

## 06. Result (closed-E)

The error type `E` in `Result[T, E]` may instead be an `enum` (the *closed* case). The
failure set is then visible in the signature and exhaustively matchable. `?` across
mismatched error enums is bridged by a registered `from func` conversion. One unified
mechanism: open vs. closed differs only in whether `E` is `error` or an enum.

```goal
enum ParseError {
    Empty
    BadKey { key: string }
}

func parse(s string) Result[Config, ParseError] { ... }
```

**Unlocks:** the exact set of failures a function can produce is in its type, and a
`match` on the result is checked for exhaustiveness like any enum. Unlike open `error`,
the closed sum value survives at runtime, enabling precise failure handling.

**Goal:** reuse the existing `enum` + `Result` machinery so there's *one* error system
with a single knob (open ↔ closed), not a second parallel mechanism.

```goal name=qclosed_match.goal
package parse

enum ParseError {
    Empty
    BadKey { key: string }
}

type Config struct {
    Raw string
}

func parse(s string) Result[Config, ParseError] {
    if s == "" {
        return Result.Err(ParseError.Empty)
    }
    return Result.Ok(Config{Raw: s})
}

func run(c Config)        {}
func report(e ParseError) {}

func handle(input string) {
    match parse(input) {
        Result.Ok(cfg) => run(cfg)
        Result.Err(e) => report(e)
    }
}
```

Transpiles to:

```go
package parse

type Result[T, E any] interface{ isResult() }
type Ok[T, E any] struct{ Value T }
type Err[T, E any] struct{ Value E }

func (Ok[T, E]) isResult()  {}
func (Err[T, E]) isResult() {}

type ParseError interface{ isParseError() }

type ParseError_Empty struct{}
type ParseError_BadKey struct {
	Key string
}

func (ParseError_Empty) isParseError()  {}
func (ParseError_BadKey) isParseError() {}

type Config struct {
	Raw string
}

func parse(s string) Result[Config, ParseError] {
	if s == "" {
		return Err[Config, ParseError]{Value: ParseError(ParseError_Empty{})}
	}
	return Ok[Config, ParseError]{Value: Config{Raw: s}}
}

func run(c Config)        {}
func report(e ParseError) {}

func handle(input string) {
	switch __gop_e := parse(input).(type) {
	case Ok[Config, ParseError]:
		cfg := __gop_e.Value
		run(cfg)
	case Err[Config, ParseError]:
		e := __gop_e.Value
		report(e)
	default:
		panic("unreachable: non-exhaustive Result[Config, ParseError] (compiler invariant violated)")
	}
}
```

**Lowers to:** a generic sum encoding — `interface{ isResult() }` with `Ok[T,E]` and
`Err[T,E]` cases. `match` becomes a type switch with a defensive panic default; `?`
becomes type-switch-and-return, inserting a `from func` conversion in the `Err` arm when
error types differ. (Open-E and closed-E can coexist in one file — see
[composition](#composition-all-eleven-in-one-file).)

---

# Contracts on types

## 07. implements

The inline `implements` clause on a struct declares — at the type's own site — that the
struct satisfies one or more interfaces. Structural satisfaction is unchanged; this only
*adds* a checked assertion.

```goal
type Point struct implements Stringer { X int; Y int }   // any interface(s); comma-separated
```

**Unlocks:** Go's silent, unchecked-at-declaration interface satisfaction becomes a
compile error at the struct. A missing or mis-signed method is caught at the
declaration, not at a distant call site (or never).

**Goal:** convert an invisible "does this still satisfy the interface?" into located
feedback at the type itself — kept additive, so structural typing elsewhere is
untouched.

```goal name=value_recv.goal
package shapes

type Stringer interface {
    String() string
}

type Point struct implements Stringer {
    X int
    Y int
}

func (p Point) String() string {
    return "point"
}
```

Transpiles to:

```go
package shapes

type Stringer interface {
	String() string
}

type Point struct {
	X int
	Y int
}

var _ Stringer = Point{}

func (p Point) String() string {
	return "point"
}
```

**Lowers to:** the idiomatic Go compile-time assertion `var _ Stringer = Point{}` (or
`&Point{}` for pointer-receiver methods).

## 08. No zero value

Constructing a struct requires every declared field be set explicitly. The escape hatch
is an explicit trailing `...defaults`, which fills the unset fields with their zero
values — at most once per literal.

```goal
User{ name: "a", email: "b@c", ...defaults }   // fill the rest with zeros
User{ name: "a", email: "b@c", role: r, admin: t }   // complete literal
```

**Unlocks:** Go's notorious zero-value footgun — an unset field silently reading back as
`0`/`""`/`nil` — becomes a located compile error. Forgetting a field is caught by the
checker, not discovered in production.

**Goal:** attack a *named, Go-specific* weakness by making required-field construction
the default, with `...defaults` as a single, explicit, greppable opt-out.

```goal name=complete.goal
package users

type Role int

const (
	RoleMember Role = iota
	RoleAdmin
)

type User struct {
	name  string
	email string
	role  Role
	admin bool
}

// newAdmin sets every field explicitly — a complete literal passes through verbatim.
func newAdmin() User {
	return User{name: "root", email: "root@x", role: RoleAdmin, admin: true}
}
```

Transpiles to:

```go
package users

type Role int

const (
	RoleMember Role = iota
	RoleAdmin
)

type User struct {
	name  string
	email string
	role  Role
	admin bool
}

// newAdmin sets every field explicitly — a complete literal passes through verbatim.
func newAdmin() User {
	return User{name: "root", email: "root@x", role: RoleAdmin, admin: true}
}
```

**Lowers to:** a complete literal passes through verbatim; a `...defaults` literal
expands its safe fields to explicit per-field zero values (e.g. `User{name: name, role: 0,
active: false}`) and *rejects* any field whose zero is unsafe (see below). The completeness
guarantee is a checker concern — the emitted Go is ordinary.

## Rejecting an unsafe default

`...defaults` only fills fields whose zero value is *safe*. A field whose zero is a latent
hazard — a `nil` map (panics on write), a `nil` pointer (panics on deref), a `nil` channel or
func, or a sum type with no valid variant — is a **located compile error**, not a silent zero.
Set such a field explicitly, or use `Option[T]` for a genuinely-optional reference.

```goal name=defaults_unsafe.goal
package cache

type Store struct {
	name    string
	entries map[string]int
}

func newStore(name string) Store {
	return Store{name: name, ...defaults}
}
```

Rejected with:

```error
pass defaults: `...defaults` at 9:27 cannot default field `entries` of type `map[string]int`: a nil map panics on write — set it explicitly (e.g. `map[string]int{}`)
```

**Why:** a `nil` map reads fine but panics the moment something writes to it — exactly the
silent zero-value footgun this feature exists to close. `entries` has no safe default, so
construction must name it (`entries: map[string]int{}`); the genuinely-safe fields can still
ride `...defaults`.

# Runtime & test feedback

## 10. assert

`assert cond` checks a boolean invariant at runtime and panics on failure, with the
source expression text in the message. An optional printf-style explanation is appended.

```goal
assert cond                          // bare — auto message from the expression text
assert cond, "fmt", args...          // explanation appended
```

**Unlocks:** invariants the type system can't capture become located panics — the
offending condition is always visible in the failure, instead of a silent logical error
propagating.

**Goal:** apply the located-feedback principle at the band just below compile time —
encode invariants cheaply and familiarly so runtime failures point at the exact
condition.

```goal name=bank.goal
package bank

// withdraw asserts an invariant the type system can't capture (amount > 0). The
// bare form needs no message: the panic carries the source expression text.
func withdraw(balance int, amount int) int {
	assert amount > 0
	return balance - amount
}
```

Transpiles to:

```go
package bank

// withdraw asserts an invariant the type system can't capture (amount > 0). The
// bare form needs no message: the panic carries the source expression text.
func withdraw(balance int, amount int) int {
	if !(amount > 0) {
		panic("assertion failed: amount > 0")
	}
	return balance - amount
}
```

**Lowers to:** `if !(cond) { panic("assertion failed: " + cond) }`; the printf form
appends `fmt.Sprintf(...)` to the message.

## 11. Doctests

Executable examples live in `///` doc comments: a `>>> <expr>` line followed by its
expected output. They're extracted into a generated `_test.go` and run under
`go test` — so an example either passes as a test or fails to compile.

```goal
/// Adds two ints.
/// >>> add(2, 3)
/// 5
func add(a int, b int) int { ... }
```

**Unlocks:** documentation examples that *can't silently drift* — the example is a real
test, so a stale or non-running doctest is impossible by construction (it's a failing
test or a compile error).

**Goal:** put executable verification in the **top feedback band** (tests > compiler >
prose), co-located with the documentation it verifies. The hard requirement: *no way*
for a doctest to silently skip.

```goal name=add.goal
package mathx

/// Adds two ints.
/// >>> add(2, 3)
/// 5
func add(a int, b int) int {
	return a + b
}
```

Transpiles to (the `_test.go` side output, via `goalc -test`):

```go
package mathx

import "testing"

func TestDoctest_add_1(t *testing.T) {
	got := add(2, 3)
	want := 5
	if got != want {
		t.Errorf("doctest add: got %v, want %v", got, want)
	}
}
```

**Lowers to:** a generated `<base>_doctest_test.go` with one `TestDoctest_*` per example
— `got := <expr>; want := <expected>; if got != want { t.Errorf(...) }`. It's a
*side output*, emitted alongside the transpiled Go rather than inline.

## 12. derive-convert

`derive func` declarations build a struct-to-struct conversion field-by-field, pulling
each field from a registry of `from func` leaf conversions. Missing fields are a
compile error, not silent zeros. `_` intentionally skips a field; `...derive(src)` fills
the remainder from the registry.

```goal
from func uuidToString(u UUID) string { ... }   // leaf conversion (registry entry)
derive func toIDs(g Group) IDList               // bodyless: fill all fields
derive func g(s S) T {                           // with exceptions
    return T{ Field: expr, Skip: _, ...derive(s) }
}
```

**Unlocks:** the silent zero-value gaps of multi-layer codebases — a forgotten field in
a conversion, an unregistered type pair, an ambiguous narrowing — become located compile
errors. Every target field must be sourced from the registry or explicitly provided.

**Goal:** generalize feature 08's completeness principle to *type-pair* conversion — the
high-leverage case where forgotten fields read back as zero — so a conversion is provably
total.

```goal name=slice.goal
package conv

// Local stand-in (see to_storage.goal / SYNTAX.md).
type UUID struct{ s string }

func (u UUID) String() string { return u.s }

from func uuidToString(u UUID) string { return u.String() }

type Group struct {
	Name    string
	Members []UUID
}

type IDList struct {
	Name    string
	Members []string
}

// Container recursion (built-in deriver rule): Members []UUID -> []string is filled
// automatically from the registered UUID -> string; the user writes only the leaf.
derive func toIDs(g Group) IDList
```

Transpiles to:

```go
package conv

// Local stand-in (see to_storage.goal / SYNTAX.md).
type UUID struct{ s string }

func (u UUID) String() string { return u.s }

func uuidToString(u UUID) string { return u.String() }

type Group struct {
	Name    string
	Members []UUID
}

type IDList struct {
	Name    string
	Members []string
}

// Container recursion (built-in deriver rule): Members []UUID -> []string is filled
// automatically from the registered UUID -> string; the user writes only the leaf.
func toIDs(g Group) IDList {
	var out IDList
	out.Name = g.Name
	out.Members = make([]string, len(g.Members))
	for i := range g.Members {
		out.Members[i] = uuidToString(g.Members[i])
	}
	return out
}
```

**Lowers to:** `from func` strips to a plain `func`; `derive func` expands to
field-by-field assignments with registry lookups and container recursion;
`...derive(src)` expands to per-field conversions, skipping overridden and `_`-marked
fields.

---

# Composition: all eleven in one file

The headline claim is that the features **compose** — there's one front-end pipeline,
not eleven tools, and any combination transpiles in a single pass. This program uses
enums, match, Result (open-E), `?`, implements (both a sealed-interface marker and a
plain assertion), no-zero-value `...defaults`, and assert together:

```goal name=kitchen_sink.goal
package app

import "errors"

type Role int

type User struct {
    name   string
    role   Role
    active bool
}

sealed interface Shape {}

type Circle struct implements Shape {
    r float64
}

type Stringer interface {
    String() string
}

type Tag struct implements Stringer {
    v string
}

func (t Tag) String() string {
    return t.v
}

enum State {
    Idle
    Running { pid: int }
}

func square(x int) int {
    return x * x
}

func makeUser(name string, age int) Result[User, error] {
    assert age >= 0, "negative age: %d", age
    if name == "" {
        return Result.Err(errors.New("empty name"))
    }
    return Result.Ok(User{name: name, ...defaults})
}

func register(name string, age int) Result[User, error] {
    u := makeUser(name, age)?
    return Result.Ok(u)
}

func describe(s State) string {
    var label string = match s {
        State.Idle => "idle"
        State.Running(r) => "running"
    }
    return label
}
```

Transpiles to:

```go
package app

import "fmt"

import "errors"

type Role int

type User struct {
	name   string
	role   Role
	active bool
}

type Shape interface{ isShape() }

type Circle struct {
	r float64
}

func (Circle) isShape() {}

type Stringer interface {
	String() string
}

type Tag struct {
	v string
}

var _ Stringer = Tag{}

func (t Tag) String() string {
	return t.v
}

type State interface{ isState() }

type State_Idle struct{}
type State_Running struct {
	Pid int
}

func (State_Idle) isState()    {}
func (State_Running) isState() {}

func square(x int) int {
	return x * x
}

func makeUser(name string, age int) (__gop_ok User, __gop_err error) {
	if !(age >= 0) {
		panic("assertion failed: age >= 0: " + fmt.Sprintf("negative age: %d", age))
	}
	if name == "" {
		return __gop_ok, errors.New("empty name")
	}
	return User{name: name, role: 0, active: false}, nil
}

func register(name string, age int) (__gop_ok User, __gop_err error) {
	u, __gop_err := makeUser(name, age)
	if __gop_err != nil {
		return __gop_ok, __gop_err
	}
	return u, nil
}

func describe(s State) string {
	var label string
	switch s.(type) {
	case State_Idle:
		label = "idle"
	case State_Running:
		label = "running"
	default:
		panic("unreachable: non-exhaustive State (compiler invariant violated)")
	}
	return label
}
```

Notice how each feature lowers independently yet correctly side-by-side: `implements`
on `Circle` becomes a marker method (because `Shape` is a sealed interface) while
`implements` on `Tag` becomes a `var _` assertion (because `Stringer` is an ordinary
interface); the `assert` pulls in `fmt`; `...defaults` expands to `role: 0, active:
false`; and the value-position `match` writes into `label` with a panicking default. The
even harder case — open-E and closed-E `Result` in the *same* file — lives in
[`testdata/open_closed_mix.goal`](../testdata/open_closed_mix.goal).

---

# Maintaining this doc

Every goal→Go pair here is copied from the project's verified golden files:

- Per-feature snippets come from `features/NN-*/examples/*.goal` and their
  `*.go.expected` counterparts (each feature also has a `SYNTAX.md` and `TRANSPILE.md`).
- The composition snippet is `testdata/kitchen_sink.goal` / `.go.expected`.

Those `.go.expected` files are generated *from* the transpiler and verified to compile
independently (`go test ./...` builds the generated programs in a throwaway module). When
a feature's lowering changes, regenerate the goldens and re-sync the affected section
here. When the **checker** lands, add the located-diagnostic examples (the "what the
error looks like" half) beneath each feature's guarantee.

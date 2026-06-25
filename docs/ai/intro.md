goal is a thin dialect of Go that **transpiles source-to-source to plain Go**. It adds a
small set of correctness features that turn Go's *silent* failure classes — ignored errors,
nil derefs, forgotten struct fields, non-exhaustive switches — into **located compile/check-time
errors** or **tests**. You keep Go's runtime, GC, stdlib, and toolchain unchanged. Everything
you write in `.goal` lowers to idiomatic `.go` you could have written by hand. Your existing Go
fluency carries over directly; only the divergences documented here matter.

**Design bias (why a feature exists):** move every error from "silent runtime failure" into
"fast, located, structured feedback." Ranking: **tests > compiler errors > human prose**. Stay
Go-shaped; where Go lacks a feature, borrow the widely-seen idiom (Rust/Swift) rather than
invent syntax.

Every example below — its goal source and the Go it lowers to — was produced by transpiling it
with *this* binary at the moment you ran `goal ai`. It cannot be stale: change the language and
this guide changes with it.

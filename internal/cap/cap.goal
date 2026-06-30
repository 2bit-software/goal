// Package cap defines goscript's capability/authority model.
//
// goscript restricts power by what the host grants, not by offering a different
// language grammar (REWRITE-ARCHITECTURE.md §4): the runtime "removes"
// concurrency or I/O by the host not granting the corresponding capability. This
// package is that seam. v1 grants everything by default (see GrantAll, used by the
// default run path); denial is a host opt-in. Enforcement at effect sites is later
// runtime work — here the model only needs to answer membership.
package cap

// Capability names a single host authority the runtime can be granted or denied.
type Capability int

// The defined capabilities. Order is stable; allCapabilities mirrors it.
const (
	Stdout Capability = iota // write to standard output
	Stdin                    // read from standard input
	FileRead                 // read from the filesystem
	FileWrite                // write to the filesystem
	Net                      // open network connections
	Concurrency              // start goroutines / use channels
	Time                     // read the wall clock
	Env                      // read environment variables
)

// String returns the readable name of the capability.
func (c Capability) String() string {
	switch c {
	case Stdout:
		return "Stdout"
	case Stdin:
		return "Stdin"
	case FileRead:
		return "FileRead"
	case FileWrite:
		return "FileWrite"
	case Net:
		return "Net"
	case Concurrency:
		return "Concurrency"
	case Time:
		return "Time"
	case Env:
		return "Env"
	default:
		return "Capability(" + itoa(int(c)) + ")"
	}
}

// allCapabilities returns every defined capability in declaration order. It is the
// single source of truth for "every capability" used by GrantAll and exhaustive
// tests; adding a capability means adding it here and to the enum.
func allCapabilities() []Capability {
	return []Capability{Stdout, Stdin, FileRead, FileWrite, Net, Concurrency, Time, Env}
}

// CapabilitySet is the set of capabilities currently granted. The zero value holds
// none. It is a value type (a bitset) and is safe to copy.
type CapabilitySet struct {
	bits uint64
}

// Has reports whether the set holds capability c.
func (s CapabilitySet) Has(c Capability) bool {
	return s.bits&(1<<uint(c)) != 0
}

// Grant adds capability c to the set.
func (s *CapabilitySet) Grant(c Capability) {
	s.bits |= 1 << uint(c)
}

// GrantAll returns a set holding every defined capability. This is the default
// authority goscript runs with in v1.
func GrantAll() CapabilitySet {
	var s CapabilitySet
	for _, c := range allCapabilities() {
		s.Grant(c)
	}
	return s
}

// DenyAll returns a set holding no capabilities.
func DenyAll() CapabilitySet {
	return CapabilitySet{}
}

// itoa renders a non-negative int without importing strconv (keeps the package
// dependency-free for the unknown-capability String fallback).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

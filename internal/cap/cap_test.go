package cap

import "testing"

func TestGrantAllHasEvery(t *testing.T) {
	s := GrantAll()
	for _, c := range allCapabilities() {
		if !s.Has(c) {
			t.Errorf("GrantAll().Has(%v) = false, want true", c)
		}
	}
}

func TestDenyAllHasNone(t *testing.T) {
	s := DenyAll()
	for _, c := range allCapabilities() {
		if s.Has(c) {
			t.Errorf("DenyAll().Has(%v) = true, want false", c)
		}
	}
}

func TestGrantAddsCapability(t *testing.T) {
	s := DenyAll()
	s.Grant(Net)
	if !s.Has(Net) {
		t.Errorf("after Grant(Net), Has(Net) = false, want true")
	}
	if s.Has(Stdout) {
		t.Errorf("after Grant(Net) only, Has(Stdout) = true, want false")
	}
}

func TestStringNonEmpty(t *testing.T) {
	for _, c := range allCapabilities() {
		got := c.String()
		if got == "" {
			t.Errorf("Capability(%d).String() is empty", int(c))
		}
		if len(got) >= len("Capability(") && got[:len("Capability(")] == "Capability(" {
			t.Errorf("Capability(%d).String() = %q, want a named form (String switch out of sync with enum)", int(c), got)
		}
	}
}

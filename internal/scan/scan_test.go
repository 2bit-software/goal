package scan

import "testing"

func TestCalleeKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"os.MkdirAll(p)", "os.MkdirAll"},
		{"doThing(x)", "doThing"},
		{"f.Close()", "f.Close"},
		{"f[T](x)", "f"},
		{"pkg.Func[T](x)", "pkg.Func"},
		{"  os.Mkdir(p)", "os.Mkdir"},
		{"(a + b)", ""},
		{"", ""},
		// A chained call's outermost call is a method — not resolvable to the head of the
		// chain, so the key is "" (the `?` lowering keeps the safe two-value form).
		{"exec.Command(bin, \"capture-pane\", \"-t\", s).Output()", ""},
		{"f(x).M()", ""},
		{"a.b(x).c()", ""},
		{"pkg.Sub.Func(x)", ""}, // pkg.Sub is a value, .Func a method on it — unresolvable
	}
	for _, c := range cases {
		if got := CalleeKey(c.in); got != c.want {
			t.Errorf("CalleeKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

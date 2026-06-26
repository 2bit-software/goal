package scan

import "testing"

func TestCalleeKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"os.MkdirAll(p)", "os.MkdirAll"},
		{"doThing(x)", "doThing"},
		{"f.Close()", "f.Close"},
		{"f[T](x)", "f"},
		{"pkg.Sub.Func(x)", "pkg.Sub"},
		{"  os.Mkdir(p)", "os.Mkdir"},
		{"(a + b)", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := CalleeKey(c.in); got != c.want {
			t.Errorf("CalleeKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

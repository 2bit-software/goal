package analyze

import "testing"

// TestStructTagsDoNotBreakFieldParse guards against a struct tag being mistaken for the
// field type: a back-quoted (or double-quoted) tag must be stripped so the field name and
// type read correctly. Regression: `Name string ` + "`toml:\"name\"`" parsed as two
// fields named `Name` and `string`, breaking no-zero-value field matching.
func TestStructTagsDoNotBreakFieldParse(t *testing.T) {
	cases := []struct {
		name string
		decl string
	}{
		{
			name: "backtick tag",
			decl: "package p\n\ntype Tagged struct {\n\tName string `toml:\"name\"`\n\tAge  int    `toml:\"age\"`\n}\n",
		},
		{
			name: "double-quoted tag",
			decl: "package p\n\ntype Tagged struct {\n\tName string \"toml:\\\"name\\\"\"\n\tAge  int    \"toml:\\\"age\\\"\"\n}\n",
		},
		{
			name: "tag with trailing comment",
			decl: "package p\n\ntype Tagged struct {\n\tName string `json:\"name\"` // the name\n\tAge  int\n}\n",
		},
	}
	want := []Field{{Name: "Name", Type: "string"}, {Name: "Age", Type: "int"}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Build(c.decl).Structs["Tagged"]
			if len(got) != len(want) {
				t.Fatalf("got %d fields %+v, want %d %+v", len(got), got, len(want), want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("field %d: got %+v, want %+v", i, got[i], want[i])
				}
			}
		})
	}
}

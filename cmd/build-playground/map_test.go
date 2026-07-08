package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"goal/internal/byexample"
)

// playgroundMap mirrors site/playground-map.json — the presentation layer for the
// playground redesign. It carries only presentation (grouping, accent, display
// number, and per-idea `examples`); all content comes from features.json. Each
// example links a `valid` anchor to an optional `error` anchor.
type playgroundMap struct {
	Categories []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Accent   string `json:"accent"`
		Features []struct {
			Anchor   string `json:"anchor"`
			Num      string `json:"num"`
			Examples []struct {
				Label string `json:"label"`
				Valid string `json:"valid"`
				Error string `json:"error"`
			} `json:"examples"`
		} `json:"features"`
	} `json:"categories"`
}

// docPath / mapPath resolve the repo's doc and map from the package dir
// (cmd/build-playground) up to the module root.
const (
	docPath = "../../docs/by-example.md"
	mapPath = "../../site/playground-map.json"
)

// TestPlaygroundMapMatchesDoc is the drift gate the user asked for: the redesign's
// presentation map (site/playground-map.json) must stay in one-to-one correspondence
// with the by-example doc. Each idea is one doc primary; each of its examples links a
// valid primary to an optional error variant. If a feature is added, removed, renamed,
// re-categorized, or gains a new error in docs/by-example.md, this fails until the map
// links it — so a new example in the doc forces a matching example in the redesign, and
// the page can never silently fall out of sync with the language it documents.
func TestPlaygroundMapMatchesDoc(t *testing.T) {
	doc := parseDoc(t)
	pm := loadMap(t)

	// The doc's authoritative view: primaries (a feature you edit) vs. error
	// variants (their valid<->error toggle targets), each with its category.
	docPrimaryCat := map[string]string{}
	docErrors := map[string]bool{}
	for _, cat := range doc.Categories {
		for _, f := range cat.Features {
			if f.OutputKind == "error" || f.OutputKind == "doctest-failure" {
				docErrors[f.Anchor] = true
				continue
			}
			docPrimaryCat[f.Anchor] = cat.Name
		}
	}

	// The map's view: one idea per anchor, its category, every example's valid/error.
	ideaCat := map[string]string{}
	validRefs := map[string]bool{} // primaries linked as some example's valid
	errorOwner := map[string]string{}
	catNames := map[string]bool{}
	for _, cat := range pm.Categories {
		catNames[cat.Name] = true
		for _, f := range cat.Features {
			if prev, dup := ideaCat[f.Anchor]; dup {
				t.Errorf("map lists idea %q twice (in %q and %q)", f.Anchor, prev, cat.Name)
			}
			ideaCat[f.Anchor] = cat.Name
			if len(f.Examples) == 0 {
				t.Errorf("idea %q has no examples — every idea needs at least one", f.Anchor)
			}
			for _, ex := range f.Examples {
				if ex.Valid == "" {
					t.Errorf("idea %q has an example with no `valid` anchor", f.Anchor)
					continue
				}
				validRefs[ex.Valid] = true
				if ex.Error == "" {
					continue
				}
				if owner, dup := errorOwner[ex.Error]; dup {
					t.Errorf("error %q is linked by two examples (%q and %q)", ex.Error, owner, f.Anchor)
				}
				errorOwner[ex.Error] = f.Anchor
			}
		}
	}

	// Ideas ↔ doc primaries is a bijection (every feature gets exactly one nav entry).
	assertSameSet(t, "idea/primary features", keys(docPrimaryCat), keys(ideaCat),
		"add it as an idea", "the doc no longer has it — remove the idea")
	// Every primary is reachable as some example's valid (nothing shown-less).
	assertSubset(t, "primaries not linked as any example `valid`", keys(docPrimaryCat), validRefs)
	// Every valid anchor the map references is a real doc primary.
	assertSubset(t, "example `valid` anchors not present in the doc", keysOfBool(validRefs), toSet(keys(docPrimaryCat)))
	// Errors ↔ example.error is a bijection (every rejection shown exactly once).
	assertSameSet(t, "error variants", keysOfBool(docErrors), keys(errorOwner),
		"link it as some example's `error`", "the doc no longer has it — remove the link")

	// A feature must sit in the same category in the map as in the doc.
	for anchor, docC := range docPrimaryCat {
		if mapC, ok := ideaCat[anchor]; ok && mapC != docC {
			t.Errorf("feature %q is under %q in the doc but %q in the map", anchor, docC, mapC)
		}
	}
	for name := range catNames {
		if !docHasCategory(doc, name) {
			t.Errorf("map category %q does not exist in the doc", name)
		}
	}
}

func parseDoc(t *testing.T) byexample.Doc {
	t.Helper()
	raw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	doc, err := byexample.Parse(string(raw), filepath.Base(docPath))
	if err != nil {
		t.Fatalf("parse doc: %v", err)
	}
	return doc
}

func loadMap(t *testing.T) playgroundMap {
	t.Helper()
	raw, err := os.ReadFile(mapPath)
	if err != nil {
		t.Fatalf("read map: %v", err)
	}
	var pm playgroundMap
	if err := json.Unmarshal(raw, &pm); err != nil {
		t.Fatalf("parse map: %v", err)
	}
	return pm
}

func docHasCategory(doc byexample.Doc, name string) bool {
	for _, c := range doc.Categories {
		if c.Name == name {
			return true
		}
	}
	return false
}

// assertSameSet fails with the precise anchors that are missing from / extra in the
// map, telling the maintainer exactly how to reconcile site/playground-map.json.
func assertSameSet(t *testing.T, label string, doc, mapped []string, addHint, removeHint string) {
	t.Helper()
	docSet := toSet(doc)
	mapSet := toSet(mapped)
	var inDocNotMap, inMapNotDoc []string
	for a := range docSet {
		if !mapSet[a] {
			inDocNotMap = append(inDocNotMap, a)
		}
	}
	for a := range mapSet {
		if !docSet[a] {
			inMapNotDoc = append(inMapNotDoc, a)
		}
	}
	sort.Strings(inDocNotMap)
	sort.Strings(inMapNotDoc)
	if len(inDocNotMap) > 0 {
		t.Errorf("%s in docs/by-example.md but missing from site/playground-map.json: %s\n  %s",
			label, strings.Join(inDocNotMap, ", "), addHint)
	}
	if len(inMapNotDoc) > 0 {
		t.Errorf("%s in site/playground-map.json but not in docs/by-example.md: %s\n  %s",
			label, strings.Join(inMapNotDoc, ", "), removeHint)
	}
}

// assertSubset fails listing each element of want missing from have.
func assertSubset(t *testing.T, label string, want []string, have map[string]bool) {
	t.Helper()
	var missing []string
	for _, w := range want {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("%s: %s", label, strings.Join(missing, ", "))
	}
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func keysOfBool(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

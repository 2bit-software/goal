// Package byexample parses docs/by-example.md into structured per-feature records.
//
// The by-example doc is the project's single source of truth for "here is a goal
// feature, here is a runnable example, here is the Go it lowers to." Each feature
// section carries prose, exactly one name-tagged ```goal``` example, and a locked
// output block (a "Transpiles to" ```go``` block, a "Rejected with" ```error```
// block, or a doctest-failure ```testfail``` block). This package turns that
// document into records every consumer can render
// its own way — the playground renders HTML, the AI guide renders Markdown — so the
// parsing lives in one place instead of being duplicated per consumer.
//
// Records carry raw Markdown lines, not rendered output, so each consumer controls
// its own rendering. Verifying an example against the live transpiler is the
// consumer's job (it requires the pipeline); this package only reads structure.
package byexample

import (
	"fmt"
	"regexp"
	"strings"
)

// Feature is one parsed by-example entry: its prose, the runnable goal example, and
// the output block locked in the doc. DescriptionMD and LoweringMD are raw Markdown
// lines (before/after the example) for the consumer to render.
type Feature struct {
	Anchor         string   // GitHub-style slug of the section heading, e.g. "01-enums"
	Title          string   // heading with any leading "NN. " number stripped
	Category       string   // the enclosing category divider (or, for a level-1 feature, its own prefix)
	DescriptionMD  []string // doc lines before the example, including any un-named syntax sketch
	Source         string   // the .goal example source
	SourceName     string   // the example's name= attribute, e.g. "traffic.goal"
	LoweringMD     []string // doc lines after the output block (the "lowers to" prose), or nil
	OutputKind     string   // "go" | "test" | "error" | "doctest-failure"
	LockedExpected string   // the doc's locked output block, verbatim
}

// Category groups consecutive features under one divider heading.
type Category struct {
	Name     string
	Features []Feature
}

// Doc is a parsed by-example document: its title, the path it came from, and its
// features grouped by category in document order.
type Doc struct {
	Title         string
	GeneratedFrom string
	Categories    []Category
}

var (
	headingRe    = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	goalFenceRe  = regexp.MustCompile("^```goal(?:\\s+(.*))?$")
	goFenceRe       = regexp.MustCompile("^```go\\b")
	errorFenceRe    = regexp.MustCompile("^```error\\b")
	testFailFenceRe = regexp.MustCompile("^```testfail\\b")
	anyFenceRe      = regexp.MustCompile("^```")
	nameAttrRe   = regexp.MustCompile(`name=(\S+)`)
)

// Parse reads the by-example document and returns its features grouped by category.
// A section that contains a name-tagged goal block is a feature; a bare level-1
// heading with no example is a category divider. Document order guarantees a
// category heading is seen before the features it contains.
func Parse(doc, docPath string) (Doc, error) {
	out := Doc{Title: "goal by Example", GeneratedFrom: docPath}
	sections := splitSections(doc)

	category := ""
	for _, sec := range sections {
		if sec.title == "goal by Example" || sec.title == "Contents" {
			continue
		}
		if !hasNamedGoalBlock(sec.body) {
			if sec.level == 1 {
				category = sec.title // a category divider
			}
			continue // headings without an example aren't features (e.g. "Maintaining this doc")
		}
		feat, err := parseFeature(sec)
		if err != nil {
			return Doc{}, fmt.Errorf("feature %q: %w", sec.title, err)
		}
		cat := category
		if sec.level == 1 {
			// A level-1 heading that is itself a feature (the composition section)
			// supplies its own category from the text before the first colon.
			cat = strings.SplitN(sec.title, ":", 2)[0]
		}
		feat.Category = cat
		addFeature(&out, cat, feat)
	}
	if len(out.Categories) == 0 {
		return Doc{}, fmt.Errorf("no features parsed from %s", docPath)
	}
	return out, nil
}

func addFeature(d *Doc, catName string, f Feature) {
	for i := range d.Categories {
		if d.Categories[i].Name == catName {
			d.Categories[i].Features = append(d.Categories[i].Features, f)
			return
		}
	}
	d.Categories = append(d.Categories, Category{Name: catName, Features: []Feature{f}})
}

type section struct {
	level int
	title string
	body  []string // lines after the heading, up to the next heading
}

func splitSections(doc string) []section {
	var out []section
	var cur *section
	for line := range strings.SplitSeq(doc, "\n") {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			out = append(out, section{level: len(m[1]), title: strings.TrimSpace(m[2])})
			cur = &out[len(out)-1]
			continue
		}
		if cur != nil {
			cur.body = append(cur.body, line)
		}
	}
	return out
}

func hasNamedGoalBlock(body []string) bool {
	inFence := false
	for _, line := range body {
		if m := goalFenceRe.FindStringSubmatch(line); m != nil && !inFence {
			if nameAttrRe.MatchString(m[1]) {
				return true
			}
		}
		if anyFenceRe.MatchString(line) {
			inFence = !inFence
		}
	}
	return false
}

// parseFeature splits a feature section into its description (everything up to the
// named goal block, including the un-named syntax sketch), the runnable goal
// example, and the locked output block. It does not verify the output against the
// transpiler — that is the consumer's responsibility.
func parseFeature(sec section) (Feature, error) {
	descLines, exampleStart := collectDescription(sec.body)
	if exampleStart < 0 {
		return Feature{}, fmt.Errorf("no named goal block found")
	}
	source, sourceName, afterSource, err := readGoalExample(sec.body, exampleStart)
	if err != nil {
		return Feature{}, err
	}
	expected, blockKind, afterGo, err := readOutputBlock(sec.body, afterSource)
	if err != nil {
		return Feature{}, err
	}

	// The output block is a "Transpiles to" go block, a "Rejected with" error block
	// (a feature whose example is a located compile error), or a doctest-failure block
	// (a feature whose example is a failing doctest, showing its test failure). A go
	// block whose example carries doctests yields a _test.go instead.
	outputKind := "go"
	switch {
	case blockKind == "error":
		outputKind = "error"
	case blockKind == "testfail":
		outputKind = "doctest-failure"
	case strings.Contains(source, "/// >>>"):
		outputKind = "test"
	}

	var loweringMD []string
	if afterGo >= 0 && afterGo < len(sec.body) {
		loweringMD = sec.body[afterGo:]
	}

	return Feature{
		Anchor:         slugify(sec.title),
		Title:          strings.TrimSpace(stripLeadingNumber(sec.title)),
		DescriptionMD:  descLines,
		Source:         source,
		SourceName:     sourceName,
		LoweringMD:     loweringMD,
		OutputKind:     outputKind,
		LockedExpected: expected,
	}, nil
}

// collectDescription returns the lines before the runnable example (the feature's
// prose, including any un-named syntax-sketch code block) and the index of the line
// that opens the named goal block.
func collectDescription(body []string) (desc []string, exampleStart int) {
	inFence := false
	for i, line := range body {
		if !inFence {
			if m := goalFenceRe.FindStringSubmatch(line); m != nil && nameAttrRe.MatchString(m[1]) {
				return desc, i
			}
		}
		if anyFenceRe.MatchString(line) {
			inFence = !inFence
		}
		desc = append(desc, line)
	}
	return desc, -1
}

func readGoalExample(body []string, start int) (source, name string, next int, err error) {
	m := goalFenceRe.FindStringSubmatch(body[start])
	if am := nameAttrRe.FindStringSubmatch(m[1]); am != nil {
		name = am[1]
	}
	var code []string
	for i := start + 1; i < len(body); i++ {
		if anyFenceRe.MatchString(body[i]) {
			return strings.Join(code, "\n"), name, i + 1, nil
		}
		code = append(code, body[i])
	}
	return "", "", -1, fmt.Errorf("unterminated goal block")
}

// readOutputBlock reads the feature's locked output: the first fenced block at or
// after from that is a "Transpiles to" go block (kind "go"), a "Rejected with" error
// block (kind "error", holding the exact located compile error), or a doctest-failure
// block (kind "testfail", holding a failing doctest's locked failure output).
func readOutputBlock(body []string, from int) (expected, kind string, next int, err error) {
	for i := from; i < len(body); i++ {
		switch {
		case goFenceRe.MatchString(body[i]):
			exp, n, e := readFenceBody(body, i)
			return exp, "go", n, e
		case errorFenceRe.MatchString(body[i]):
			exp, n, e := readFenceBody(body, i)
			return exp, "error", n, e
		case testFailFenceRe.MatchString(body[i]):
			exp, n, e := readFenceBody(body, i)
			return exp, "testfail", n, e
		}
	}
	return "", "", -1, fmt.Errorf("no \"Transpiles to\" go block, \"Rejected with\" error block, or doctest-failure block found")
}

// readFenceBody returns the lines inside the fenced block opening at body[openIdx]
// and the index just past its closing fence.
func readFenceBody(body []string, openIdx int) (string, int, error) {
	var code []string
	for j := openIdx + 1; j < len(body); j++ {
		if anyFenceRe.MatchString(body[j]) {
			return strings.Join(code, "\n"), j + 1, nil
		}
		code = append(code, body[j])
	}
	return "", -1, fmt.Errorf("unterminated output block")
}

var (
	leadingNumberRe = regexp.MustCompile(`^\d+\.\s+`)
	nonSlugRe       = regexp.MustCompile(`[^a-z0-9]+`)
)

func stripLeadingNumber(s string) string {
	return leadingNumberRe.ReplaceAllString(s, "")
}

// slugify mirrors the GitHub-style anchor used in the doc's table of contents.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "`", "")
	s = nonSlugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

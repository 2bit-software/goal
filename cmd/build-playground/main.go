// Command build-playground generates the playground manifest (site/features.json)
// from docs/by-example.md.
//
// The doc is the single source of truth: every feature section there
// (## heading + prose + a ```goal name=…``` example + its "Transpiles to" Go
// block) becomes one playground feature. Because this tool imports the real
// pipeline, it RE-TRANSPILES each example at build time and asserts the result
// matches the Go block locked in the doc — so the manifest (and the doc) cannot
// drift from the transpiler, and the seed output the page shows is guaranteed to
// equal what the WASM build produces live.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"goal/internal/pipeline"
)

func main() {
	in := flag.String("in", "docs/by-example.md", "path to the by-example markdown")
	out := flag.String("out", "site/features.json", "path to write the generated manifest")
	flag.Parse()

	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "build-playground:", err)
		os.Exit(1)
	}
}

// run parses the by-example doc at inPath, verifies every example against the
// live transpiler, and writes the manifest JSON to outPath.
func run(inPath, outPath string) error {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read doc: %w", err)
	}
	manifest, err := parse(string(raw), inPath)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	fmt.Printf("wrote %s: %d features across %d categories\n",
		outPath, manifest.featureCount(), len(manifest.Categories))
	return nil
}

// Manifest is the top-level shape of features.json, consumed by site/app.js.
type Manifest struct {
	Title         string     `json:"title"`
	GeneratedFrom string     `json:"generatedFrom"`
	Categories    []Category `json:"categories"`
}

func (m Manifest) featureCount() int {
	n := 0
	for _, c := range m.Categories {
		n += len(c.Features)
	}
	return n
}

// Category groups related features in the sidebar.
type Category struct {
	Name     string    `json:"name"`
	Features []Feature `json:"features"`
}

// Feature is one playground entry: a goal source example and the Go it lowers to.
type Feature struct {
	Anchor          string `json:"anchor"`
	Title           string `json:"title"`
	TitleHTML       string `json:"titleHtml"`
	DescriptionHTML string `json:"descriptionHtml"`
	LoweringHTML    string `json:"loweringHtml,omitempty"`
	Source          string `json:"source"`         // the .goal example
	SourceName      string `json:"sourceName"`     // e.g. "traffic.goal"
	OutputKind      string `json:"outputKind"`     // "go" | "test"
	Expected        string `json:"expected"`       // the locked, verified output
	ExpectedLabel   string `json:"expectedLabel"`  // pane label, e.g. "transpiled Go"
}

var (
	headingRe   = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	goalFenceRe = regexp.MustCompile("^```goal(?:\\s+(.*))?$")
	goFenceRe   = regexp.MustCompile("^```go\\b")
	anyFenceRe  = regexp.MustCompile("^```")
	nameAttrRe  = regexp.MustCompile(`name=(\S+)`)
)

// parse walks the doc as a sequence of heading-delimited sections. A section that
// contains a `name=`-tagged goal block is a feature; a bare level-1 heading is a
// category. Document order guarantees a category heading is seen before the
// features it contains.
func parse(doc, docPath string) (Manifest, error) {
	manifest := Manifest{Title: "goal by Example", GeneratedFrom: docPath}
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
			return Manifest{}, fmt.Errorf("feature %q: %w", sec.title, err)
		}
		cat := category
		if sec.level == 1 {
			// A level-1 heading that is itself a feature (the composition section).
			cat = strings.SplitN(sec.title, ":", 2)[0]
		}
		// Drop a redundant "Category: " prefix from the display title.
		display := strings.TrimPrefix(feat.Title, cat+": ")
		feat.Title = display
		feat.TitleHTML = renderInline(display)
		addFeature(&manifest, cat, feat)
	}
	if len(manifest.Categories) == 0 {
		return Manifest{}, fmt.Errorf("no features parsed from %s", docPath)
	}
	return manifest, nil
}

func addFeature(m *Manifest, catName string, f Feature) {
	for i := range m.Categories {
		if m.Categories[i].Name == catName {
			m.Categories[i].Features = append(m.Categories[i].Features, f)
			return
		}
	}
	m.Categories = append(m.Categories, Category{Name: catName, Features: []Feature{f}})
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
// example, and the locked Go output — then verifies the output against the live
// transpiler.
func parseFeature(sec section) (Feature, error) {
	descLines, exampleStart := collectDescription(sec.body)
	if exampleStart < 0 {
		return Feature{}, fmt.Errorf("no named goal block found")
	}
	source, sourceName, afterSource, err := readGoalExample(sec.body, exampleStart)
	if err != nil {
		return Feature{}, err
	}
	expected, afterGo, err := readGoBlock(sec.body, afterSource)
	if err != nil {
		return Feature{}, err
	}

	outputKind := "go"
	expectedLabel := "transpiled Go"
	if strings.Contains(source, "/// >>>") {
		outputKind = "test"
		expectedLabel = "generated _test.go"
	}

	if err := verify(source, expected, outputKind); err != nil {
		return Feature{}, err
	}

	loweringHTML := ""
	if afterGo >= 0 && afterGo < len(sec.body) {
		loweringHTML = renderMarkdown(sec.body[afterGo:])
	}

	return Feature{
		Anchor:          slugify(sec.title),
		Title:           strings.TrimSpace(stripLeadingNumber(sec.title)),
		DescriptionHTML: renderMarkdown(descLines),
		LoweringHTML:    loweringHTML,
		Source:          source,
		SourceName:      sourceName,
		OutputKind:      outputKind,
		Expected:        expected,
		ExpectedLabel:   expectedLabel,
	}, nil
}

// collectDescription returns the lines before the runnable example (rendered as
// the feature's prose, including any un-named syntax-sketch code block) and the
// index of the line that opens the named goal block.
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

func readGoBlock(body []string, from int) (expected string, next int, err error) {
	for i := from; i < len(body); i++ {
		if goFenceRe.MatchString(body[i]) {
			var code []string
			for j := i + 1; j < len(body); j++ {
				if anyFenceRe.MatchString(body[j]) {
					return strings.Join(code, "\n"), j + 1, nil
				}
				code = append(code, body[j])
			}
			return "", -1, fmt.Errorf("unterminated go block")
		}
	}
	return "", -1, fmt.Errorf("no \"Transpiles to\" go block found")
}

// verify re-transpiles source and asserts the result matches the doc's locked
// block, so the manifest cannot drift from the transpiler.
func verify(source, expected, kind string) error {
	res, err := pipeline.Transpile(source)
	if err != nil {
		return fmt.Errorf("live transpile failed: %w", err)
	}
	got := res.Go
	if kind == "test" {
		got = res.Test
	}
	if strings.TrimRight(got, "\n") != strings.TrimRight(expected, "\n") {
		return fmt.Errorf("doc output does not match live transpiler\n--- doc ---\n%s\n--- live ---\n%s",
			expected, got)
	}
	return nil
}

// --------------------------------------------------------------------------- //
// minimal markdown rendering (the controlled subset the doc uses)
// --------------------------------------------------------------------------- //

// renderMarkdown turns a block of doc lines into HTML, handling paragraphs,
// fenced code blocks, and inline code / bold / links.
func renderMarkdown(lines []string) string {
	var b strings.Builder
	var para []string
	flush := func() {
		if len(para) == 0 {
			return
		}
		b.WriteString("<p>")
		b.WriteString(renderInline(strings.Join(para, " ")))
		b.WriteString("</p>\n")
		para = nil
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if m := anyFenceRe.FindString(line); m != "" {
			flush()
			lang := strings.TrimPrefix(strings.TrimSpace(line), "```")
			lang = strings.Fields(lang + " ")[0]
			var code []string
			for i++; i < len(lines); i++ {
				if anyFenceRe.MatchString(lines[i]) {
					break
				}
				code = append(code, lines[i])
			}
			fmt.Fprintf(&b, "<pre class=\"code lang-%s\"><code>%s</code></pre>\n",
				escapeHTML(lang), escapeHTML(strings.Join(code, "\n")))
			continue
		}
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		para = append(para, strings.TrimSpace(line))
	}
	flush()
	return strings.TrimSpace(b.String())
}

var (
	codeSpanRe = regexp.MustCompile("`([^`]+)`")
	boldRe     = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe   = regexp.MustCompile(`\*([^*]+)\*`)
	linkRe     = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

// renderInline applies the inline markdown the doc uses. Code spans are extracted
// first (and protected from further substitution) so their contents are never
// treated as bold or link syntax.
func renderInline(s string) string {
	var spans []string
	s = codeSpanRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := m[1 : len(m)-1]
		spans = append(spans, "<code>"+escapeHTML(inner)+"</code>")
		return fmt.Sprintf("\x00%d\x00", len(spans)-1)
	})
	s = escapeHTML(s)
	s = boldRe.ReplaceAllString(s, "<strong>$1</strong>")
	s = italicRe.ReplaceAllString(s, "<em>$1</em>")
	s = linkRe.ReplaceAllString(s, `<a href="$2">$1</a>`)
	for i, span := range spans {
		s = strings.ReplaceAll(s, fmt.Sprintf("\x00%d\x00", i), span)
	}
	return s
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
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

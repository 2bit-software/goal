// Command build-playground generates the playground manifest (site/features.json)
// from docs/by-example.md.
//
// The doc is the single source of truth: every feature section there
// (## heading + prose + a ```goal name=…``` example + its "Transpiles to" Go
// block) becomes one playground feature. Parsing is shared with the rest of the
// toolchain via internal/byexample; this command renders the parsed records to
// HTML and, because it imports the real pipeline, RE-TRANSPILES each example at
// build time and asserts the result matches the Go block locked in the doc — so
// the manifest (and the doc) cannot drift from the transpiler, and the seed output
// the page shows is guaranteed to equal what the WASM build produces live.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"goal/internal/backend"
	"goal/internal/byexample"
	"goal/internal/sema"
)

func main() {
	in := flag.String("in", "docs/by-example.md", "path to the by-example markdown")
	overview := flag.String("overview", "docs/overview.md", "path to the landing-page markdown")
	out := flag.String("out", "site/features.json", "path to write the generated manifest")
	flag.Parse()

	if err := run(*in, *overview, *out); err != nil {
		fmt.Fprintln(os.Stderr, "build-playground:", err)
		os.Exit(1)
	}
}

// run parses the by-example doc at inPath, verifies every example against the
// live transpiler, renders the landing page from overviewPath, and writes the
// manifest JSON to outPath.
func run(inPath, overviewPath, outPath string) error {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read doc: %w", err)
	}
	manifest, err := buildManifest(string(raw), inPath)
	if err != nil {
		return err
	}
	if intro, err := os.ReadFile(overviewPath); err == nil {
		manifest.IntroHTML = renderMarkdown(strings.Split(string(intro), "\n"))
	} else {
		return fmt.Errorf("read overview: %w", err)
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
	IntroHTML     string     `json:"introHtml"`
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
	Source          string `json:"source"`        // the .goal example
	SourceName      string `json:"sourceName"`    // e.g. "traffic.goal"
	OutputKind      string `json:"outputKind"`    // "go" | "test"
	Expected        string `json:"expected"`      // the locked, verified output
	ExpectedLabel   string `json:"expectedLabel"` // pane label, e.g. "transpiled Go"
}

// buildManifest parses the doc with the shared parser, then renders each feature to
// HTML and verifies its locked output against the live transpiler.
func buildManifest(doc, docPath string) (Manifest, error) {
	parsed, err := byexample.Parse(doc, docPath)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{Title: parsed.Title, GeneratedFrom: parsed.GeneratedFrom}
	for _, cat := range parsed.Categories {
		for _, f := range cat.Features {
			if err := verify(f.Source, f.LockedExpected, f.OutputKind, f.SourceName); err != nil {
				return Manifest{}, fmt.Errorf("feature %q: %w", f.Title, err)
			}
			// The doc keeps keyword headings lowercase (implements, assert) and
			// numbers feature headings; the playground display drops a redundant
			// "Category: " prefix and sentence-cases the first letter so nav reads
			// uniformly. Titles that lead with a code span or symbol are left as-is.
			display := capitalizeFirst(strings.TrimPrefix(f.Title, cat.Name+": "))
			addFeature(&manifest, cat.Name, Feature{
				Anchor:          f.Anchor,
				Title:           display,
				TitleHTML:       renderInline(display),
				DescriptionHTML: renderMarkdown(f.DescriptionMD),
				LoweringHTML:    renderMarkdown(f.LoweringMD),
				Source:          f.Source,
				SourceName:      f.SourceName,
				OutputKind:      f.OutputKind,
				Expected:        f.LockedExpected,
				ExpectedLabel:   expectedLabel(f.OutputKind),
			})
		}
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

// expectedLabel is the output-pane label for a feature's output kind.
func expectedLabel(kind string) string {
	switch kind {
	case "error":
		return "rejected"
	case "test":
		return "generated _test.go"
	default:
		return "transpiled Go"
	}
}

// verify re-runs source through the live toolchain and asserts the result matches
// the doc's locked block, so the manifest cannot drift. For an error feature it
// delegates to verifyError; go/test features are re-transpiled and compared.
func verify(source, expected, kind, sourceName string) error {
	if kind == "error" {
		return verifyError(source, expected, sourceName)
	}
	res, err := backend.Transpile(source)
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

// verifyError verifies an error feature's locked block against the live toolchain.
//
// When the locked block is a located checker diagnostic (its first line begins
// with the feature's SourceName, e.g. "traffic.goal:9:2: error: [code] …"), it is
// verified against the checker: sema.Analyze must report an Error-severity
// diagnostic whose Render(sourceName) equals the locked block exactly (trailing
// newline trimmed). This is the same path goalc's checker uses.
//
// Otherwise the block is a backend transpile rejection (e.g. the "backend: …"
// unsafe-default example) and is verified against backend.Transpile, unchanged.
func verifyError(source, expected, sourceName string) error {
	if isCheckerDiagnostic(expected, sourceName) {
		diags, err := sema.Analyze(source)
		if err != nil {
			return fmt.Errorf("checker failed to analyze source: %w", err)
		}
		d, ok := firstError(diags)
		if !ok {
			return fmt.Errorf("expected a checker error diagnostic, but the checker reported none")
		}
		got := d.Render(sourceName)
		if strings.TrimRight(got, "\n") != strings.TrimRight(expected, "\n") {
			return fmt.Errorf("doc error does not match live checker\n--- doc ---\n%s\n--- live ---\n%s",
				expected, got)
		}
		return nil
	}
	res, err := backend.Transpile(source)
	if err == nil {
		return fmt.Errorf("expected transpile to be rejected, but it succeeded:\n%s", res.Go)
	}
	if strings.TrimRight(err.Error(), "\n") != strings.TrimRight(expected, "\n") {
		return fmt.Errorf("doc error does not match live transpiler\n--- doc ---\n%s\n--- live ---\n%s",
			expected, err.Error())
	}
	return nil
}

// isCheckerDiagnostic reports whether a locked error block is a located checker
// diagnostic — its first non-empty line begins with the feature's SourceName
// followed by ":", the form Diagnostic.Render produces. A backend rejection block
// (e.g. "backend: …") does not, so it routes to the backend fallback.
func isCheckerDiagnostic(expected, sourceName string) bool {
	if sourceName == "" {
		return false
	}
	first := strings.TrimLeft(expected, "\n")
	if i := strings.IndexByte(first, '\n'); i >= 0 {
		first = first[:i]
	}
	return strings.HasPrefix(first, sourceName+":")
}

// firstError returns the first Error-severity diagnostic, if any.
func firstError(diags []sema.Diagnostic) (sema.Diagnostic, bool) {
	for _, d := range diags {
		if sema.HasErrors([]sema.Diagnostic{d}) {
			return d, true
		}
	}
	return sema.Diagnostic{}, false
}

// --------------------------------------------------------------------------- //
// minimal markdown rendering (the controlled subset the doc uses)
// --------------------------------------------------------------------------- //

var (
	headingLineRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	listItemRe    = regexp.MustCompile(`^[-*]\s+(.*)$`)
	anyFenceRe    = regexp.MustCompile("^```")
)

// renderMarkdown turns a block of doc lines into HTML, handling the markdown
// subset the docs use: headings, paragraphs, unordered lists, blockquotes,
// fenced code blocks, and inline code / bold / italic / links.
func renderMarkdown(lines []string) string {
	var b strings.Builder
	var para []string
	flush := func() {
		if len(para) == 0 {
			return
		}
		fmt.Fprintf(&b, "<p>%s</p>\n", renderInline(strings.Join(para, " ")))
		para = nil
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```"):
			flush()
			i = writeFence(&b, lines, i)
		case headingLineRe.MatchString(line):
			flush()
			m := headingLineRe.FindStringSubmatch(line)
			lvl := len(m[1])
			fmt.Fprintf(&b, "<h%d>%s</h%d>\n", lvl, renderInline(strings.TrimSpace(m[2])), lvl)
		case listItemRe.MatchString(trimmed):
			flush()
			i = writeList(&b, lines, i)
		case strings.HasPrefix(trimmed, ">"):
			flush()
			i = writeQuote(&b, lines, i)
		case trimmed == "":
			flush()
		default:
			para = append(para, trimmed)
		}
	}
	flush()
	return strings.TrimSpace(b.String())
}

// writeFence emits a code block starting at lines[start] and returns the index
// of its closing fence.
func writeFence(b *strings.Builder, lines []string, start int) int {
	lang := strings.Fields(strings.TrimPrefix(strings.TrimSpace(lines[start]), "```") + " ")[0]
	var code []string
	i := start + 1
	for ; i < len(lines); i++ {
		if anyFenceRe.MatchString(lines[i]) {
			break
		}
		code = append(code, lines[i])
	}
	fmt.Fprintf(b, "<pre class=\"code lang-%s\"><code>%s</code></pre>\n",
		escapeHTML(lang), escapeHTML(strings.Join(code, "\n")))
	return i
}

// writeList emits a <ul> consuming consecutive list items and returns the index
// of the last item line.
func writeList(b *strings.Builder, lines []string, start int) int {
	b.WriteString("<ul>\n")
	i := start
	for ; i < len(lines); i++ {
		m := listItemRe.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if m == nil {
			break
		}
		fmt.Fprintf(b, "<li>%s</li>\n", renderInline(m[1]))
	}
	b.WriteString("</ul>\n")
	return i - 1
}

// writeQuote emits a <blockquote> consuming consecutive `>` lines and returns the
// index of the last quoted line.
func writeQuote(b *strings.Builder, lines []string, start int) int {
	var quoted []string
	i := start
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(t, ">") {
			break
		}
		quoted = append(quoted, strings.TrimSpace(strings.TrimPrefix(t, ">")))
	}
	fmt.Fprintf(b, "<blockquote><p>%s</p></blockquote>\n", renderInline(strings.Join(quoted, " ")))
	return i - 1
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

// capitalizeFirst upper-cases the first character when it is a lowercase ASCII
// letter, so nav titles read consistently. Titles that lead with a symbol or
// code span (e.g. "`?` propagation") are returned unchanged.
func capitalizeFirst(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-('a'-'A')) + s[1:]
}

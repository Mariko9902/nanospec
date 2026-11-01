package specdoc

import (
	"reflect"
	"testing"

	"nanospec/internal/diagnostic"
)

func TestParseDocumentReadsSupportedFrontmatter(t *testing.T) {
	source := "---\r\n" +
		"id: story-01-sample # identity\r\n" +
		"cycle: 'cycle-01-sample'\r\n" +
		"depends_on: [story-00-base, \"story-02,a\"] # dependencies\r\n" +
		"window:\r\n" +
		"  - current\r\n" +
		"empty: []\r\n" +
		"---\r\n" +
		"## Orientation\r\n\r\nResult.\r\n"
	var report diagnostic.Report

	document := ParseDocument("story.md", source, &report)

	assertScalar(t, document.Frontmatter, "id", "story-01-sample")
	assertScalar(t, document.Frontmatter, "cycle", "cycle-01-sample")
	assertList(t, document.Frontmatter, "depends_on", []string{"story-00-base", "story-02,a"})
	assertList(t, document.Frontmatter, "window", []string{"current"})
	assertList(t, document.Frontmatter, "empty", []string{})
	if keys := document.Frontmatter.Keys(); !reflect.DeepEqual(keys, []string{"cycle", "depends_on", "empty", "id", "window"}) {
		t.Fatalf("Keys = %#v", keys)
	}
	if document.Body != "## Orientation\r\n\r\nResult.\r\n" {
		t.Fatalf("Body = %q", document.Body)
	}
	if report.HasErrors() {
		t.Fatalf("report = %#v", report.Issues())
	}
}

func TestParseDocumentReportsMissingFrontmatter(t *testing.T) {
	var report diagnostic.Report
	document := ParseDocument("story.md", "## Orientation\n", &report)

	if document.Body != "## Orientation\n" {
		t.Fatalf("Body = %q", document.Body)
	}
	if len(report.Errors()) != 1 || report.Errors()[0].Message != "missing YAML frontmatter" {
		t.Fatalf("errors = %#v", report.Errors())
	}
}

func TestParseMarkdownBuildsFenceAwareSections(t *testing.T) {
	source := "## Definition\r\n\r\nIntro\r\n\r\n" +
		"~~~markdown\r\n## Hidden\r\n~~~\r\n\r\n" +
		"### Description\r\n\r\nDetails\r\n\r\n" +
		"## Acceptance\r\n\r\n- [ ] A1: Result.\r\n"

	markdown := ParseMarkdown(source)

	if _, ok := markdown.Section(2, "Hidden"); ok {
		t.Fatal("fenced heading was indexed")
	}
	definition, ok := markdown.Section(2, "Definition")
	if !ok {
		t.Fatal("Definition section not found")
	}
	wantDefinition := "Intro\r\n\r\n~~~markdown\r\n## Hidden\r\n~~~\r\n\r\n### Description\r\n\r\nDetails"
	if definition.Content != wantDefinition {
		t.Fatalf("Definition content = %q, want %q", definition.Content, wantDefinition)
	}
	description, ok := markdown.Section(3, "Description")
	if !ok || description.Content != "Details" {
		t.Fatalf("Description = %#v, found = %t", description, ok)
	}
	if len(markdown.Sections(2, "Acceptance")) != 1 {
		t.Fatalf("Acceptance sections = %#v", markdown.Sections(2, "Acceptance"))
	}
	if len(markdown.Headings()) != 3 {
		t.Fatalf("headings = %#v", markdown.Headings())
	}
}

func TestMeaningfulAndExplicitNone(t *testing.T) {
	if Meaningful("<!-- hidden -->\n- [ ]") {
		t.Fatal("structural marks are meaningful")
	}
	if !Meaningful("- saved result") {
		t.Fatal("content is not meaningful")
	}
	if !IsExplicitNone("- none") {
		t.Fatal("- none was not recognized")
	}
	if IsExplicitNone("none yet") {
		t.Fatal("ordinary text was recognized as none")
	}
}

func TestFrontmatterValuesReturnCopies(t *testing.T) {
	var report diagnostic.Report
	document := ParseDocument("story.md", "---\ndepends_on: [story-01-base]\n---\n", &report)
	value, _ := document.Frontmatter.Get("depends_on")
	items, _ := value.Items()
	items[0] = "changed"
	actual, _ := value.Items()
	if actual[0] != "story-01-base" {
		t.Fatal("Value.Items exposed mutable storage")
	}
	keys := document.Frontmatter.Keys()
	keys[0] = "changed"
	if document.Frontmatter.Keys()[0] != "depends_on" {
		t.Fatal("Frontmatter.Keys exposed mutable storage")
	}
}

func assertScalar(t *testing.T, frontmatter Frontmatter, key, want string) {
	t.Helper()
	value, ok := frontmatter.Get(key)
	if !ok {
		t.Fatalf("field %q not found", key)
	}
	actual, ok := value.Scalar()
	if !ok || actual != want {
		t.Fatalf("%s = %q, scalar = %t, want %q", key, actual, ok, want)
	}
}

func assertList(t *testing.T, frontmatter Frontmatter, key string, want []string) {
	t.Helper()
	value, ok := frontmatter.Get(key)
	if !ok {
		t.Fatalf("field %q not found", key)
	}
	actual, ok := value.Items()
	if !ok || !reflect.DeepEqual(actual, want) {
		t.Fatalf("%s = %#v, list = %t, want %#v", key, actual, ok, want)
	}
}

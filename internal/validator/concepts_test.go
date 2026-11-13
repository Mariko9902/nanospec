package validator

import (
	"path/filepath"
	"strings"
	"testing"

	"nanospec/internal/diagnostic"
)

func TestValidateStoryConceptsPlacement(t *testing.T) {
	concepts := "### Concepts\n\n- **Reservation**: Set aside the stock needed for this order.\n\n"
	cases := []struct {
		name              string
		before            string
		after             string
		outsideDefinition bool
		wantError         string
		wantWarning       string
	}{
		{name: "omitted"},
		{name: "within definition", before: concepts},
		{name: "empty", before: "### Concepts\n\n", wantWarning: "optional section has no content: Concepts"},
		{name: "explicit none", before: "### Concepts\n\nnone\n\n", wantWarning: "optional section has no content: Concepts"},
		{name: "top level", before: strings.Replace(concepts, "###", "##", 1), wantError: "Concepts must be nested under Definition"},
		{name: "outside definition", outsideDefinition: true, wantError: "Concepts must be nested under Definition"},
		{name: "too deeply nested", before: strings.Replace(concepts, "###", "####", 1), wantError: "Concepts must be a level-3 subsection of Definition"},
		{name: "level five", before: strings.Replace(concepts, "###", "#####", 1), wantError: "Concepts must be a level-3 subsection of Definition"},
		{name: "after description", after: concepts, wantError: "Definition section out of order: Description"},
		{name: "duplicated", before: concepts + concepts, wantError: "Definition has duplicate section: Concepts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := basicStory("story-01-order", "", "none")
			document = strings.Replace(document, "### Description", tc.before+"### Description", 1)
			document = strings.Replace(document, "### Business Logic", tc.after+"### Business Logic", 1)
			if tc.outsideDefinition {
				document = strings.Replace(document, "## Acceptance", "## Out of Scope\n\n"+concepts+"## Acceptance", 1)
			}
			directory := writeStoryFixture(t, "story-01-order", document)
			var report diagnostic.Report
			ValidateStoryDirectory(directory, fixtureCycleID, &report)
			if tc.wantError != "" {
				assertIssue(t, report.Errors(), tc.wantError)
			} else if report.HasErrors() {
				t.Fatalf("errors = %#v", report.Errors())
			}
			if tc.wantWarning != "" {
				assertIssue(t, report.Warnings(), tc.wantWarning)
			} else if len(report.Warnings()) != 0 {
				t.Fatalf("warnings = %#v", report.Warnings())
			}
		})
	}
}

func TestExecutionPackagePreservesConcepts(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	storyFile := filepath.Join(cycleDirectory, "story-01-base", "story.md")
	concepts := "### Concepts\n\n- **Reservation**: Set aside the stock needed for this order.\n\n"
	document := strings.Replace(readTestFile(t, storyFile), "### Description", concepts+"### Description", 1)
	writeFixtureFile(t, storyFile, document)
	stateFile := filepath.Join(cycleDirectory, "cycle.md")
	stateBefore := readTestFile(t, stateFile)

	prepared, result := BuildExecutionPackage(storyFile, "# Rules\n\nUse current facts.")
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(prepared.ContextMarkdown, document) {
		t.Fatal("execution package omitted or changed the Story, including its concept explanation")
	}
	if readTestFile(t, storyFile) != document || readTestFile(t, stateFile) != stateBefore {
		t.Fatal("preparation changed specification or execution state")
	}
}

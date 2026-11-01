package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"nanospec/internal/diagnostic"
)

func TestValidateSpecSynchronizesAndRechecksCycleViews(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base":  {},
		"story-02-order": {"story-01-base"},
	})
	baseFile := filepath.Join(cycleDirectory, "story-01-base", "story.md")
	base := strings.Replace(readTestFile(t, baseFile), "- Title: Save an order", "- Title: Base capability", 1)
	writeFixtureFile(t, baseFile, base)
	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	cycle := strings.Replace(readTestFile(t, cycleFile), "`story-01-base`: Save an order", "`story-01-base`: Old title；Start first", 1)
	writeFixtureFile(t, cycleFile, cycle)

	result := ValidateSpec(cycleDirectory, true)
	if len(result.Errors) != 0 {
		t.Fatalf("sync errors = %#v", result.Errors)
	}
	if len(result.Updates) == 0 {
		t.Fatal("sync produced no updates")
	}
	if err := SaveViewUpdates(result.Updates); err != nil {
		t.Fatal(err)
	}

	cycleText, err := os.ReadFile(filepath.Join(cycleDirectory, "cycle.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(cycleText)
	if !strings.Contains(content, "#### Wave 2") {
		t.Fatalf("cycle does not contain Wave 2:\n%s", content)
	}
	if !strings.Contains(content, "S01 --> S02") {
		t.Fatalf("cycle does not contain dependency edge:\n%s", content)
	}
	if !strings.Contains(content, "`story-01-base`: Base capability; Start first") || !strings.Contains(content, `S01["S01 Base capability"]`) {
		t.Fatalf("cycle does not use Story Title as the generated name:\n%s", content)
	}

	checked := ValidateSpec(cycleDirectory, false)
	if len(checked.Errors) != 0 {
		t.Fatalf("read-only errors = %#v", checked.Errors)
	}
	repeated := ValidateSpec(cycleDirectory, true)
	if len(repeated.Errors) != 0 || len(repeated.Updates) != 0 {
		t.Fatalf("repeated sync = %#v", repeated)
	}
}

func TestValidateSpecRejectsDependencyCycle(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base":  {"story-02-order"},
		"story-02-order": {"story-01-base"},
	})

	result := ValidateSpec(cycleDirectory, true)

	assertDiagnostic(t, result.Errors, "story dependency cycle detected")
	if len(result.Updates) != 0 {
		t.Fatalf("invalid cycle produced updates: %#v", result.Updates)
	}
}

func TestValidateSpecWarnsAboutTransitiveDependency(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base":   {},
		"story-02-order":  {"story-01-base"},
		"story-03-report": {"story-01-base", "story-02-order"},
	})

	result := ValidateSpec(cycleDirectory, true)

	if len(result.Errors) != 0 {
		t.Fatalf("errors = %#v", result.Errors)
	}
	assertDiagnostic(t, result.Warnings, "contains transitive dependency story-01-base")
}

func TestValidateSpecChecksAcceptedStateAgainstOutcome(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	stateFile := filepath.Join(cycleDirectory, "execution-state.md")
	state := readTestFile(t, stateFile)
	state = strings.Replace(state, "- [ ] story-01-base", "- [x] story-01-base", 1)
	state = strings.Replace(state, "Acceptance: pending: not observed", "Acceptance: accepted: confirmed", 1)
	writeFixtureFile(t, stateFile, state)

	result := ValidateSpec(cycleDirectory, true)

	assertDiagnostic(t, result.Errors, "accepted story must contain outcome.md: story-01-base")
}

func TestValidateSpecRejectsUnsupportedProtocolSections(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	stateFile := filepath.Join(cycleDirectory, "execution-state.md")
	cycle := strings.Replace(readTestFile(t, cycleFile), "## Story Plan", "## Extra Section\n\n- Undefined content.\n\n## Story Plan", 1)
	state := strings.Replace(readTestFile(t, stateFile), "## Progress Tree", "## Extra Section\n\n- Undefined content.\n\n## Progress Tree", 1)
	writeFixtureFile(t, cycleFile, cycle)
	writeFixtureFile(t, stateFile, state)

	result := ValidateSpec(cycleDirectory, true)

	assertDiagnostic(t, result.Errors, "Cycle has unsupported section: Extra Section")
	assertDiagnostic(t, result.Errors, "Execution State has unsupported section: Extra Section")
}

func TestSaveViewUpdatesRejectsChangedSource(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cycle.md")
	writeFixtureFile(t, file, "before")
	writeFixtureFile(t, file, "changed")

	err := SaveViewUpdates([]ViewUpdate{{File: file, Before: "before", After: "after"}})

	if err == nil || !strings.Contains(err.Error(), "file changed during validation") {
		t.Fatalf("error = %v", err)
	}
	if actual := readTestFile(t, file); actual != "changed" {
		t.Fatalf("file = %q", actual)
	}
}

func TestValidateSpecRejectsCycleDependencies(t *testing.T) {
	for _, dependencies := range []string{"[]", "[cycle-00-base]"} {
		t.Run(dependencies, func(t *testing.T) {
			cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
			cycleFile := filepath.Join(cycleDirectory, "cycle.md")
			cycle := strings.Replace(readTestFile(t, cycleFile), "window:", "depends_on: "+dependencies+"\nwindow:", 1)
			writeFixtureFile(t, cycleFile, cycle)

			result := ValidateSpec(cycleDirectory, true)

			assertDiagnostic(t, result.Errors, "Cycle frontmatter has unsupported key: depends_on")
		})
	}
}

func writeCycleFixture(t *testing.T, dependencies map[string][]string) string {
	t.Helper()
	root := t.TempDir()
	cycleDirectory := filepath.Join(root, fixtureCycleID)
	if err := os.MkdirAll(cycleDirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	ids := make([]string, 0, len(dependencies))
	for id := range dependencies {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		document := basicStory(id, "", "none")
		list := "[]"
		if len(dependencies[id]) > 0 {
			list = "[" + strings.Join(dependencies[id], ", ") + "]"
		}
		document = strings.Replace(document, "depends_on: []", "depends_on: "+list, 1)
		directory := filepath.Join(cycleDirectory, id)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFixtureFile(t, filepath.Join(directory, "story.md"), document)
	}
	writeFixtureFile(t, filepath.Join(cycleDirectory, "cycle.md"), testCycleDocument(ids))
	writeFixtureFile(t, filepath.Join(cycleDirectory, "execution-state.md"), testStateDocument(ids))
	return cycleDirectory
}

func testCycleDocument(ids []string) string {
	quote := string(rune(96))
	entries := make([]string, 0, len(ids))
	for index, id := range ids {
		entries = append(entries, fmt.Sprintf("%d. %s%s%s: Save an order", index+1, quote, id, quote))
	}
	return "---\n" +
		"id: " + fixtureCycleID + "\n" +
		"window:\n" +
		"  - current\n" +
		"---\n\n" +
		"## Planning Context\n\n- Current delivery scope.\n\n" +
		"## Story Plan\n\n" +
		"### Dependency Waves\n\n" +
		"#### Wave 1\n\n" +
		strings.Join(entries, "\n") + "\n\n" +
		"### Dependency Graph\n\n- none\n\n" +
		"## Risks\n\n- none\n"
}

func testStateDocument(ids []string) string {
	progress := make([]string, 0, len(ids))
	statuses := make([]string, 0, len(ids))
	for _, id := range ids {
		progress = append(progress, "- [ ] "+id)
		statuses = append(statuses, "- Story: "+id+"\n  - Acceptance: pending: not observed")
	}
	return "---\n" +
		"cycle: " + fixtureCycleID + "\n" +
		"spec_rev: rev-1\n" +
		"status: active\n" +
		"last_updated_at: 2026-09-10T10:00:00+08:00\n" +
		"---\n\n" +
		"## Progress Tree\n\n" + strings.Join(progress, "\n") + "\n\n" +
		"## Current Situation\n\n- none\n\n" +
		"## Active Lanes\n\n- none\n\n" +
		"## Acceptance Status\n\n" + strings.Join(statuses, "\n") + "\n\n" +
		"## Open Issues\n\n- none\n\n" +
		"## Practice Feedback\n\n- none\n"
}

func readTestFile(t *testing.T, file string) string {
	t.Helper()
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func assertDiagnostic(t *testing.T, issues []diagnostic.Issue, fragment string) {
	t.Helper()
	for _, issue := range issues {
		if strings.Contains(issue.Message, fragment) {
			return
		}
	}
	t.Fatalf("message containing %q not found in %#v", fragment, issues)
}

func assertIssueContains(t *testing.T, issues []diagnostic.Issue, fragment string) {
	t.Helper()
	assertDiagnostic(t, issues, fragment)
}

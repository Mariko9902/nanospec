package validator

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestViewSyncPreservesParagraphNotesWithTheirStory(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}, "story-02-payment": {"story-01-order"}})
	cycleFile := filepath.Join(directory, "cycle.md")
	note := "Finance is available to review this item on Wednesday, so schedule it first.\n\n  Schedule payment confirmation after the review."
	cycle := strings.Replace(readTestFile(t, cycleFile), "`story-02-payment`: Save an order\n", "`story-02-payment`: Save an order; Confirm the received payment first\n\n"+note+"\n", 1)
	writeFixtureFile(t, cycleFile, cycle)
	result := SynchronizeCycleViews(directory)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("sync result = %#v", result)
	}
	after := readTestFile(t, cycleFile)
	if !strings.Contains(sectionContent(after, 4, "Wave 2"), note) || !strings.Contains(after, "; Confirm the received payment first") {
		t.Fatalf("Story notes were lost or moved to another Wave:\n%s", after)
	}
	if strings.Contains(sectionContent(after, 3, "Dependency Graph"), "Finance is available") {
		t.Fatal("coordination notes appeared in the dependency graph")
	}
	repeated := SynchronizeCycleViews(directory)
	if repeated.InputError != "" || len(repeated.Errors) != 0 || len(repeated.Updates) != 0 {
		t.Fatalf("repeated sync = %#v", repeated)
	}
}

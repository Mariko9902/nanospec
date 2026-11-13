package validator

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestViewSyncPreservesPlanningNotes(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}, "story-02-payment": {"story-01-order"}})
	file := filepath.Join(directory, "cycle.md")
	note := "Finance reviews the payment on Wednesday; start the order first."
	writeFixtureFile(t, file, strings.Replace(readTestFile(t, file), "Current delivery scope.", note, 1))
	result := SynchronizeCycleViews(directory)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("sync = %#v", result)
	}
	after := readTestFile(t, file)
	if !strings.Contains(sectionContent(after, 2, "Planning Context"), note) {
		t.Fatal("planning note lost")
	}
	if strings.Contains(sectionContent(after, 2, "Dependency Graph"), "Wednesday") {
		t.Fatal("planning note became a graph title")
	}
	repeated := SynchronizeCycleViews(directory)
	if repeated.InputError != "" || len(repeated.Errors) != 0 || len(repeated.Updates) != 0 {
		t.Fatalf("repeated = %#v", repeated)
	}
}

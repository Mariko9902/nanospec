package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nanospec/internal/diagnostic"
)

func TestEvidenceDirectoryCreationRollsBackWithState(t *testing.T) {
	cycle := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	story := filepath.Join(cycle, "story-01-base")
	at := time.Now()
	checkpointStoryDone(t, story, at)
	capture, result := BuildEvidenceUpdate(story, EvidenceCaptureRequest{SpecRevision: "rev-1", ImplementedBy: "executor-a", AcceptanceID: "A1", Observed: "visible", Command: "check", Result: "passed", Artifacts: "evidence/a1.txt", Context: testObservationContext("rev-1", at), At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("%#v", result)
	}
	stateFile := filepath.Join(cycle, "cycle.md")
	before := readTestFile(t, stateFile)
	err := saveViewUpdates(capture.Updates, func(from, to string) error {
		if to == stateFile && strings.Contains(filepath.Base(from), ".tmp-") {
			return fmt.Errorf("write failed")
		}
		return os.Rename(from, to)
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	if _, err := os.Stat(filepath.Join(story, "acceptance")); !os.IsNotExist(err) {
		t.Fatal("acceptance directory not rolled back")
	}
	if actual := readTestFile(t, stateFile); actual != before {
		t.Fatal("state changed")
	}
	if err := SaveViewUpdates(capture.Updates); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(story, "acceptance", "evidence")); err != nil || !info.IsDir() {
		t.Fatal("evidence directory missing")
	}
}

func TestStoryAcceptsEvidenceAssetsAndRejectsRootLevelRecord(t *testing.T) {
	dir := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	writeFixtureFile(t, filepath.Join(dir, "acceptance", "evidence", "screenshots", "page.png"), "pixels")
	var report diagnostic.Report
	ValidateStoryDirectory(dir, fixtureCycleID, &report)
	if report.HasErrors() {
		t.Fatalf("%#v", report.Errors())
	}
	writeFixtureFile(t, filepath.Join(dir, "evidence.md"), "root-level record")
	report = diagnostic.Report{}
	ValidateStoryDirectory(dir, fixtureCycleID, &report)
	found := false
	for _, issue := range report.Errors() {
		found = found || strings.Contains(issue.Message, "acceptance record must be inside acceptance/")
	}
	if !found {
		t.Fatalf("root-level record accepted: %#v", report.Errors())
	}
}

package validator

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

func TestLifecyclePreservesMetadataExamplesInPlanningContext(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	file := filepath.Join(directory, "cycle.md")
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	snippet := "```yaml\nspec_rev: upstream-4\nstatus: available\nlast_updated_at: 2026-11-09T08:00:00Z\n```"
	source := strings.Replace(readTestFile(t, file), "Current delivery scope.", "Current delivery scope.\n\n"+snippet, 1)
	writeFixtureFile(t, file, source)
	result := SynchronizeCycleViews(directory)
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("draft: %#v", result)
	}
	_, result = SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: "active", Next: "save order", Situation: "order selected", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("checkpoint: %#v", result)
	}
	storyFile := filepath.Join(target, "story.md")
	writeFixtureFile(t, storyFile, strings.Replace(readTestFile(t, storyFile), "Save an order", "Save the selected order", 1))
	_, result = ReviseStories([]string{target}, RevisionRequest{Reason: "selected order clarified", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("revise: %#v", result)
	}
	_, result = SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-2", Owner: "executor-a", Status: "active", Complete: true, Next: "none", Situation: "selected order saved", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("resume: %#v", result)
	}
	captureAcceptance(t, target, "A1", at)
	captureAcceptance(t, target, "A2", at)
	_, result = VerifyStory(target, VerificationRequest{SpecRevision: "rev-2", VerifiedBy: "reviewer-a", Mode: VerificationIndependent, Result: "pass", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("verify: %#v", result)
	}
	_, result = AcceptStory(target, AcceptanceRequest{SpecRevision: "rev-2", AcceptedBy: "owner", Method: AcceptanceHuman, At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("accept: %#v", result)
	}
	content := readTestFile(t, file)
	if !strings.Contains(sectionContent(content, 2, "Planning Context"), snippet) {
		t.Fatal("planning metadata example changed")
	}
	var report diagnostic.Report
	document := specdoc.ParseDocument(file, content, &report)
	revision, _ := document.Frontmatter.String("spec_rev")
	status, _ := document.Frontmatter.String("status")
	if revision != "rev-2" || status != "done" {
		t.Fatalf("revision=%s, status=%s", revision, status)
	}
	if result := ValidateSpec(directory, false); result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("final cycle: %#v", result)
	}
}

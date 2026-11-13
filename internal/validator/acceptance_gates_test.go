package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRejectedDependencyBlocksVerificationAndAcceptance(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}, "story-02-payment": {"story-01-order"}})
	if result := SynchronizeCycleViews(directory); result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("sync result = %#v", result)
	}
	at := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{"story-01-order", "story-02-payment"} {
		prepareReadyStory(t, filepath.Join(directory, id), at)
	}
	_, result := RejectStory(filepath.Join(directory, "story-01-order"), RejectionRequest{SpecRevision: "rev-1",
		RejectedBy: "owner", Reason: "stored order is incorrect", At: at.Add(4 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("rejection result = %#v", result)
	}
	target := filepath.Join(directory, "story-02-payment")
	before := readTestFile(t, filepath.Join(directory, "cycle.md"))
	_, result = VerifyStory(target, VerificationRequest{SpecRevision: "rev-1",
		VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(5 * time.Minute),
	})
	if !strings.Contains(result.InputError, "dependency is ready: story-01-order") {
		t.Fatalf("verification result = %#v", result)
	}
	_, result = AcceptStory(target, AcceptanceRequest{SpecRevision: "rev-1", AcceptedBy: "owner", Method: AcceptanceHuman, At: at.Add(6 * time.Minute)})
	if !strings.Contains(result.InputError, "dependency is ready: story-01-order") {
		t.Fatalf("acceptance result = %#v", result)
	}
	if readTestFile(t, filepath.Join(directory, "cycle.md")) != before || fileExists(filepath.Join(target, "acceptance", "outcome.md")) {
		t.Fatal("blocked acceptance changed the delivery state")
	}
}

func TestReadyDependencyAllowsAcceptance(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}, "story-02-payment": {"story-01-order"}})
	if result := SynchronizeCycleViews(directory); result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("sync result = %#v", result)
	}
	at := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{"story-01-order", "story-02-payment"} {
		prepareReadyStory(t, filepath.Join(directory, id), at)
	}
	_, result := AcceptStory(filepath.Join(directory, "story-02-payment"), AcceptanceRequest{SpecRevision: "rev-1",
		AcceptedBy: "owner", Method: AcceptanceHuman, At: at.Add(4 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("acceptance result = %#v", result)
	}
	if result := ValidateSpec(directory, false); len(result.Errors) != 0 {
		t.Fatalf("validation result = %#v", result)
	}
}

func TestAcceptanceRejectsStoryChangedBeforeSave(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	prepareReadyStory(t, target, at)
	commit, result := BuildAcceptanceCommit(target, AcceptanceRequest{SpecRevision: "rev-1",
		AcceptedBy: "owner", Method: AcceptanceHuman, At: at.Add(4 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("acceptance result = %#v", result)
	}
	stateFile := filepath.Join(directory, "cycle.md")
	beforeState := readTestFile(t, stateFile)
	storyFile := filepath.Join(target, "story.md")
	writeFixtureFile(t, storyFile, readTestFile(t, storyFile)+"\n")
	if err := SaveAcceptanceCommit(commit); err == nil || !strings.Contains(err.Error(), "story.md changed during acceptance") {
		t.Fatalf("save error = %v", err)
	}
	if readTestFile(t, stateFile) != beforeState || readTestFile(t, commit.EvidenceFile) != commit.EvidenceBefore {
		t.Fatal("failed acceptance changed the state or evidence")
	}
	if _, err := os.Stat(commit.OutcomeFile); !os.IsNotExist(err) {
		t.Fatal("failed acceptance created an Outcome")
	}
	if result := ValidateSpec(directory, false); len(result.Errors) != 0 {
		t.Fatalf("validation result = %#v", result)
	}
}

func prepareReadyStory(t *testing.T, target string, at time.Time) {
	t.Helper()
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	captureAcceptance(t, target, "A2", at.Add(2*time.Minute))
	_, result := VerifyStory(target, VerificationRequest{SpecRevision: "rev-1",
		VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(3 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("verification result = %#v", result)
	}
}

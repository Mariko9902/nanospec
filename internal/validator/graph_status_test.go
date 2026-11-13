package validator

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func startTestRework(t *testing.T, target, revision string, at time.Time) {
	t.Helper()
	_, result := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: revision, Owner: "executor-a", Status: "rework", Next: "address review findings", Situation: "rework assigned and started", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("start rework: %#v", result)
	}
}

func assertStoryViews(t *testing.T, directory, id string, expected StoryStatus) CycleSnapshot {
	t.Helper()
	snapshot, result := LoadCycleSnapshot(directory)
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("snapshot: %#v", result)
	}
	graph, _ := snapshot.View("graph")
	overview, _ := snapshot.View("overview")
	number := storyIDPattern.FindStringSubmatch(id)[1]
	if !strings.Contains(graph, "S"+number+"[\"["+string(expected)+"] S"+number+" ") || !strings.Contains(overview, "## ["+string(expected)+"] ") {
		t.Fatalf("graph or overview missing %s:\n%s\n%s", expected, graph, overview)
	}
	if strings.Contains(graph, "(rejected)") {
		t.Fatal("graph combines multiple states")
	}
	return snapshot
}

func TestGraphTracksAllSevenDeliveryStages(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}, "story-02-product": {}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	assertStoryViews(t, directory, "story-01-order", StoryPending)
	checkpointStoryDone(t, target, at)
	assertStoryViews(t, directory, "story-01-order", StoryActive)
	captureAcceptance(t, target, "A1", at)
	captureAcceptance(t, target, "A2", at)
	_, result := VerifyStory(target, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer-a", Mode: VerificationIndependent, Result: "pass", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("verify: %#v", result)
	}
	assertStoryViews(t, directory, "story-01-order", StoryReady)
	_, result = VerifyStory(target, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer-a", Mode: VerificationIndependent, Result: "rework", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("review: %#v", result)
	}
	rejected := assertStoryViews(t, directory, "story-01-order", StoryRejected)
	lane, _ := storyLane("story-01-order", rejected.ActiveLanes)
	if lane.Fields["Owner"] != "executor-a" || lane.Fields["Next"] == "none" {
		t.Fatal("rejection lost owner or next action")
	}
	if prepared, result := BuildExecutionPackage(target, "# Rules\n\nUse current facts."); result.InputError != "" || len(result.Errors) > 0 || prepared.Status != StoryRejected {
		t.Fatalf("prepare changed rejected work: %#v %#v", prepared, result)
	}
	assertStoryViews(t, directory, "story-01-order", StoryRejected)
	startTestRework(t, target, "rev-1", at)
	rework := assertStoryViews(t, directory, "story-01-order", StoryRework)
	resumed, _ := storyLane("story-01-order", rework.ActiveLanes)
	if resumed.ID != lane.ID {
		t.Fatal("rework replaced the original lane")
	}
	_, result = SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: "blocked", Next: "restore account", Situation: "account unavailable", Issue: "account unavailable", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("block: %#v", result)
	}
	blocked := assertStoryViews(t, directory, "story-01-order", StoryBlocked)
	waiting, _ := storyLane("story-01-order", blocked.ActiveLanes)
	if waiting.Fields["Resume Status"] != "rework" {
		t.Fatal("blocking lost the rework stage")
	}
	_, result = SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: "rework", Complete: true, Next: "none", Situation: "repair checked", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("resume: %#v", result)
	}
	assertStoryViews(t, directory, "story-01-order", StoryRework)
	captureAcceptance(t, target, "A1", at)
	captureAcceptance(t, target, "A2", at)
	_, result = VerifyStory(target, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer-a", Mode: VerificationIndependent, Result: "pass", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("recheck: %#v", result)
	}
	assertStoryViews(t, directory, "story-01-order", StoryReady)
	_, result = AcceptStory(target, AcceptanceRequest{SpecRevision: "rev-1", AcceptedBy: "owner", Method: AcceptanceHuman, At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("accept: %#v", result)
	}
	accepted := assertStoryViews(t, directory, "story-01-order", StoryAccepted)
	if _, found := storyLane("story-01-order", accepted.ActiveLanes); found {
		t.Fatal("accepted delivery retained its lane")
	}
	assertStoryViews(t, directory, "story-02-product", StoryPending)
}

func TestGraphEditCannotAcceptAStory(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	result := SynchronizeCycleViews(directory)
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("sync: %#v", result)
	}
	file := filepath.Join(directory, "cycle.md")
	writeFixtureFile(t, file, strings.Replace(readTestFile(t, file), "[pending] S01", "[accepted] S01", 1))
	result = ValidateSpec(directory, false)
	assertDiagnostic(t, result.Errors, "Dependency Graph is out of date")
	result = SynchronizeCycleViews(directory)
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("restore: %#v", result)
	}
	snapshot := assertStoryViews(t, directory, "story-01-order", StoryPending)
	if acceptanceState("story-01-order", snapshot.Acceptance) != "pending" {
		t.Fatal("graph changed acceptance")
	}
}

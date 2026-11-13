package validator

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func requireCheckpoint(t *testing.T, target, status, next, issue string, complete bool) {
	t.Helper()
	_, result := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: status, Complete: complete, Next: next, Situation: "current condition checked", Issue: issue, At: time.Now()})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("checkpoint %s: %#v", status, result)
	}
}

func TestRejectedWorkWaitsForExplicitReworkStart(t *testing.T) {
	for _, human := range []bool{false, true} {
		t.Run(map[bool]string{false: "review", true: "human"}[human], func(t *testing.T) {
			directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
			target := filepath.Join(directory, "story-01-order")
			at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
			prepareReadyStory(t, target, at)
			if human {
				_, result := RejectStory(target, RejectionRequest{SpecRevision: "rev-1", RejectedBy: "owner", Reason: "wrong amount", At: at})
				if result.InputError != "" || len(result.Errors) > 0 {
					t.Fatalf("reject: %#v", result)
				}
			} else {
				_, result := VerifyStory(target, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer", Mode: VerificationIndependent, Result: "rework", At: at})
				if result.InputError != "" || len(result.Errors) > 0 {
					t.Fatalf("review: %#v", result)
				}
			}
			file := filepath.Join(directory, "cycle.md")
			before := readTestFile(t, file)
			packageResult, validation := BuildExecutionPackage(target, "# Rules\n\nRead current facts.")
			if validation.InputError != "" || len(validation.Errors) > 0 || packageResult.Status != StoryRejected {
				t.Fatalf("prepare: %#v %#v", packageResult, validation)
			}
			if readTestFile(t, file) != before {
				t.Fatal("prepare started repair")
			}
			_, result := RecordStoryEvidence(target, EvidenceCaptureRequest{SpecRevision: "rev-1", ImplementedBy: "executor-a", AcceptanceID: "A1", Observed: "amount displayed", Command: "inspect order", Result: "correct amount", Artifacts: "evidence/order.json", Context: testObservationContext("rev-1", at), At: at})
			if !strings.Contains(result.InputError, "status is rejected") {
				t.Fatalf("evidence bypassed assignment: %#v", result)
			}
			_, result = SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: "active", Next: "change order", Situation: "review read", At: at})
			if !strings.Contains(result.InputError, "cannot checkpoint rejected as active") {
				t.Fatalf("active bypassed rework: %#v", result)
			}
			if readTestFile(t, file) != before {
				t.Fatal("rejected attempt changed state")
			}
			startTestRework(t, target, "rev-1", at)
			captureAcceptance(t, target, "A1", at)
			assertStoryViews(t, directory, "story-01-order", StoryRework)
		})
	}
}

func TestBlockedReadyCannotBeReviewedAcceptedOrUsedAsDependency(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}, "story-02-payment": {"story-01-order"}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	prepareReadyStory(t, target, at)
	requireCheckpoint(t, target, "blocked", "restore service", "service unavailable", false)
	requireCheckpoint(t, target, "blocked", "restore service access", "service still unavailable", false)
	snapshot := assertStoryViews(t, directory, "story-01-order", StoryBlocked)
	lane, _ := storyLane("story-01-order", snapshot.ActiveLanes)
	if lane.Fields["Resume Status"] != "ready" || !implementationComplete(lane) {
		t.Fatal("blocked ready lost its prior stage or completion")
	}
	item, validation := BuildExecutionPackage(target, "# Rules\n\nRead current facts.")
	packages := []ExecutionPackage{item}
	ResolvePreparationGates(packages)
	if validation.InputError != "" || len(validation.Errors) > 0 || packages[0].Gate != "blocked" || item.ResumeStatus == nil || *item.ResumeStatus != StoryReady {
		t.Fatalf("package lost blocker: %#v %#v", item, validation)
	}
	child, result := BuildExecutionPackage(filepath.Join(directory, "story-02-payment"), "# Rules\n\nRead current facts.")
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("child: %#v", result)
	}
	selected := []ExecutionPackage{child}
	ResolvePreparationGates(selected)
	if selected[0].Gate != "blocked_by_dependency" || child.Dependencies[0].Status != StoryBlocked {
		t.Fatalf("blocked ready prerequisite appeared usable: %#v", selected)
	}
	file := filepath.Join(directory, "cycle.md")
	before := readTestFile(t, file)
	_, result = VerifyStory(target, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer", Mode: VerificationIndependent, Result: "pass", At: at})
	if !strings.Contains(result.InputError, "status is blocked") {
		t.Fatalf("blocked verification: %#v", result)
	}
	_, result = AcceptStory(target, AcceptanceRequest{SpecRevision: "rev-1", AcceptedBy: "owner", Method: AcceptanceHuman, At: at})
	if !strings.Contains(result.InputError, "ready") {
		t.Fatalf("blocked acceptance: %#v", result)
	}
	_, result = SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: "rework", Next: "repair order", Situation: "service returned", At: at})
	if !strings.Contains(result.InputError, "cannot checkpoint ready as rework") {
		t.Fatalf("recovery skipped stage: %#v", result)
	}
	if readTestFile(t, file) != before {
		t.Fatal("failed recovery changed facts")
	}
	requireCheckpoint(t, target, "ready", "wait for review", "", false)
	assertStoryViews(t, directory, "story-01-order", StoryReady)
	_, result = AcceptStory(target, AcceptanceRequest{SpecRevision: "rev-1", AcceptedBy: "owner", Method: AcceptanceHuman, At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("restored ready acceptance: %#v", result)
	}
}

func TestRevisionKeepsExistingBlockerAndInvalidatesReadyRecovery(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	prepareReadyStory(t, target, at)
	requireCheckpoint(t, target, "blocked", "restore account", "account unavailable", false)
	_, result := ReviseStories([]string{target}, RevisionRequest{Reason: "order rules clarified", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("revise: %#v", result)
	}
	snapshot := assertStoryViews(t, directory, "story-01-order", StoryBlocked)
	lane, _ := storyLane("story-01-order", snapshot.ActiveLanes)
	if lane.Fields["Next"] != "restore account" || lane.Fields["Resume Status"] != "active" || implementationComplete(lane) {
		t.Fatalf("revision lost recovery or kept old ready: %#v", lane)
	}
	if !strings.Contains(sectionContent(snapshot.CycleText, 2, "Open Issues"), "account unavailable") {
		t.Fatal("revision erased original blocker")
	}
	_, result = SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-2", Owner: "executor-a", Status: "ready", Next: "accept result", Situation: "account restored", At: at})
	if !strings.Contains(result.InputError, "cannot checkpoint active as ready") {
		t.Fatalf("revised evidence restored ready: %#v", result)
	}
}

func TestRejectedBlockRestoresDecisionWithoutStartingRepair(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	prepareReadyStory(t, target, time.Now())
	_, result := RejectStory(target, RejectionRequest{SpecRevision: "rev-1", RejectedBy: "owner", Reason: "wrong amount", At: time.Now()})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("reject: %#v", result)
	}
	requireCheckpoint(t, target, "blocked", "resolve account", "account unavailable", false)
	requireCheckpoint(t, target, "rejected", "await repair assignment", "", false)
	assertStoryViews(t, directory, "story-01-order", StoryRejected)
	startTestRework(t, target, "rev-1", time.Now())
	assertStoryViews(t, directory, "story-01-order", StoryRework)
}

func TestRevisionDoesNotStartOrBlockPendingWork(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "dependency waiting", true: "unstarted blocker"}[blocked], func(t *testing.T) {
			directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}, "story-02-payment": {"story-01-order"}})
			target := filepath.Join(directory, "story-02-payment")
			stage, next, issue := "pending", "await implementation assignment", ""
			if blocked {
				stage, next, issue = "blocked", "restore account", "account unavailable"
			}
			requireCheckpoint(t, target, stage, next, issue, false)
			_, result := ReviseStories([]string{target}, RevisionRequest{Reason: "payment field clarified", At: time.Now()})
			if result.InputError != "" || len(result.Errors) > 0 {
				t.Fatalf("revise: %#v", result)
			}
			snapshot := assertStoryViews(t, directory, "story-02-payment", StoryStatus(stage))
			lane, _ := storyLane("story-02-payment", snapshot.ActiveLanes)
			if lane.Fields["Spec Changes"] != "0" || lane.Fields["Next"] != next {
				t.Fatalf("unstarted revision changed execution facts: %#v", lane)
			}
			if blocked && (lane.Fields["Resume Status"] != "pending" || !strings.Contains(sectionContent(snapshot.CycleText, 2, "Open Issues"), issue)) {
				t.Fatal("unstarted blocker lost its recovery facts")
			}
			selected := make([]ExecutionPackage, 0, 2)
			for _, id := range []string{"story-01-order", "story-02-payment"} {
				item, result := BuildExecutionPackage(filepath.Join(directory, id), "# Rules\n\nRead current facts.")
				if result.InputError != "" || len(result.Errors) > 0 {
					t.Fatalf("prepare: %#v", result)
				}
				selected = append(selected, item)
			}
			ResolvePreparationGates(selected)
			expected := "waiting_for_selected_dependency"
			if blocked {
				expected = "blocked"
			}
			if selected[1].Gate != expected {
				t.Fatalf("gate=%s, want=%s", selected[1].Gate, expected)
			}
		})
	}
}

func TestHumanRejectionWhileBlockedPreservesRecovery(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	prepareReadyStory(t, target, time.Now())
	requireCheckpoint(t, target, "blocked", "restore account", "account unavailable", false)
	_, result := RejectStory(target, RejectionRequest{SpecRevision: "rev-1", RejectedBy: "owner", Reason: "wrong amount", At: time.Now()})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("reject blocked ready: %#v", result)
	}
	snapshot := assertStoryViews(t, directory, "story-01-order", StoryBlocked)
	lane, _ := storyLane("story-01-order", snapshot.ActiveLanes)
	if lane.Fields["Resume Status"] != "rejected" || lane.Fields["Next"] != "restore account" || !strings.Contains(sectionContent(snapshot.CycleText, 2, "Open Issues"), "account unavailable") {
		t.Fatalf("rejection erased the blocker: %#v", lane)
	}
	requireCheckpoint(t, target, "rejected", "await repair assignment", "", false)
	assertStoryViews(t, directory, "story-01-order", StoryRejected)
}

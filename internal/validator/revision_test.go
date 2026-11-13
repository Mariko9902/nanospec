package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecutionWritesRejectStalePreparedRevision(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}, "story-02-product": {}})
	at := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)
	target := filepath.Join(directory, "story-02-product")
	for _, id := range []string{"story-01-order", "story-02-product"} {
		story := filepath.Join(directory, id)
		checkpointStoryDone(t, story, at)
		captureAcceptance(t, story, "A1", at)
		captureAcceptance(t, story, "A2", at)
		_, result := VerifyStory(story, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer-a", Mode: VerificationIndependent, Result: "pass", At: at})
		if result.InputError != "" || len(result.Errors) > 0 {
			t.Fatalf("verify: %#v", result)
		}
	}
	_, result := ReviseStories([]string{filepath.Join(directory, "story-01-order")}, RevisionRequest{Reason: "order behavior changed", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("revise: %#v", result)
	}
	cycleFile, evidenceFile := filepath.Join(directory, "cycle.md"), filepath.Join(target, "acceptance", "evidence.md")
	beforeCycle, beforeEvidence := readTestFile(t, cycleFile), readTestFile(t, evidenceFile)
	operations := map[string]func() ValidationResult{
		"checkpoint": func() ValidationResult {
			_, r := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: "active", Complete: true, Next: "none", Situation: "complete", At: at})
			return r
		},
		"evidence": func() ValidationResult {
			_, r := RecordStoryEvidence(target, EvidenceCaptureRequest{SpecRevision: "rev-1", ImplementedBy: "executor-a", AcceptanceID: "A1", Observed: "product saved", Command: "check product", Result: "product found", Artifacts: "evidence/product.txt", Context: testObservationContext("rev-1", at), At: at})
			return r
		},
		"verify": func() ValidationResult {
			_, r := VerifyStory(target, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer-a", Mode: VerificationIndependent, Result: "pass", At: at})
			return r
		},
		"accept": func() ValidationResult {
			_, r := AcceptStory(target, AcceptanceRequest{SpecRevision: "rev-1", AcceptedBy: "owner", Method: AcceptanceHuman, At: at})
			return r
		},
		"reject": func() ValidationResult {
			_, r := RejectStory(target, RejectionRequest{SpecRevision: "rev-1", RejectedBy: "owner", Reason: "incorrect result", At: at})
			return r
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			r := operation()
			if !strings.Contains(r.InputError, "prepared rev-1, current rev-2") || !strings.Contains(r.InputError, "rerun nanospec prepare") {
				t.Fatalf("stale write: %#v", r)
			}
			if readTestFile(t, cycleFile) != beforeCycle || readTestFile(t, evidenceFile) != beforeEvidence {
				t.Fatal("stale write modified current facts")
			}
			if _, err := os.Stat(filepath.Join(target, "acceptance", "outcome.md")); !os.IsNotExist(err) {
				t.Fatal("stale acceptance created an Outcome")
			}
		})
	}
	prepared, validation := BuildExecutionPackage(target, "# Execution Rules\n\nCheck the prepared revision.")
	if validation.InputError != "" || len(validation.Errors) > 0 || prepared.SpecRevision != "rev-2" {
		t.Fatalf("prepare: %#v, %#v", prepared, validation)
	}
	_, accepted := AcceptStory(target, AcceptanceRequest{SpecRevision: prepared.SpecRevision, AcceptedBy: "owner", Method: AcceptanceHuman, At: at})
	if accepted.InputError != "" || len(accepted.Errors) > 0 {
		t.Fatalf("unaffected result after fresh preparation: %#v", accepted)
	}
}

func TestCheckpointRequiresPreparedRevision(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	before := readTestFile(t, filepath.Join(directory, "cycle.md"))
	for _, revision := range []string{"", "latest", "rev-0"} {
		_, r := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: revision, Owner: "executor-a", Status: "active", Next: "save order", Situation: "order selected", At: time.Now()})
		if !strings.Contains(r.InputError, "spec-revision must match rev-N") {
			t.Fatalf("revision %q: %#v", revision, r)
		}
	}
	if readTestFile(t, filepath.Join(directory, "cycle.md")) != before {
		t.Fatal("missing revision changed the Cycle")
	}
}

func TestReobservedEvidenceRefreshesSummaryContext(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 11, 10, 11, 0, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at)
	captureAcceptance(t, target, "A2", at)
	evidenceFile := filepath.Join(target, "acceptance", "evidence.md")
	snapshot, _ := LoadCycleSnapshot(directory)
	record, _, result := loadEvidenceRecord(evidenceFile, snapshot.Stories["story-01-order"])
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("evidence: %#v", result)
	}
	oldContext := record.Results["A2"].evidenceContext
	_, result = ReviseStories([]string{target}, RevisionRequest{Reason: "saved order display changed", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("revise: %#v", result)
	}
	_, resumed := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-2", Owner: "executor-a", Status: "active", Next: "reobserve revised result", Situation: "revised specification checked", At: at})
	if resumed.InputError != "" || len(resumed.Errors) > 0 {
		t.Fatalf("resume: %#v", resumed)
	}
	currentContext := testObservationContext("rev-2", at.Add(time.Minute))
	_, result = RecordStoryEvidence(target, EvidenceCaptureRequest{
		SpecRevision: "rev-2", ImplementedBy: "executor-a", AcceptanceID: "A1",
		Observed: "saved order displayed", Command: "inspect order", Result: "order found",
		Artifacts: "evidence/order.txt", Context: currentContext, At: at.Add(time.Minute),
	})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("reobserve: %#v", result)
	}
	record, _, result = loadEvidenceRecord(evidenceFile, snapshot.Stories["story-01-order"])
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("evidence: %#v", result)
	}
	if record.Context != currentContext || record.Results["A1"].evidenceContext != currentContext {
		t.Fatal("current observation context was not reflected in the summary")
	}
	if record.Results["A2"].evidenceContext != oldContext || record.Results["A2"].status != "stale" {
		t.Fatal("old evidence lost its original context or stale status")
	}
}

package validator

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckpointRejectsOwnerChangeAfterEvidence(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	stateFile := filepath.Join(directory, "cycle.md")
	evidenceFile := filepath.Join(target, "acceptance", "evidence.md")
	beforeState, beforeEvidence := readTestFile(t, stateFile), readTestFile(t, evidenceFile)

	_, result := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1",
		Owner: "clerk-b", Status: "active", Next: "confirm the remaining result", Situation: "record selected", At: at.Add(2 * time.Minute),
	})
	if !strings.Contains(result.InputError, "cannot change Owner after evidence is recorded") {
		t.Fatalf("checkpoint result = %#v", result)
	}
	if readTestFile(t, stateFile) != beforeState || readTestFile(t, evidenceFile) != beforeEvidence {
		t.Fatal("rejected owner change modified records")
	}
	captureAcceptance(t, target, "A2", at.Add(3*time.Minute))
}

func TestCheckpointAllowsOwnerChangeBeforeEvidence(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	for _, owner := range []string{"clerk-a", "clerk-b"} {
		_, result := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1",
			Owner: owner, Status: "active", Next: "save order", Situation: "order selected", At: at,
		})
		if result.InputError != "" || len(result.Errors) != 0 {
			t.Fatalf("checkpoint result = %#v", result)
		}
	}
	_, result := RecordStoryEvidence(target, EvidenceCaptureRequest{SpecRevision: "rev-1",
		ImplementedBy: "clerk-b", AcceptanceID: "A1", Observed: "order saved", Command: "none: manual observation",
		Result: "saved order displayed", Artifacts: "evidence/order.png", Context: testObservationContext("rev-1", at), At: at.Add(time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("evidence result = %#v", result)
	}
}

func TestCheckpointRejectsUnusableOwnerIdentifiers(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	stateFile := filepath.Join(directory, "cycle.md")
	before := readTestFile(t, stateFile)
	for _, owner := range []string{"", "Jane Doe", "clerk:one", "-clerk"} {
		t.Run(owner, func(t *testing.T) {
			_, result := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-1",
				Owner: owner, Status: "active", Next: "save order", Situation: "order selected", At: time.Now(),
			})
			if result.InputError != "owner must be a stable identifier" {
				t.Fatalf("result = %#v", result)
			}
			if readTestFile(t, stateFile) != before {
				t.Fatal("invalid owner modified the execution state")
			}
		})
	}
}

func TestValidateRejectsUnusableLaneOwner(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	checkpointStoryDone(t, target, time.Now())
	stateFile := filepath.Join(directory, "cycle.md")
	writeFixtureFile(t, stateFile, strings.Replace(readTestFile(t, stateFile), "Owner: executor-a", "Owner: Jane Doe", 1))
	result := ValidateSpec(directory, false)
	assertDiagnostic(t, result.Errors, "Owner must be a stable identifier")
}

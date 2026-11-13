package validator

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEvidenceRejectsSharedContextReferencesWithoutChangingRecords(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at)
	cycleFile, evidenceFile := filepath.Join(directory, "cycle.md"), filepath.Join(target, "acceptance", "evidence.md")
	beforeCycle, beforeEvidence := readTestFile(t, cycleFile), readTestFile(t, evidenceFile)
	for _, context := range []string{"same as shared Verification Context", "Verification Context", "see verification   context", "same as A1 Evidence Context", "same as A1", "同上", "参见 A1", "same as shared context"} {
		_, result := RecordStoryEvidence(target, EvidenceCaptureRequest{SpecRevision: "rev-1", ImplementedBy: "executor-a", AcceptanceID: "A2", Observed: "order unchanged", Command: "save order", Result: "existing amount preserved", Artifacts: "evidence/order.txt", Context: context, At: at})
		if !strings.Contains(result.InputError, "context must be a JSON object with spec_revision, code_revision, environment and observed_at") {
			t.Fatalf("context %q: %#v", context, result)
		}
		if readTestFile(t, cycleFile) != beforeCycle || readTestFile(t, evidenceFile) != beforeEvidence {
			t.Fatal("rejected reference changed execution or evidence")
		}
	}
}

func TestEvidenceContextChangesPreserveEachOriginalObservation(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
	target := filepath.Join(directory, "story-01-order")
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	contexts := map[string]string{"A1": testObservationContext("rev-1", at), "A2": testObservationContext("rev-1", at.Add(10*time.Minute))}
	for _, id := range []string{"A1", "A2"} {
		_, result := RecordStoryEvidence(target, EvidenceCaptureRequest{SpecRevision: "rev-1", ImplementedBy: "executor-a", AcceptanceID: id, Observed: "order result displayed", Command: "inspect order", Result: "expected amount displayed", Artifacts: "evidence/order.txt", Context: contexts[id], At: at})
		if result.InputError != "" || len(result.Errors) > 0 {
			t.Fatalf("capture: %#v", result)
		}
	}
	_, result := VerifyStory(target, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer-a", Mode: VerificationIndependent, Result: "pass", At: at})
	if result.InputError != "" || len(result.Errors) > 0 {
		t.Fatalf("verify: %#v", result)
	}
	snapshot, _ := LoadCycleSnapshot(directory)
	record, _, result := loadEvidenceRecord(filepath.Join(target, "acceptance", "evidence.md"), snapshot.Stories["story-01-order"])
	if result.InputError != "" || len(result.Errors) > 0 || record.Context != contexts["A2"] {
		t.Fatalf("summary: %#v, %#v", record, result)
	}
	for id, context := range contexts {
		if record.Results[id].evidenceContext != context {
			t.Fatalf("%s observation context changed", id)
		}
	}
}

func TestVerificationRejectsUnresolvedContextInExistingEvidence(t *testing.T) {
	for _, summary := range []bool{false, true} {
		t.Run(map[bool]string{false: "item", true: "summary"}[summary], func(t *testing.T) {
			directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
			target := filepath.Join(directory, "story-01-order")
			at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
			checkpointStoryDone(t, target, at)
			captureAcceptance(t, target, "A1", at)
			captureAcceptance(t, target, "A2", at)
			evidenceFile, cycleFile := filepath.Join(target, "acceptance", "evidence.md"), filepath.Join(directory, "cycle.md")
			source := readTestFile(t, evidenceFile)
			prefix := "  - Evidence Context: "
			if summary {
				prefix = "Verification Context: "
			}
			lines := strings.Split(source, "\n")
			for index, line := range lines {
				if strings.HasPrefix(line, prefix) {
					lines[index] = prefix + "same as shared Verification Context"
					break
				}
			}
			invalid := strings.Join(lines, "\n")
			writeFixtureFile(t, evidenceFile, invalid)
			beforeCycle := readTestFile(t, cycleFile)
			_, result := VerifyStory(target, VerificationRequest{SpecRevision: "rev-1", VerifiedBy: "reviewer-a", Mode: VerificationIndependent, Result: "pass", At: at})
			assertDiagnostic(t, result.Errors, "context must be a JSON object with spec_revision, code_revision, environment and observed_at")
			if readTestFile(t, evidenceFile) != invalid || readTestFile(t, cycleFile) != beforeCycle {
				t.Fatal("invalid evidence changed during verification")
			}
		})
	}
}

func testObservationContext(revision string, at time.Time) string {
	return `{"spec_revision":"` + revision + `","code_revision":"` + strings.Repeat("a", 40) + `","environment":"order-desk","observed_at":"` + at.Format(time.RFC3339) + `"}`
}

func TestObservationContextRequiresCompleteVersionedMetadata(t *testing.T) {
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	valid := testObservationContext("rev-1", at)
	for _, input := range []string{
		"same as A1", `"same as A1"`, "同上", "{}", "null", valid + valid,
		strings.Replace(valid, `"spec_revision":"rev-1",`, "", 1),
		strings.Replace(valid, strings.Repeat("a", 40), "same as A1", 1),
		strings.Replace(valid, `"environment":"order-desk"`, `"environment":""`, 1),
		strings.Replace(valid, at.Format(time.RFC3339), "now", 1),
		strings.Replace(valid, `"observed_at":`, `"other":"A1","observed_at":`, 1),
	} {
		if _, err := parseObservationContext(input); err == nil {
			t.Fatalf("accepted incomplete context: %s", input)
		}
	}
	digestContext := strings.Replace(valid, strings.Repeat("a", 40), "sha256:"+strings.Repeat("b", 64), 1)
	if _, err := parseObservationContext(digestContext); err != nil {
		t.Fatalf("artifact context: %v", err)
	}
	if _, err := normalizeObservationContext(valid, "rev-2"); err == nil {
		t.Fatal("context revision did not match the prepared revision")
	}
}

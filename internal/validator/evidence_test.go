package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEvidenceVerificationAndHumanAcceptanceLifecycle(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	checkpointStoryDone(t, target, at)

	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	captureAcceptance(t, target, "A2", at.Add(2*time.Minute))
	verification, result := BuildVerificationUpdate(target, VerificationRequest{SpecRevision: "rev-1",
		VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(3 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("verification result = %#v", result)
	}
	if err := SaveViewUpdates(verification.Updates); err != nil {
		t.Fatal(err)
	}

	commit, result := BuildAcceptanceCommit(target, AcceptanceRequest{SpecRevision: "rev-1",
		AcceptedBy: "product-owner", Method: AcceptanceHuman, At: at.Add(4 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("acceptance result = %#v", result)
	}
	if err := SaveAcceptanceCommit(commit); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "acceptance", "evidence.md")); !os.IsNotExist(err) {
		t.Fatalf("evidence.md still exists: %v", err)
	}
	outcome := readTestFile(t, filepath.Join(target, "acceptance", "outcome.md"))
	storySource, err := os.ReadFile(filepath.Join(target, "story.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"spec_rev: rev-1", "story_digest: " + storyContentDigest(storySource),
		"Verification Context: " + testObservationContext("rev-1", at.Add(2*time.Minute)),
		"verification_mode: self", "verified_acceptance: [A1, A2]", "first_pass: true",
		"accepted_by: product-owner", "acceptance_method: human", "- A1: confirmed", "- A2: confirmed",
	} {
		if !strings.Contains(outcome, fragment) {
			t.Fatalf("outcome missing %q:\n%s", fragment, outcome)
		}
	}
	state := readTestFile(t, filepath.Join(cycleDirectory, "cycle.md"))
	for _, fragment := range []string{"status: done", "[accepted] S01", "Acceptance: accepted:", "## Active Lanes\n\n- none"} {
		if !strings.Contains(state, fragment) {
			t.Fatalf("state missing %q:\n%s", fragment, state)
		}
	}
	if validated := ValidateSpec(cycleDirectory, false); len(validated.Errors) != 0 {
		t.Fatalf("errors = %#v", validated.Errors)
	}
}

func TestAcceptedStoryMutationBreaksOutcomeBinding(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 15, 30, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	captureAcceptance(t, target, "A2", at.Add(2*time.Minute))
	verification, result := BuildVerificationUpdate(target, VerificationRequest{SpecRevision: "rev-1",
		VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(3 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("verification result = %#v", result)
	}
	if err := SaveViewUpdates(verification.Updates); err != nil {
		t.Fatal(err)
	}
	commit, result := BuildAcceptanceCommit(target, AcceptanceRequest{SpecRevision: "rev-1",
		AcceptedBy: "product-owner", Method: AcceptanceHuman, At: at.Add(4 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("acceptance result = %#v", result)
	}
	if err := SaveAcceptanceCommit(commit); err != nil {
		t.Fatal(err)
	}
	storyFile := filepath.Join(target, "story.md")
	writeFixtureFile(t, storyFile, readTestFile(t, storyFile)+"\nchanged after acceptance\n")

	validated := ValidateSpec(cycleDirectory, false)
	assertIssueContains(t, validated.Errors, "outcome.story_digest does not match story.md")
}

func TestRevisionMakesExistingEvidenceStale(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 16, 0, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	captureAcceptance(t, target, "A2", at.Add(2*time.Minute))

	revision, result := BuildRevisionUpdate([]string{target}, RevisionRequest{Reason: "acceptance changed", At: at.Add(3 * time.Minute)})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("revision result = %#v", result)
	}
	if len(revision.Updates) != 2 {
		t.Fatalf("updates = %#v", revision.Updates)
	}
	if err := SaveViewUpdates(revision.Updates); err != nil {
		t.Fatal(err)
	}
	evidence := readTestFile(t, filepath.Join(target, "acceptance", "evidence.md"))
	for _, fragment := range []string{"spec_rev: rev-2", "spec_change_count: 1", "- A1: stale", "- A2: stale"} {
		if !strings.Contains(evidence, fragment) {
			t.Fatalf("evidence missing %q:\n%s", fragment, evidence)
		}
	}
	if validated := ValidateSpec(cycleDirectory, false); len(validated.Errors) != 0 {
		t.Fatalf("errors = %#v", validated.Errors)
	}

	checkpoint, checked := SaveStoryCheckpoint(target, CheckpointRequest{SpecRevision: "rev-2", Owner: "executor-a", Status: "active", Complete: true, Next: "none", Situation: "revised result implemented", At: at.Add(4 * time.Minute)})
	if checked.InputError != "" || len(checked.Errors) > 0 || checkpoint.Status != "active" {
		t.Fatalf("resume: %#v", checked)
	}
	captureAcceptance(t, target, "A1", at.Add(4*time.Minute))
	_, result = BuildVerificationUpdate(target, VerificationRequest{SpecRevision: "rev-2",
		VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(5 * time.Minute),
	})
	if !strings.Contains(result.InputError, "current evidence for A2") {
		t.Fatalf("verification result = %#v", result)
	}
}

func TestEvidenceRejectsNonCanonicalAcceptanceID(t *testing.T) {
	_, result := BuildEvidenceUpdate("unused", EvidenceCaptureRequest{SpecRevision: "rev-1",
		ImplementedBy: "executor-a", AcceptanceID: "QA1", Observed: "observed", Command: "check",
		Result: "passed", Artifacts: "artifact", Context: "context", At: time.Now(),
	})
	if !strings.Contains(result.InputError, "stable Acceptance ID") {
		t.Fatalf("result = %#v", result)
	}
}

func TestRevisionCarriesUnaffectedEvidenceToNewCycleRevision(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base": {},
		"story-02-help": {},
	})
	at := time.Date(2026, 9, 10, 16, 30, 0, 0, time.UTC)
	for _, storyID := range []string{"story-01-base", "story-02-help"} {
		target := filepath.Join(cycleDirectory, storyID)
		checkpointStoryDone(t, target, at)
		captureAcceptance(t, target, "A1", at.Add(time.Minute))
		captureAcceptance(t, target, "A2", at.Add(2*time.Minute))
	}

	revision, result := BuildRevisionUpdate(
		[]string{filepath.Join(cycleDirectory, "story-01-base")},
		RevisionRequest{Reason: "base behavior changed", At: at.Add(3 * time.Minute)},
	)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("revision result = %#v", result)
	}
	if len(revision.Updates) != 3 {
		t.Fatalf("updates = %#v", revision.Updates)
	}
	if err := SaveViewUpdates(revision.Updates); err != nil {
		t.Fatal(err)
	}

	unaffected := readTestFile(t, filepath.Join(cycleDirectory, "story-02-help", "acceptance", "evidence.md"))
	for _, fragment := range []string{"spec_rev: rev-2", "- A1: observed", "- A2: observed"} {
		if !strings.Contains(unaffected, fragment) {
			t.Fatalf("unaffected evidence missing %q:\n%s", fragment, unaffected)
		}
	}
	verification, verified := BuildVerificationUpdate(
		filepath.Join(cycleDirectory, "story-02-help"),
		VerificationRequest{SpecRevision: "rev-2", VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(4 * time.Minute)},
	)
	if verified.InputError != "" || len(verified.Errors) != 0 || verification.Status != "passed" {
		t.Fatalf("verification = %#v, result = %#v", verification, verified)
	}
}

func TestRevisionMigratesEvidenceAfterAcceptanceIDsChange(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 16, 45, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	captureAcceptance(t, target, "A2", at.Add(2*time.Minute))

	storyFile := filepath.Join(target, "story.md")
	story := readTestFile(t, storyFile)
	story = strings.ReplaceAll(story, "A2", "A3")
	writeFixtureFile(t, storyFile, story)

	revision, result := BuildRevisionUpdate(
		[]string{target},
		RevisionRequest{Reason: "acceptance identity changed", At: at.Add(3 * time.Minute)},
	)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("revision result = %#v", result)
	}
	if err := SaveViewUpdates(revision.Updates); err != nil {
		t.Fatal(err)
	}
	evidence := readTestFile(t, filepath.Join(target, "acceptance", "evidence.md"))
	if !strings.Contains(evidence, "- A1: stale") || strings.Contains(evidence, "- A2:") || strings.Contains(evidence, "- A3:") {
		t.Fatalf("evidence was not migrated to current Acceptance IDs:\n%s", evidence)
	}
	if validated := ValidateSpec(cycleDirectory, false); len(validated.Errors) != 0 {
		t.Fatalf("errors = %#v", validated.Errors)
	}
}

func TestReworkIsCountedAndNewEvidenceResetsReviewState(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 17, 0, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))

	verification, result := BuildVerificationUpdate(target, VerificationRequest{SpecRevision: "rev-1",
		VerifiedBy: "reviewer", Mode: VerificationIndependent, Result: "rework", At: at.Add(2 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 || verification.ReworkCount != 1 {
		t.Fatalf("verification = %#v, result = %#v", verification, result)
	}
	if err := SaveViewUpdates(verification.Updates); err != nil {
		t.Fatal(err)
	}
	startTestRework(t, target, "rev-1", at.Add(3*time.Minute))
	captureAcceptance(t, target, "A1", at.Add(3*time.Minute))
	evidence := readTestFile(t, filepath.Join(target, "acceptance", "evidence.md"))
	for _, fragment := range []string{"verification_status: pending", "verified_by: none", "first_pass: false", "rework_count: 1"} {
		if !strings.Contains(evidence, fragment) {
			t.Fatalf("evidence missing %q:\n%s", fragment, evidence)
		}
	}
}

func TestAcceptanceRejectionReturnsReadyDeliveryToRework(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 17, 30, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	captureAcceptance(t, target, "A2", at.Add(2*time.Minute))
	verification, result := BuildVerificationUpdate(target, VerificationRequest{SpecRevision: "rev-1",
		VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(3 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("verification result = %#v", result)
	}
	if err := SaveViewUpdates(verification.Updates); err != nil {
		t.Fatal(err)
	}

	rejection, result := BuildRejectionUpdate(target, RejectionRequest{SpecRevision: "rev-1",
		RejectedBy: "product-owner", Reason: "failure state is unclear", At: at.Add(4 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 || rejection.ReworkCount != 1 {
		t.Fatalf("rejection = %#v, result = %#v", rejection, result)
	}
	if err := SaveViewUpdates(rejection.Updates); err != nil {
		t.Fatal(err)
	}
	evidence := readTestFile(t, filepath.Join(target, "acceptance", "evidence.md"))
	state := readTestFile(t, filepath.Join(cycleDirectory, "cycle.md"))
	for _, fragment := range []string{"verification_status: rework", "rework_count: 1", "- A1: observed", "- A2: observed"} {
		if !strings.Contains(evidence, fragment) {
			t.Fatalf("evidence missing %q:\n%s", fragment, evidence)
		}
	}
	for _, fragment := range []string{"Acceptance: rejected: rejected_by product-owner; failure state is unclear; acceptance/evidence.md", "Status: rejected", "Next: Await rework assignment; failure state is unclear"} {
		if !strings.Contains(state, fragment) {
			t.Fatalf("state missing %q:\n%s", fragment, state)
		}
	}

	startTestRework(t, target, "rev-1", at.Add(5*time.Minute))
	captureAcceptance(t, target, "A1", at.Add(5*time.Minute))
	state = readTestFile(t, filepath.Join(cycleDirectory, "cycle.md"))
	if !strings.Contains(state, "Acceptance: pending: acceptance/evidence.md updated for A1") {
		t.Fatalf("state did not return to pending:\n%s", state)
	}
}

func TestVerifiedAcceptanceRequiresIndependentVerification(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	storyFile := filepath.Join(target, "story.md")
	story := readTestFile(t, storyFile)
	story = strings.Replace(story, "implementation_brief: none\n", `implementation_brief: none
acceptance_policy: verified
acceptance_authority: product-rule
acceptance_scope: all-current-acceptance
acceptance_review: independent-full-coverage
`, 1)
	writeFixtureFile(t, storyFile, story)
	at := time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	captureAcceptance(t, target, "A2", at.Add(2*time.Minute))
	verification, result := BuildVerificationUpdate(target, VerificationRequest{SpecRevision: "rev-1",
		VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(3 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("verification result = %#v", result)
	}
	if err := SaveViewUpdates(verification.Updates); err != nil {
		t.Fatal(err)
	}

	_, result = BuildAcceptanceCommit(target, AcceptanceRequest{SpecRevision: "rev-1",
		AcceptedBy: "executor-a", Method: AcceptanceVerified, At: at.Add(4 * time.Minute),
	})
	if !strings.Contains(result.InputError, "independent verification") {
		t.Fatalf("acceptance result = %#v", result)
	}
}

func TestSaveAcceptanceCommitRestoresEvidenceWhenStateChanged(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 19, 0, 0, 0, time.UTC)
	checkpointStoryDone(t, target, at)
	captureAcceptance(t, target, "A1", at.Add(time.Minute))
	captureAcceptance(t, target, "A2", at.Add(2*time.Minute))
	verification, result := BuildVerificationUpdate(target, VerificationRequest{SpecRevision: "rev-1",
		VerifiedBy: "executor-a", Mode: VerificationSelf, Result: "pass", At: at.Add(3 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("verification result = %#v", result)
	}
	if err := SaveViewUpdates(verification.Updates); err != nil {
		t.Fatal(err)
	}
	commit, result := BuildAcceptanceCommit(target, AcceptanceRequest{SpecRevision: "rev-1",
		AcceptedBy: "product-owner", Method: AcceptanceHuman, At: at.Add(4 * time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("acceptance result = %#v", result)
	}
	stateFile := filepath.Join(cycleDirectory, "cycle.md")
	state := readTestFile(t, stateFile)
	writeFixtureFile(t, stateFile, strings.Replace(state, "status: active", "status: unknown", 1))

	if err := SaveAcceptanceCommit(commit); err == nil {
		t.Fatal("expected stale state failure")
	}
	if _, err := os.Stat(filepath.Join(target, "acceptance", "outcome.md")); !os.IsNotExist(err) {
		t.Fatalf("outcome.md exists after rollback: %v", err)
	}
	if evidence := readTestFile(t, filepath.Join(target, "acceptance", "evidence.md")); evidence != commit.EvidenceBefore {
		t.Fatal("evidence was not restored")
	}
}

func checkpointStoryDone(t *testing.T, target string, at time.Time) {
	t.Helper()
	checkpoint, result := BuildCheckpointUpdate(target, CheckpointRequest{SpecRevision: "rev-1",
		Owner: "executor-a", Status: "active", Complete: true, Next: "none", Situation: "implementation complete", At: at,
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("checkpoint result = %#v", result)
	}
	if err := SaveViewUpdates([]ViewUpdate{checkpoint.Update}); err != nil {
		t.Fatal(err)
	}
}

func captureAcceptance(t *testing.T, target, acceptanceID string, at time.Time) {
	t.Helper()
	snapshot, validation := LoadCycleSnapshot(filepath.Dir(target))
	if validation.InputError != "" || len(validation.Errors) > 0 {
		t.Fatalf("prepare capture: %#v", validation)
	}
	capture, result := BuildEvidenceUpdate(target, EvidenceCaptureRequest{SpecRevision: snapshot.SpecRevision,
		ImplementedBy: "executor-a", AcceptanceID: acceptanceID,
		Observed: "observed " + acceptanceID, Command: "go test ./...", Result: "passed " + acceptanceID,
		Artifacts: "artifacts/" + strings.ToLower(acceptanceID) + ".txt",
		Context:   testObservationContext(snapshot.SpecRevision, at), At: at,
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("capture result = %#v", result)
	}
	if err := SaveViewUpdates(capture.Updates); err != nil {
		t.Fatal(err)
	}
}

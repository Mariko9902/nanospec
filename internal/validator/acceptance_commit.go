package validator

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"nanospec/internal/diagnostic"
)

// AcceptanceCommit holds the evidence and state changes needed to freeze a delivery.
type AcceptanceCommit struct {
	CycleID        string
	StoryID        string
	StoryFile      string
	StoryDigest    string
	EvidenceFile   string
	OutcomeFile    string
	EvidenceBefore string
	OutcomeContent string
	StateUpdate    ViewUpdate
}

// BuildAcceptanceCommit checks acceptance conditions and prepares the frozen result.
func BuildAcceptanceCommit(target string, request AcceptanceRequest) (AcceptanceCommit, ValidationResult) {
	if inputError := validateAcceptanceRequest(request); inputError != "" {
		return AcceptanceCommit{}, ValidationResult{InputError: inputError}
	}
	storyDirectory, inputError := resolveStoryDirectory(target)
	if inputError != "" {
		return AcceptanceCommit{}, ValidationResult{InputError: inputError}
	}
	snapshot, result := LoadCycleSnapshot(filepath.Dir(storyDirectory))
	if result.InputError != "" || len(result.Errors) > 0 {
		return AcceptanceCommit{}, result
	}
	storyID := filepath.Base(storyDirectory)
	story, found := snapshot.Stories[storyID]
	if !found {
		return AcceptanceCommit{}, ValidationResult{InputError: "Story is not part of the Cycle: " + storyID}
	}
	if story.HasOutcome || acceptanceState(story.ID, snapshot.Acceptance) == "accepted" {
		return AcceptanceCommit{}, ValidationResult{InputError: "accepted Story is frozen: " + story.ID}
	}
	if acceptanceState(story.ID, snapshot.Acceptance) != "ready" {
		return AcceptanceCommit{}, ValidationResult{InputError: "acceptance requires ready verification state"}
	}
	if inputError := validateExecutionGate(story, snapshot); inputError != "" {
		return AcceptanceCommit{}, ValidationResult{InputError: inputError}
	}
	evidenceFile := filepath.Join(storyDirectory, "acceptance", "evidence.md")
	record, evidenceBefore, result := loadEvidenceRecord(evidenceFile, story)
	if result.InputError != "" || len(result.Errors) > 0 {
		return AcceptanceCommit{}, result
	}
	if record.SpecRevision != snapshot.SpecRevision || record.VerificationStatus != "passed" {
		return AcceptanceCommit{}, ValidationResult{InputError: "acceptance requires passed evidence for the current specification revision"}
	}
	lane, found := storyLane(story.ID, snapshot.ActiveLanes)
	if !found || lane.Fields["Status"] != "done" || record.SpecChangeCount != laneSpecChanges(lane) {
		return AcceptanceCommit{}, ValidationResult{InputError: "acceptance requires a matching done checkpoint"}
	}
	if request.Method == AcceptanceVerified {
		if story.AcceptancePolicy != AcceptanceVerified {
			return AcceptanceCommit{}, ValidationResult{InputError: "verified acceptance requires Story acceptance_policy: verified"}
		}
		if record.VerificationMode != string(VerificationIndependent) {
			return AcceptanceCommit{}, ValidationResult{InputError: "verified acceptance requires independent verification"}
		}
	}
	outcomeFile := filepath.Join(storyDirectory, "acceptance", "outcome.md")
	if fileExists(outcomeFile) {
		return AcceptanceCommit{}, ValidationResult{InputError: "outcome.md already exists"}
	}
	storySource, err := os.ReadFile(story.File)
	if err != nil {
		return AcceptanceCommit{}, ValidationResult{InputError: "cannot read story.md: " + err.Error()}
	}
	storyDigest := storyContentDigest(storySource)
	outcome := renderOutcomeRecord(record, story.AcceptanceIDs, request, storyDigest)
	var outcomeReport diagnostic.Report
	validateOutcomeSource(outcomeFile, outcome, story, &outcomeReport)
	if errors := outcomeReport.Errors(); len(errors) > 0 {
		return AcceptanceCommit{}, ValidationResult{Errors: errors, Warnings: outcomeReport.Warnings()}
	}

	state, err := updateAcceptanceState(snapshot.StateText, story.ID, "accepted: all Acceptance confirmed; acceptance/outcome.md; accepted_by "+request.AcceptedBy+" ("+string(request.Method)+")")
	if err == nil {
		state, err = updateManagedStateLine(state, "Current Situation", story.ID, "")
	}
	if err == nil {
		state, err = updateManagedStateLine(state, "Open Issues", story.ID, "")
	}
	if err == nil {
		state, err = removeStoryLane(state, story.ID)
	}
	if err == nil {
		state, err = replaceFrontmatterScalar(state, "last_updated_at", request.At.Format(time.RFC3339))
	}
	if err != nil {
		return AcceptanceCommit{}, ValidationResult{InputError: err.Error()}
	}
	stories := cloneStories(snapshot.Stories)
	acceptedStory := stories[story.ID]
	acceptedStory.HasOutcome = true
	stories[story.ID] = acceptedStory
	views, err := RenderCycleViews(snapshot.CycleText, state, stories)
	if err != nil {
		return AcceptanceCommit{}, ValidationResult{InputError: err.Error()}
	}
	state = views.State
	if allStoriesAccepted(state, stories) {
		state, err = replaceFrontmatterScalar(state, "status", "done")
		if err != nil {
			return AcceptanceCommit{}, ValidationResult{InputError: err.Error()}
		}
	}
	stateFile := filepath.Join(snapshot.Directory, "execution-state.md")
	if validation := validateUpdatedStateWithStories(stateFile, snapshot.ID, stories, state); validation.InputError != "" || len(validation.Errors) > 0 {
		return AcceptanceCommit{}, validation
	}
	return AcceptanceCommit{
		CycleID: snapshot.ID, StoryID: story.ID, EvidenceFile: evidenceFile, OutcomeFile: outcomeFile,
		StoryFile: story.File, StoryDigest: storyDigest,
		EvidenceBefore: evidenceBefore, OutcomeContent: outcome,
		StateUpdate: ViewUpdate{File: stateFile, Before: snapshot.StateText, After: state},
	}, ValidationResult{}
}

func storyContentDigest(source []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(source))
}

// SaveAcceptanceCommit freezes evidence and updates state, restoring the originals if saving fails.
func SaveAcceptanceCommit(commit AcceptanceCommit) error {
	locks, err := lockViewUpdates([]ViewUpdate{commit.StateUpdate})
	if err != nil {
		return err
	}
	defer locks.release()
	return saveAcceptanceCommit(commit)
}

func saveAcceptanceCommit(commit AcceptanceCommit) error {
	if err := validateAcceptanceStory(commit); err != nil {
		return err
	}
	if fileExists(commit.OutcomeFile) {
		return fmt.Errorf("outcome.md already exists: %s", commit.OutcomeFile)
	}
	if err := saveViewUpdates([]ViewUpdate{{File: commit.EvidenceFile, Before: commit.EvidenceBefore, After: commit.OutcomeContent}}, os.Rename); err != nil {
		return err
	}
	if err := os.Rename(commit.EvidenceFile, commit.OutcomeFile); err != nil {
		restoreErr := saveViewUpdates([]ViewUpdate{{File: commit.EvidenceFile, Before: commit.OutcomeContent, After: commit.EvidenceBefore}}, os.Rename)
		if restoreErr != nil {
			return fmt.Errorf("freeze evidence: %v; restore evidence: %v", err, restoreErr)
		}
		return err
	}
	err := validateAcceptanceStory(commit)
	if err == nil {
		err = saveViewUpdates([]ViewUpdate{commit.StateUpdate}, os.Rename)
	}
	if err != nil {
		renameErr := os.Rename(commit.OutcomeFile, commit.EvidenceFile)
		restoreErr := error(nil)
		if renameErr == nil {
			restoreErr = saveViewUpdates([]ViewUpdate{{File: commit.EvidenceFile, Before: commit.OutcomeContent, After: commit.EvidenceBefore}}, os.Rename)
		}
		if renameErr != nil || restoreErr != nil {
			return fmt.Errorf("save acceptance state: %v; restore rename: %v; restore evidence: %v", err, renameErr, restoreErr)
		}
		return err
	}
	return nil
}

func validateAcceptanceStory(commit AcceptanceCommit) error {
	source, err := os.ReadFile(commit.StoryFile)
	if err != nil {
		return err
	}
	if storyContentDigest(source) != commit.StoryDigest {
		return fmt.Errorf("story.md changed during acceptance: %s", commit.StoryFile)
	}
	return nil
}

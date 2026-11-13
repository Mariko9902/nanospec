package validator

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"nanospec/internal/specdoc"
)

// CheckpointRequest describes the current execution and recovery facts.
type CheckpointRequest struct {
	SpecRevision string
	Owner        string
	Status       string
	Complete     bool
	Next         string
	Situation    string
	Issue        string
	At           time.Time
}

// CheckpointResult holds a proposed checkpoint change.
type CheckpointResult struct {
	CycleID      string
	StoryID      string
	LaneID       string
	Status       string
	SpecRevision string
	Update       ViewUpdate
}

// BuildCheckpointUpdate builds and validates a checkpoint without saving it.
func BuildCheckpointUpdate(target string, request CheckpointRequest) (CheckpointResult, ValidationResult) {
	if inputError := validateCheckpointRequest(request); inputError != "" {
		return CheckpointResult{}, ValidationResult{InputError: inputError}
	}
	storyDirectory, inputError := resolveStoryDirectory(target)
	if inputError != "" {
		return CheckpointResult{}, ValidationResult{InputError: inputError}
	}
	snapshot, result := LoadCycleSnapshot(filepath.Dir(storyDirectory))
	if result.InputError != "" || len(result.Errors) > 0 {
		return CheckpointResult{}, result
	}
	if inputError := validatePreparedRevision(request.SpecRevision, snapshot.SpecRevision); inputError != "" {
		return CheckpointResult{}, ValidationResult{InputError: inputError}
	}
	storyID := filepath.Base(storyDirectory)
	story, found := snapshot.Stories[storyID]
	if !found {
		return CheckpointResult{}, ValidationResult{InputError: "Story is not part of the Cycle: " + storyID}
	}
	current := storyStatus(storyID, snapshot.Acceptance, snapshot.ActiveLanes)
	previous, _ := storyLane(storyID, snapshot.ActiveLanes)
	if inputError := validateCheckpointTransition(current, StoryStatus(request.Status), StoryStatus(previous.Fields["Resume Status"])); inputError != "" {
		return CheckpointResult{}, ValidationResult{InputError: inputError}
	}
	if request.Status == "ready" {
		if validation := validateReadyCheckpoint(story, snapshot, previous); validation.InputError != "" || len(validation.Errors) > 0 {
			return CheckpointResult{}, validation
		}
	}
	if checkpointNeedsGate(StoryStatus(request.Status)) {
		if inputError := validateExecutionGate(story, snapshot); inputError != "" {
			return CheckpointResult{}, ValidationResult{InputError: inputError}
		}
	}

	lanes := cloneFieldBlocks(snapshot.ActiveLanes)
	laneIndex := -1
	for index, lane := range lanes {
		if lane.Fields["Story"] == storyID {
			if laneIndex >= 0 {
				return CheckpointResult{}, ValidationResult{InputError: "Story has more than one active Lane: " + storyID}
			}
			laneIndex = index
		}
	}
	if laneIndex < 0 {
		lanes = append(lanes, FieldBlock{ID: nextLaneID(lanes), Fields: map[string]string{"Story": storyID, "Spec Changes": "0"}})
		laneIndex = len(lanes) - 1
	}
	lane := &lanes[laneIndex]
	if owner := lane.Fields["Owner"]; owner != "" && owner != request.Owner && fileExists(filepath.Join(storyDirectory, "acceptance", "evidence.md")) {
		return CheckpointResult{}, ValidationResult{InputError: "cannot change Owner after evidence is recorded; continue with Owner " + owner}
	}
	lane.Fields["Owner"] = request.Owner
	lane.Fields["Status"] = request.Status
	lane.Fields["Next"] = request.Next
	if request.Status == "blocked" {
		if current != StoryBlocked {
			lane.Fields["Resume Status"] = string(current)
		}
	} else {
		delete(lane.Fields, "Resume Status")
		if request.Status != "ready" {
			lane.Fields["Implementation Complete"] = strconv.FormatBool(request.Complete)
		}
	}
	if lane.Fields["Implementation Complete"] == "" {
		lane.Fields["Implementation Complete"] = "false"
	}

	if lane.Fields["Spec Changes"] == "" {
		lane.Fields["Spec Changes"] = "0"
	}

	updated, err := specdoc.ParseMarkdown(snapshot.CycleText).ReplaceSection(2, "Active Lanes", renderLanes(lanes))
	if err != nil {
		return CheckpointResult{}, ValidationResult{InputError: err.Error()}
	}
	updated, err = updateManagedStateLine(updated, "Current Situation", storyID, request.Situation)
	if err != nil {
		return CheckpointResult{}, ValidationResult{InputError: err.Error()}
	}
	updated, err = updateManagedStateLine(updated, "Open Issues", storyID, request.Issue)
	if err != nil {
		return CheckpointResult{}, ValidationResult{InputError: err.Error()}
	}
	updated, err = specdoc.ReplaceFrontmatterScalar(updated, "last_updated_at", request.At.Format(time.RFC3339))
	if err != nil {
		return CheckpointResult{}, ValidationResult{InputError: err.Error()}
	}
	stateFile := filepath.Join(snapshot.Directory, "cycle.md")
	var validation ValidationResult
	updated, validation = canonicalizeCycleState(stateFile, snapshot.ID, snapshot.Stories, updated)
	if validation.InputError != "" || len(validation.Errors) > 0 {
		return CheckpointResult{}, validation
	}
	return CheckpointResult{
		CycleID:      snapshot.ID,
		StoryID:      storyID,
		LaneID:       lane.ID,
		Status:       request.Status,
		SpecRevision: snapshot.SpecRevision,
		Update:       ViewUpdate{File: stateFile, Before: snapshot.CycleText, After: updated},
	}, ValidationResult{}
}

func validateExecutionGate(story Story, snapshot CycleSnapshot) string {
	storySource, err := os.ReadFile(story.File)
	if err != nil {
		return "cannot read story.md: " + err.Error()
	}
	if hasOpenQuestions(string(storySource)) {
		return "cannot start or continue a Story with open questions: " + story.ID
	}
	for _, dependency := range dependencyStatuses(story, snapshot) {
		if dependency.Status != StoryReady && dependency.Status != StoryAccepted {
			return "cannot start or continue before dependency is ready: " + dependency.StoryID
		}
	}
	return ""
}

func validateCheckpointRequest(request CheckpointRequest) string {
	if !executorIDPattern.MatchString(request.Owner) {
		return "owner must be a stable identifier"
	}
	if !singleLineText(request.Owner) || !singleLineText(request.Next) || !singleLineText(request.Situation) {
		return "owner, next and situation must each be one non-empty line"
	}
	if !validCheckpointStatus(StoryStatus(request.Status)) {
		return "checkpoint status must be pending, active, ready, rejected, rework or blocked"
	}
	if request.Status == "blocked" && !singleLineText(request.Issue) {
		return "blocked checkpoint requires one issue line"
	}
	if request.Issue != "" && !singleLineText(request.Issue) {
		return "checkpoint issue must be one line"
	}
	if request.Complete && request.Status != "active" && request.Status != "rework" {
		return "complete checkpoint requires active or rework status"
	}
	if request.Complete && request.Next != "none" {
		return "complete checkpoint requires next=none"
	}
	if request.Status == "blocked" && request.Next == "none" {
		return "blocked checkpoint requires a recovery action"
	}
	if request.At.IsZero() {
		return "checkpoint timestamp is required"
	}
	return ""
}

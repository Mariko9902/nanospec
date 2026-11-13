package validator

import (
	"fmt"
	"path/filepath"
)

// StoryStatus identifies the current delivery stage.
type StoryStatus string

const (
	StoryPending  StoryStatus = "pending"
	StoryActive   StoryStatus = "active"
	StoryReady    StoryStatus = "ready"
	StoryRejected StoryStatus = "rejected"
	StoryRework   StoryStatus = "rework"
	StoryBlocked  StoryStatus = "blocked"
	StoryAccepted StoryStatus = "accepted"
)

func storyStatus(id string, acceptance, lanes []FieldBlock) StoryStatus {
	state := StoryStatus(acceptanceState(id, acceptance))
	if state == StoryAccepted {
		return state
	}
	if lane, found := storyLane(id, lanes); found {
		return StoryStatus(lane.Fields["Status"])
	}
	return state
}

func implementationComplete(lane FieldBlock) bool {
	return lane.Fields["Implementation Complete"] == "true"
}

func checkpointTargets(current StoryStatus) []StoryStatus {
	switch current {
	case StoryPending:
		return []StoryStatus{StoryPending, StoryActive, StoryBlocked}
	case StoryActive:
		return []StoryStatus{StoryActive, StoryBlocked}
	case StoryReady:
		return []StoryStatus{StoryReady, StoryBlocked}
	case StoryRejected:
		return []StoryStatus{StoryRejected, StoryRework, StoryBlocked}
	case StoryRework:
		return []StoryStatus{StoryRework, StoryBlocked}
	}
	return nil
}

func validCheckpointStatus(status StoryStatus) bool {
	return status == StoryBlocked || len(checkpointTargets(status)) > 0
}

func validateCheckpointTransition(current, target, resume StoryStatus) string {
	if current == StoryAccepted {
		return "accepted Story is frozen"
	}
	if current == StoryBlocked {
		if target == StoryBlocked {
			return ""
		}
		current = resume
	}
	for _, allowed := range checkpointTargets(current) {
		if target == allowed {
			return ""
		}
	}
	return fmt.Sprintf("cannot checkpoint %s as %s", current, target)
}

func checkpointNeedsGate(status StoryStatus) bool {
	return status == StoryActive || status == StoryRework || status == StoryReady
}

func validateStoryStage(current StoryStatus, allowed ...StoryStatus) string {
	for _, status := range allowed {
		if current == status {
			return ""
		}
	}
	return fmt.Sprintf("cannot perform this operation while Story status is %s; record the appropriate checkpoint first", current)
}

func storyResumeStatus(id string, lanes []FieldBlock) *StoryStatus {
	lane, found := storyLane(id, lanes)
	if !found || lane.Fields["Status"] != "blocked" {
		return nil
	}
	status := StoryStatus(lane.Fields["Resume Status"])
	return &status
}

func deriveCycleStatus(acceptance, lanes []FieldBlock) string {
	allAccepted, allBlocked := true, true
	for _, entry := range acceptance {
		status := storyStatus(entry.ID, acceptance, lanes)
		if status == StoryAccepted {
			continue
		}
		allAccepted = false
		if status != StoryBlocked {
			allBlocked = false
		}
	}
	if allAccepted {
		return "done"
	}
	if allBlocked {
		return "blocked"
	}
	return "active"
}

func validateReadyCheckpoint(story Story, snapshot CycleSnapshot, lane FieldBlock) ValidationResult {
	if acceptanceState(story.ID, snapshot.Acceptance) != "ready" || !implementationComplete(lane) {
		return ValidationResult{InputError: "ready checkpoint requires previously passed verification"}
	}
	record, _, result := loadEvidenceRecord(filepath.Join(story.Directory, "acceptance", "evidence.md"), story)
	if result.InputError != "" || len(result.Errors) > 0 {
		return result
	}
	if record.VerificationStatus != "passed" || record.SpecRevision != snapshot.SpecRevision || record.SpecChangeCount != laneSpecChanges(lane) {
		return ValidationResult{InputError: "ready checkpoint requires current passed evidence"}
	}
	return ValidationResult{}
}

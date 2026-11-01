package validator

import (
	"path/filepath"
	"strconv"
	"time"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

// EvidenceCaptureRequest describes an observed Acceptance result.
type EvidenceCaptureRequest struct {
	ImplementedBy string
	AcceptanceID  string
	Observed      string
	Command       string
	Result        string
	Artifacts     string
	Context       string
	At            time.Time
}

// EvidenceCaptureResult holds the file changes for recording evidence.
type EvidenceCaptureResult struct {
	CycleID      string
	StoryID      string
	AcceptanceID string
	Updates      []ViewUpdate
}

// VerificationRequest describes a complete evidence review.
type VerificationRequest struct {
	VerifiedBy string
	Mode       VerificationMode
	Result     string
	At         time.Time
}

// VerificationResult holds the changes produced by a review.
type VerificationResult struct {
	CycleID     string
	StoryID     string
	Status      string
	ReworkCount int
	Updates     []ViewUpdate
}

// RejectionRequest records an explicit decision to reject a delivery.
type RejectionRequest struct {
	RejectedBy string
	Reason     string
	At         time.Time
}

// RejectionResult holds the changes needed to return a delivery for rework.
type RejectionResult struct {
	CycleID     string
	StoryID     string
	ReworkCount int
	Updates     []ViewUpdate
}

// AcceptanceRequest describes an acceptance decision that has already been authorized.
type AcceptanceRequest struct {
	AcceptedBy string
	Method     AcceptancePolicy
	At         time.Time
}

type evidenceRecord struct {
	StoryID            string
	SpecRevision       string
	ImplementedBy      string
	VerificationStatus string
	VerifiedBy         string
	VerificationMode   string
	VerifiedAt         string
	VerifiedAcceptance []string
	ReworkCount        int
	SpecChangeCount    int
	Context            string
	Results            map[string]acceptanceResult
}

// BuildEvidenceUpdate records one observation and updates the acceptance state.
func BuildEvidenceUpdate(target string, request EvidenceCaptureRequest) (EvidenceCaptureResult, ValidationResult) {
	if inputError := validateEvidenceCaptureRequest(request); inputError != "" {
		return EvidenceCaptureResult{}, ValidationResult{InputError: inputError}
	}
	storyDirectory, snapshot, story, lane, result := loadActiveStory(target)
	if result.InputError != "" || len(result.Errors) > 0 {
		return EvidenceCaptureResult{}, result
	}
	if story.HasOutcome {
		return EvidenceCaptureResult{}, ValidationResult{InputError: "accepted Story is frozen: " + story.ID}
	}
	if lane.Fields["Owner"] != request.ImplementedBy {
		return EvidenceCaptureResult{}, ValidationResult{InputError: "implemented-by must match the active Lane owner"}
	}
	if !containsString(story.AcceptanceIDs, request.AcceptanceID) {
		return EvidenceCaptureResult{}, ValidationResult{InputError: "unknown Acceptance ID: " + request.AcceptanceID}
	}

	evidenceFile := filepath.Join(storyDirectory, "acceptance", "evidence.md")
	record := newEvidenceRecord(story, snapshot.SpecRevision, request, laneSpecChanges(lane))
	before := ""
	if fileExists(evidenceFile) {
		loaded, source, loadedResult := loadEvidenceRecord(evidenceFile, story)
		if loadedResult.InputError != "" || len(loadedResult.Errors) > 0 {
			return EvidenceCaptureResult{}, loadedResult
		}
		if loaded.SpecRevision != snapshot.SpecRevision {
			return EvidenceCaptureResult{}, ValidationResult{InputError: "evidence revision does not match current Cycle revision"}
		}
		if loaded.ImplementedBy != request.ImplementedBy {
			return EvidenceCaptureResult{}, ValidationResult{InputError: "implemented-by does not match existing evidence"}
		}
		record = loaded
		before = source
		record.SpecChangeCount = laneSpecChanges(lane)
	}
	record.VerificationStatus = "pending"
	record.VerifiedBy = "none"
	record.VerificationMode = "none"
	record.VerifiedAt = "none"
	record.VerifiedAcceptance = []string{}
	for id, item := range record.Results {
		if item.status == "confirmed" {
			item.status = "observed"
			record.Results[id] = item
		}
	}
	record.Results[request.AcceptanceID] = acceptanceResult{
		id: request.AcceptanceID, status: "observed", observed: request.Observed,
		evidenceCommand: request.Command, evidenceResult: request.Result, evidenceArtifacts: request.Artifacts, evidenceContext: request.Context,
	}
	after := renderEvidenceRecord(record, story.AcceptanceIDs)
	if validation := validateEvidenceSource(evidenceFile, story, after); validation.InputError != "" || len(validation.Errors) > 0 {
		return EvidenceCaptureResult{}, validation
	}

	state, err := updateAcceptanceState(snapshot.StateText, story.ID, "pending: acceptance/evidence.md updated for "+request.AcceptanceID+" at "+snapshot.SpecRevision)
	if err == nil {
		state, err = updateManagedStateLine(state, "Current Situation", story.ID, "Evidence recorded for "+request.AcceptanceID)
	}
	if err == nil {
		state, err = replaceFrontmatterScalar(state, "last_updated_at", request.At.Format(time.RFC3339))
	}
	if err != nil {
		return EvidenceCaptureResult{}, ValidationResult{InputError: err.Error()}
	}
	stateFile := filepath.Join(snapshot.Directory, "execution-state.md")
	if validation := validateUpdatedState(stateFile, snapshot, state); validation.InputError != "" || len(validation.Errors) > 0 {
		return EvidenceCaptureResult{}, validation
	}
	return EvidenceCaptureResult{
		CycleID: snapshot.ID, StoryID: story.ID, AcceptanceID: request.AcceptanceID,
		Updates: []ViewUpdate{{File: evidenceFile, Before: before, After: after}, {File: stateFile, Before: snapshot.StateText, After: state}},
	}, ValidationResult{}
}

// BuildVerificationUpdate records a passed review or a return for rework.
func BuildVerificationUpdate(target string, request VerificationRequest) (VerificationResult, ValidationResult) {
	if inputError := validateVerificationRequest(request); inputError != "" {
		return VerificationResult{}, ValidationResult{InputError: inputError}
	}
	storyDirectory, snapshot, story, lane, result := loadActiveStory(target)
	if result.InputError != "" || len(result.Errors) > 0 {
		return VerificationResult{}, result
	}
	if request.Result == "pass" {
		if inputError := validateExecutionGate(story, snapshot); inputError != "" {
			return VerificationResult{}, ValidationResult{InputError: inputError}
		}
	}
	evidenceFile := filepath.Join(storyDirectory, "acceptance", "evidence.md")
	if !fileExists(evidenceFile) {
		return VerificationResult{}, ValidationResult{InputError: "verification requires evidence.md"}
	}
	record, before, result := loadEvidenceRecord(evidenceFile, story)
	if result.InputError != "" || len(result.Errors) > 0 {
		return VerificationResult{}, result
	}
	if record.SpecRevision != snapshot.SpecRevision || record.SpecChangeCount != laneSpecChanges(lane) {
		return VerificationResult{}, ValidationResult{InputError: "evidence does not match current specification facts"}
	}
	if request.Mode == VerificationSelf && record.ImplementedBy != request.VerifiedBy {
		return VerificationResult{}, ValidationResult{InputError: "self verification requires verifier to match implementer"}
	}
	if request.Mode == VerificationIndependent && record.ImplementedBy == request.VerifiedBy {
		return VerificationResult{}, ValidationResult{InputError: "independent verification requires a different verifier"}
	}
	if request.Result == "pass" && lane.Fields["Status"] != "done" {
		return VerificationResult{}, ValidationResult{InputError: "passing verification requires a done checkpoint"}
	}

	record.VerifiedBy = request.VerifiedBy
	record.VerificationMode = string(request.Mode)
	record.VerifiedAt = request.At.Format(time.RFC3339)
	stateValue := "pending: verification requires rework; acceptance/evidence.md"
	if request.Result == "pass" {
		for _, id := range story.AcceptanceIDs {
			item, found := record.Results[id]
			if !found || item.status == "stale" {
				return VerificationResult{}, ValidationResult{InputError: "verification requires current evidence for " + id}
			}
			item.status = "confirmed"
			record.Results[id] = item
		}
		record.VerificationStatus = "passed"
		record.VerifiedAcceptance = append([]string(nil), story.AcceptanceIDs...)
		stateValue = "ready: all Acceptance verified by " + request.VerifiedBy + " (" + string(request.Mode) + "); acceptance/evidence.md"
	} else {
		record.VerificationStatus = "rework"
		record.VerifiedAcceptance = existingEvidenceIDs(record, story.AcceptanceIDs)
		record.ReworkCount++
		for id, item := range record.Results {
			if item.status == "confirmed" {
				item.status = "observed"
				record.Results[id] = item
			}
		}
	}
	after := renderEvidenceRecord(record, story.AcceptanceIDs)
	if validation := validateEvidenceSource(evidenceFile, story, after); validation.InputError != "" || len(validation.Errors) > 0 {
		return VerificationResult{}, validation
	}
	state, err := updateAcceptanceState(snapshot.StateText, story.ID, stateValue)
	if err == nil {
		state, err = updateManagedStateLine(state, "Current Situation", story.ID, "Verification result: "+request.Result)
	}
	if err == nil {
		state, err = replaceFrontmatterScalar(state, "last_updated_at", request.At.Format(time.RFC3339))
	}
	if err != nil {
		return VerificationResult{}, ValidationResult{InputError: err.Error()}
	}
	stateFile := filepath.Join(snapshot.Directory, "execution-state.md")
	if validation := validateUpdatedState(stateFile, snapshot, state); validation.InputError != "" || len(validation.Errors) > 0 {
		return VerificationResult{}, validation
	}
	return VerificationResult{
		CycleID: snapshot.ID, StoryID: story.ID, Status: record.VerificationStatus, ReworkCount: record.ReworkCount,
		Updates: []ViewUpdate{{File: evidenceFile, Before: before, After: after}, {File: stateFile, Before: snapshot.StateText, After: state}},
	}, ValidationResult{}
}

// BuildRejectionUpdate returns reviewed evidence for rework after rejection.
func BuildRejectionUpdate(target string, request RejectionRequest) (RejectionResult, ValidationResult) {
	if inputError := validateRejectionRequest(request); inputError != "" {
		return RejectionResult{}, ValidationResult{InputError: inputError}
	}
	storyDirectory, snapshot, story, lane, result := loadActiveStory(target)
	if result.InputError != "" || len(result.Errors) > 0 {
		return RejectionResult{}, result
	}
	if acceptanceState(story.ID, snapshot.Acceptance) != "ready" {
		return RejectionResult{}, ValidationResult{InputError: "rejection requires ready verification state"}
	}
	if story.HasOutcome {
		return RejectionResult{}, ValidationResult{InputError: "accepted Story is frozen: " + story.ID}
	}
	evidenceFile := filepath.Join(storyDirectory, "acceptance", "evidence.md")
	record, evidenceBefore, result := loadEvidenceRecord(evidenceFile, story)
	if result.InputError != "" || len(result.Errors) > 0 {
		return RejectionResult{}, result
	}
	if record.SpecRevision != snapshot.SpecRevision || record.VerificationStatus != "passed" {
		return RejectionResult{}, ValidationResult{InputError: "rejection requires passed evidence for the current specification revision"}
	}
	if lane.Fields["Status"] != "done" || record.SpecChangeCount != laneSpecChanges(lane) {
		return RejectionResult{}, ValidationResult{InputError: "rejection requires a matching done checkpoint"}
	}

	record.VerificationStatus = "rework"
	record.ReworkCount++
	for id, item := range record.Results {
		if item.status == "confirmed" {
			item.status = "observed"
			record.Results[id] = item
		}
	}
	evidenceAfter := renderEvidenceRecord(record, story.AcceptanceIDs)
	if validation := validateEvidenceSource(evidenceFile, story, evidenceAfter); validation.InputError != "" || len(validation.Errors) > 0 {
		return RejectionResult{}, validation
	}

	lanes := cloneFieldBlocks(snapshot.ActiveLanes)
	for index := range lanes {
		if lanes[index].Fields["Story"] == story.ID {
			lanes[index].Fields["Status"] = "paused"
			lanes[index].Fields["Next"] = "Address rejection: " + request.Reason
		}
	}
	state, err := specdoc.ParseMarkdown(snapshot.StateText).ReplaceSection(2, "Active Lanes", renderLanes(lanes))
	if err == nil {
		state, err = updateAcceptanceState(state, story.ID, "rejected: rejected_by "+request.RejectedBy+"; "+request.Reason+"; acceptance/evidence.md")
	}
	if err == nil {
		state, err = updateManagedStateLine(state, "Current Situation", story.ID, "Acceptance rejected; "+request.Reason)
	}
	if err == nil {
		state, err = updateManagedStateLine(state, "Open Issues", story.ID, request.Reason)
	}
	if err == nil {
		state, err = replaceFrontmatterScalar(state, "last_updated_at", request.At.Format(time.RFC3339))
	}
	if err != nil {
		return RejectionResult{}, ValidationResult{InputError: err.Error()}
	}
	stateFile := filepath.Join(snapshot.Directory, "execution-state.md")
	if validation := validateUpdatedState(stateFile, snapshot, state); validation.InputError != "" || len(validation.Errors) > 0 {
		return RejectionResult{}, validation
	}
	return RejectionResult{
		CycleID: snapshot.ID, StoryID: story.ID, ReworkCount: record.ReworkCount,
		Updates: []ViewUpdate{{File: evidenceFile, Before: evidenceBefore, After: evidenceAfter}, {File: stateFile, Before: snapshot.StateText, After: state}},
	}, ValidationResult{}
}

func loadActiveStory(target string) (string, CycleSnapshot, Story, FieldBlock, ValidationResult) {
	storyDirectory, inputError := resolveStoryDirectory(target)
	if inputError != "" {
		return "", CycleSnapshot{}, Story{}, FieldBlock{}, ValidationResult{InputError: inputError}
	}
	snapshot, result := LoadCycleSnapshot(filepath.Dir(storyDirectory))
	if result.InputError != "" || len(result.Errors) > 0 {
		return "", CycleSnapshot{}, Story{}, FieldBlock{}, result
	}
	storyID := filepath.Base(storyDirectory)
	story, found := snapshot.Stories[storyID]
	if !found {
		return "", CycleSnapshot{}, Story{}, FieldBlock{}, ValidationResult{InputError: "Story is not part of the Cycle: " + storyID}
	}
	lane, found := storyLane(storyID, snapshot.ActiveLanes)
	if !found {
		return "", CycleSnapshot{}, Story{}, FieldBlock{}, ValidationResult{InputError: "Story requires a checkpoint before recording evidence"}
	}
	return storyDirectory, snapshot, story, lane, ValidationResult{}
}

func storyLane(storyID string, lanes []FieldBlock) (FieldBlock, bool) {
	for _, lane := range lanes {
		if lane.Fields["Story"] == storyID {
			return lane, true
		}
	}
	return FieldBlock{}, false
}

func laneSpecChanges(lane FieldBlock) int {
	value, _ := strconv.Atoi(lane.Fields["Spec Changes"])
	return value
}

func validateEvidenceCaptureRequest(request EvidenceCaptureRequest) string {
	if !executorIDPattern.MatchString(request.ImplementedBy) {
		return "implemented-by must be a stable identifier"
	}
	if !canonicalAcceptanceID.MatchString(request.AcceptanceID) {
		return "acceptance must be a stable Acceptance ID"
	}
	for _, field := range []struct{ name, value string }{
		{"observed", request.Observed}, {"command", request.Command}, {"result", request.Result},
		{"artifacts", request.Artifacts}, {"context", request.Context},
	} {
		if !singleLineText(field.value) {
			return field.name + " must be one non-empty line"
		}
	}
	if request.At.IsZero() {
		return "evidence timestamp is required"
	}
	return ""
}

func validateVerificationRequest(request VerificationRequest) string {
	if !executorIDPattern.MatchString(request.VerifiedBy) {
		return "verified-by must be a stable identifier"
	}
	if request.Mode != VerificationSelf && request.Mode != VerificationIndependent {
		return "verification mode must be self or independent"
	}
	if request.Result != "pass" && request.Result != "rework" {
		return "verification result must be pass or rework"
	}
	if request.At.IsZero() {
		return "verification timestamp is required"
	}
	return ""
}

func validateAcceptanceRequest(request AcceptanceRequest) string {
	if !executorIDPattern.MatchString(request.AcceptedBy) {
		return "accepted-by must be a stable identifier"
	}
	if request.Method != AcceptanceHuman && request.Method != AcceptanceVerified {
		return "acceptance method must be human or verified"
	}
	if request.At.IsZero() {
		return "acceptance timestamp is required"
	}
	return ""
}

func validateRejectionRequest(request RejectionRequest) string {
	if !executorIDPattern.MatchString(request.RejectedBy) {
		return "rejected-by must be a stable identifier"
	}
	if !singleLineText(request.Reason) {
		return "rejection reason must be one non-empty line"
	}
	if request.At.IsZero() {
		return "rejection timestamp is required"
	}
	return ""
}

func existingEvidenceIDs(record evidenceRecord, order []string) []string {
	ids := make([]string, 0, len(record.Results))
	for _, id := range order {
		if _, found := record.Results[id]; found {
			ids = append(ids, id)
		}
	}
	return ids
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func removeStoryLane(source, storyID string) (string, error) {
	markdown := specdoc.ParseMarkdown(source)
	lanes := extractLaneBlocks(source)
	remaining := make([]FieldBlock, 0, len(lanes))
	for _, lane := range lanes {
		if lane.Fields["Story"] != storyID {
			remaining = append(remaining, lane)
		}
	}
	return markdown.ReplaceSection(2, "Active Lanes", renderLanes(remaining))
}

func cloneStories(source map[string]Story) map[string]Story {
	result := make(map[string]Story, len(source))
	for id, story := range source {
		result[id] = story
	}
	return result
}

func allStoriesAccepted(state string, stories map[string]Story) bool {
	statuses := extractAcceptanceBlocks(specdoc.ParseDocument("execution-state.md", state, &diagnostic.Report{}).Body)
	for id := range stories {
		if acceptanceState(id, statuses) != "accepted" {
			return false
		}
	}
	return true
}

func validateUpdatedStateWithStories(file, cycleID string, stories map[string]Story, source string) ValidationResult {
	var report diagnostic.Report
	validateExecutionState(file, cycleID, stories, source, &report)
	return ValidationResult{Errors: report.Errors(), Warnings: report.Warnings()}
}

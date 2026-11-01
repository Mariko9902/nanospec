package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

// CheckpointRequest describes the current execution and recovery facts.
type CheckpointRequest struct {
	Owner     string
	Status    string
	Next      string
	Situation string
	Issue     string
	At        time.Time
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

// RevisionRequest describes a semantic change already made to the specification.
type RevisionRequest struct {
	Reason string
	At     time.Time
}

// RevisionResult holds the changes for one Cycle revision.
type RevisionResult struct {
	CycleID     string
	StoryIDs    []string
	NewRevision string
	Update      ViewUpdate
	Updates     []ViewUpdate
}

var laneNumberPattern = regexp.MustCompile(`^lane-(\d{2})(?:-|$)`)

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
	storyID := filepath.Base(storyDirectory)
	story, found := snapshot.Stories[storyID]
	if !found {
		return CheckpointResult{}, ValidationResult{InputError: "Story is not part of the Cycle: " + storyID}
	}
	acceptance := acceptanceState(storyID, snapshot.Acceptance)
	if acceptance == "ready" || acceptance == "accepted" {
		return CheckpointResult{}, ValidationResult{InputError: "cannot checkpoint a Story with Acceptance state " + acceptance}
	}
	if request.Status != "blocked" {
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
	if lane.Fields["Spec Changes"] == "" {
		lane.Fields["Spec Changes"] = "0"
	}

	updated, err := specdoc.ParseMarkdown(snapshot.StateText).ReplaceSection(2, "Active Lanes", renderLanes(lanes))
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
	updated, err = replaceFrontmatterScalar(updated, "last_updated_at", request.At.Format(time.RFC3339))
	if err != nil {
		return CheckpointResult{}, ValidationResult{InputError: err.Error()}
	}
	stateFile := filepath.Join(snapshot.Directory, "execution-state.md")
	if validation := validateUpdatedState(stateFile, snapshot, updated); validation.InputError != "" || len(validation.Errors) > 0 {
		return CheckpointResult{}, validation
	}
	return CheckpointResult{
		CycleID:      snapshot.ID,
		StoryID:      storyID,
		LaneID:       lane.ID,
		Status:       request.Status,
		SpecRevision: snapshot.SpecRevision,
		Update:       ViewUpdate{File: stateFile, Before: snapshot.StateText, After: updated},
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
	for _, dependency := range dependencyStatuses(story, snapshot.Acceptance) {
		if dependency.Acceptance != "ready" && dependency.Acceptance != "accepted" {
			return "cannot start or continue before dependency is ready: " + dependency.StoryID
		}
	}
	return ""
}

// BuildRevisionUpdate updates revision facts after the edited specification passes validation.
func BuildRevisionUpdate(targets []string, request RevisionRequest) (RevisionResult, ValidationResult) {
	if len(targets) == 0 {
		return RevisionResult{}, ValidationResult{InputError: "revise requires at least one Story"}
	}
	if !singleLineText(request.Reason) {
		return RevisionResult{}, ValidationResult{InputError: "revision reason must be one non-empty line"}
	}
	if request.At.IsZero() {
		return RevisionResult{}, ValidationResult{InputError: "revision timestamp is required"}
	}

	directories := make([]string, 0, len(targets))
	for _, target := range targets {
		directory, inputError := resolveStoryDirectory(target)
		if inputError != "" {
			return RevisionResult{}, ValidationResult{InputError: inputError}
		}
		directories = append(directories, directory)
	}
	cycleDirectory := filepath.Dir(directories[0])
	for _, directory := range directories[1:] {
		if filepath.Dir(directory) != cycleDirectory {
			return RevisionResult{}, ValidationResult{InputError: "all revised Stories must belong to the same Cycle"}
		}
	}
	snapshot, result := loadCycleSnapshot(cycleDirectory, false)
	if result.InputError != "" || len(result.Errors) > 0 {
		return RevisionResult{}, result
	}

	storyIDs := make([]string, 0, len(directories))
	seen := make(map[string]struct{}, len(directories))
	for _, directory := range directories {
		storyID := filepath.Base(directory)
		if _, duplicate := seen[storyID]; duplicate {
			return RevisionResult{}, ValidationResult{InputError: "duplicate revised Story: " + storyID}
		}
		seen[storyID] = struct{}{}
		story, found := snapshot.Stories[storyID]
		if !found {
			return RevisionResult{}, ValidationResult{InputError: "Story is not part of the Cycle: " + storyID}
		}
		if acceptanceState(storyID, snapshot.Acceptance) == "accepted" || story.HasOutcome {
			return RevisionResult{}, ValidationResult{InputError: "accepted Story is frozen: " + storyID}
		}
		storyIDs = append(storyIDs, storyID)
	}

	revisionNumber, err := strconv.Atoi(strings.TrimPrefix(snapshot.SpecRevision, "rev-"))
	if err != nil || revisionNumber < 1 {
		return RevisionResult{}, ValidationResult{InputError: "invalid current spec revision: " + snapshot.SpecRevision}
	}
	newRevision := fmt.Sprintf("rev-%d", revisionNumber+1)
	lanes := cloneFieldBlocks(snapshot.ActiveLanes)
	for index := range lanes {
		if _, affected := seen[lanes[index].Fields["Story"]]; !affected {
			continue
		}
		count, _ := strconv.Atoi(lanes[index].Fields["Spec Changes"])
		lanes[index].Fields["Spec Changes"] = strconv.Itoa(count + 1)
	}

	updated, err := specdoc.ParseMarkdown(snapshot.StateText).ReplaceSection(2, "Active Lanes", renderLanes(lanes))
	if err != nil {
		return RevisionResult{}, ValidationResult{InputError: err.Error()}
	}
	for _, storyID := range storyIDs {
		updated, err = updateAcceptanceState(updated, storyID, "pending: specification revised at "+newRevision+"; "+request.Reason)
		if err != nil {
			return RevisionResult{}, ValidationResult{InputError: err.Error()}
		}
		updated, err = updateManagedStateLine(updated, "Current Situation", storyID, "Specification updated to "+newRevision+"; "+request.Reason)
		if err != nil {
			return RevisionResult{}, ValidationResult{InputError: err.Error()}
		}
	}
	updated, err = replaceFrontmatterScalar(updated, "spec_rev", newRevision)
	if err != nil {
		return RevisionResult{}, ValidationResult{InputError: err.Error()}
	}
	updated, err = replaceFrontmatterScalar(updated, "last_updated_at", request.At.Format(time.RFC3339))
	if err != nil {
		return RevisionResult{}, ValidationResult{InputError: err.Error()}
	}
	stateFile := filepath.Join(snapshot.Directory, "execution-state.md")
	if validation := validateUpdatedState(stateFile, snapshot, updated); validation.InputError != "" || len(validation.Errors) > 0 {
		return RevisionResult{}, validation
	}
	stateBefore := snapshot.StateText
	updates := make([]ViewUpdate, 0, len(result.Updates)+1)
	for _, viewUpdate := range result.Updates {
		if viewUpdate.File == stateFile {
			stateBefore = viewUpdate.Before
			continue
		}
		updates = append(updates, viewUpdate)
	}
	stateUpdate := ViewUpdate{File: stateFile, Before: stateBefore, After: updated}
	updates = append(updates, stateUpdate)
	for _, storyID := range sortedStoryIDs(snapshot.Stories) {
		evidenceFile := filepath.Join(snapshot.Directory, storyID, "acceptance", "evidence.md")
		if !fileExists(evidenceFile) {
			continue
		}
		story := snapshot.Stories[storyID]
		_, affected := seen[storyID]
		var record evidenceRecord
		var before string
		var evidenceResult ValidationResult
		if affected {
			record, before, evidenceResult = loadEvidenceRecordForRevision(evidenceFile, story)
		} else {
			record, before, evidenceResult = loadEvidenceRecord(evidenceFile, story)
		}
		if evidenceResult.InputError != "" || len(evidenceResult.Errors) > 0 {
			return RevisionResult{}, evidenceResult
		}
		record.SpecRevision = newRevision
		if affected {
			record.VerificationStatus = "pending"
			record.VerifiedBy = "none"
			record.VerificationMode = "none"
			record.VerifiedAt = "none"
			record.VerifiedAcceptance = []string{}
			if lane, found := storyLane(storyID, lanes); found {
				record.SpecChangeCount = laneSpecChanges(lane)
			}
			for id, item := range record.Results {
				if !containsString(story.AcceptanceIDs, id) {
					delete(record.Results, id)
					continue
				}
				item.status = "stale"
				record.Results[id] = item
			}
		}
		after := renderEvidenceRecord(record, story.AcceptanceIDs)
		if validation := validateEvidenceSource(evidenceFile, story, after); validation.InputError != "" || len(validation.Errors) > 0 {
			return RevisionResult{}, validation
		}
		updates = append(updates, ViewUpdate{File: evidenceFile, Before: before, After: after})
	}
	return RevisionResult{
		CycleID:     snapshot.ID,
		StoryIDs:    storyIDs,
		NewRevision: newRevision,
		Update:      stateUpdate,
		Updates:     updates,
	}, ValidationResult{}
}

func validateCheckpointRequest(request CheckpointRequest) string {
	if !executorIDPattern.MatchString(request.Owner) {
		return "owner must be a stable identifier"
	}
	if !singleLineText(request.Owner) || !singleLineText(request.Next) || !singleLineText(request.Situation) {
		return "owner, next and situation must each be one non-empty line"
	}
	statuses := map[string]struct{}{"active": {}, "paused": {}, "blocked": {}, "done": {}}
	if _, found := statuses[request.Status]; !found {
		return "checkpoint status must be active, paused, blocked or done"
	}
	if request.Status == "blocked" && !singleLineText(request.Issue) {
		return "blocked checkpoint requires one issue line"
	}
	if request.Issue != "" && !singleLineText(request.Issue) {
		return "checkpoint issue must be one line"
	}
	if request.Status == "done" && request.Next != "none" {
		return "done checkpoint requires next=none"
	}
	if request.At.IsZero() {
		return "checkpoint timestamp is required"
	}
	return ""
}

func singleLineText(value string) bool {
	return strings.TrimSpace(value) != "" && !strings.ContainsAny(value, "\r\n")
}

func cloneFieldBlocks(source []FieldBlock) []FieldBlock {
	result := make([]FieldBlock, 0, len(source))
	for _, block := range source {
		fields := make(map[string]string, len(block.Fields)+1)
		for key, value := range block.Fields {
			fields[key] = value
		}
		result = append(result, FieldBlock{ID: block.ID, Fields: fields})
	}
	return result
}

func nextLaneID(lanes []FieldBlock) string {
	maximum := 0
	for _, lane := range lanes {
		match := laneNumberPattern.FindStringSubmatch(lane.ID)
		if len(match) < 2 {
			continue
		}
		number, _ := strconv.Atoi(match[1])
		if number > maximum {
			maximum = number
		}
	}
	return fmt.Sprintf("lane-%02d", maximum+1)
}

func renderLanes(lanes []FieldBlock) string {
	if len(lanes) == 0 {
		return "- none"
	}
	blocks := make([]string, 0, len(lanes))
	for _, lane := range lanes {
		lines := []string{
			"- Lane: " + lane.ID,
			"  - Story: " + lane.Fields["Story"],
			"  - Owner: " + lane.Fields["Owner"],
			"  - Status: " + lane.Fields["Status"],
			"  - Next: " + lane.Fields["Next"],
		}
		if lane.Fields["Spec Changes"] != "" {
			lines = append(lines, "  - Spec Changes: "+lane.Fields["Spec Changes"])
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n")
}

func updateManagedStateLine(source, sectionName, storyID, value string) (string, error) {
	markdown := specdoc.ParseMarkdown(source)
	found, ok := markdown.Section(2, sectionName)
	if !ok {
		return "", fmt.Errorf("missing required section: %s", sectionName)
	}
	prefix := "- " + storyID + ":"
	lines := make([]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(found.Content, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "- none" || strings.HasPrefix(trimmed, prefix) {
			continue
		}
		lines = append(lines, line)
	}
	if value != "" {
		lines = append(lines, prefix+" "+value)
	}
	content := strings.TrimSpace(strings.Join(lines, "\n"))
	if content == "" {
		content = "- none"
	}
	return markdown.ReplaceSection(2, sectionName, content)
}

func updateAcceptanceState(source, storyID, value string) (string, error) {
	markdown := specdoc.ParseMarkdown(source)
	found, ok := markdown.Section(2, "Acceptance Status")
	if !ok {
		return "", fmt.Errorf("missing required section: Acceptance Status")
	}
	lines := strings.Split(strings.ReplaceAll(found.Content, "\r\n", "\n"), "\n")
	foundStory := false
	updated := false
	for index, line := range lines {
		if strings.TrimSpace(line) == "- Story: "+storyID {
			foundStory = true
			continue
		}
		if foundStory && strings.HasPrefix(strings.TrimSpace(line), "- Acceptance:") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[index] = indent + "- Acceptance: " + value
			updated = true
			break
		}
		if foundStory && strings.HasPrefix(strings.TrimSpace(line), "- Story:") {
			break
		}
	}
	if !updated {
		return "", fmt.Errorf("Acceptance Status missing Story: %s", storyID)
	}
	return markdown.ReplaceSection(2, "Acceptance Status", strings.Join(lines, "\n"))
}

func replaceFrontmatterScalar(source, key, value string) (string, error) {
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:[^\r\n]*$`)
	if len(pattern.FindAllStringIndex(source, -1)) != 1 {
		return "", fmt.Errorf("frontmatter must contain exactly one %s field", key)
	}
	return pattern.ReplaceAllString(source, key+": "+value), nil
}

func validateUpdatedState(file string, snapshot CycleSnapshot, source string) ValidationResult {
	var report diagnostic.Report
	validateExecutionState(file, snapshot.ID, snapshot.Stories, source, &report)
	return ValidationResult{Errors: report.Errors(), Warnings: report.Warnings()}
}

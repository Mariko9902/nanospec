package validator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

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
		stage := lanes[index].Fields["Status"]
		if stage == "blocked" {
			stage = lanes[index].Fields["Resume Status"]
		}
		if stage == "pending" {
			continue
		}
		count, _ := strconv.Atoi(lanes[index].Fields["Spec Changes"])
		lanes[index].Fields["Spec Changes"] = strconv.Itoa(count + 1)
		lane := &lanes[index]
		lane.Fields["Implementation Complete"] = "false"
		if lane.Fields["Status"] == "blocked" {
			if lane.Fields["Resume Status"] == "ready" {
				lane.Fields["Resume Status"] = "active"
			}
		} else if lane.Fields["Status"] == "rejected" {
			lane.Fields["Next"] = "Await rework assignment at " + newRevision
		} else {
			resume := lane.Fields["Status"]
			if resume == "ready" {
				resume = "active"
			}
			lane.Fields["Resume Status"] = resume
			lane.Fields["Status"] = "blocked"
			lane.Fields["Next"] = "Reassess specification at " + newRevision + "; " + request.Reason
		}
	}

	updated, err := specdoc.ParseMarkdown(snapshot.CycleText).ReplaceSection(2, "Active Lanes", renderLanes(lanes))
	if err != nil {
		return RevisionResult{}, ValidationResult{InputError: err.Error()}
	}
	for _, storyID := range storyIDs {
		stateValue := "pending: specification revised at " + newRevision + "; " + request.Reason
		if lane, found := storyLane(storyID, lanes); found && (lane.Fields["Status"] == "rejected" || lane.Fields["Resume Status"] == "rejected") {
			stateValue = "rejected: specification revised at " + newRevision + "; await rework assignment"
		}
		updated, err = updateAcceptanceState(updated, storyID, stateValue)
		if err != nil {
			return RevisionResult{}, ValidationResult{InputError: err.Error()}
		}
		updated, err = updateManagedStateLine(updated, "Current Situation", storyID, "Specification updated to "+newRevision+"; "+request.Reason)
		if err != nil {
			return RevisionResult{}, ValidationResult{InputError: err.Error()}
		}
	}
	updated, err = specdoc.ReplaceFrontmatterScalar(updated, "spec_rev", newRevision)
	if err != nil {
		return RevisionResult{}, ValidationResult{InputError: err.Error()}
	}
	updated, err = specdoc.ReplaceFrontmatterScalar(updated, "last_updated_at", request.At.Format(time.RFC3339))
	if err != nil {
		return RevisionResult{}, ValidationResult{InputError: err.Error()}
	}
	stateFile := filepath.Join(snapshot.Directory, "cycle.md")
	var validation ValidationResult
	updated, validation = canonicalizeCycleState(stateFile, snapshot.ID, snapshot.Stories, updated)
	if validation.InputError != "" || len(validation.Errors) > 0 {
		return RevisionResult{}, validation
	}
	stateBefore := snapshot.CycleText
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
			"  - Implementation Complete: " + lane.Fields["Implementation Complete"],
		}
		if lane.Fields["Resume Status"] != "" {
			lines = append(lines, "  - Resume Status: "+lane.Fields["Resume Status"])
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

func setStoryLane(lanes []FieldBlock, storyID, status, next string, complete bool) []FieldBlock {
	updated := cloneFieldBlocks(lanes)
	for index := range updated {
		if updated[index].Fields["Story"] == storyID {
			updated[index].Fields["Status"] = status
			updated[index].Fields["Next"] = next
			updated[index].Fields["Implementation Complete"] = strconv.FormatBool(complete)
			delete(updated[index].Fields, "Resume Status")
		}
	}
	return updated
}

func canonicalizeCycleState(file, cycleID string, stories map[string]Story, source string) (string, ValidationResult) {
	updated, err := RenderCycleViews(source, stories)
	if err != nil {
		return "", ValidationResult{InputError: err.Error()}
	}
	var report diagnostic.Report
	validateCycleFile(file, cycleID, stories, updated, &report)
	return updated, ValidationResult{Errors: report.Errors(), Warnings: report.Warnings()}
}

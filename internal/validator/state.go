package validator

import (
	"regexp"
	"strconv"
	"strings"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

type progressItem struct {
	id   string
	done bool
}

type feedbackBlock struct {
	problem string
	fields  map[string]string
}

var (
	progressPattern     = regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+(story-\S+)\s*$`)
	unsupportedProgress = regexp.MustCompile(`^\s*-\s+\[[ xX]\]\s+(\S+)\s*$`)
	storyBlockPattern   = regexp.MustCompile(`^\s*-\s+Story:\s*(story-\S+)\s*$`)
	laneBlockPattern    = regexp.MustCompile(`^\s*-\s+Lane:\s*(\S+)\s*$`)
	blockFieldPattern   = regexp.MustCompile(`^\s+-\s+([A-Za-z][A-Za-z ]*):\s*(.*?)\s*$`)
	feedbackProblem     = regexp.MustCompile(`^\s*-\s+Observed Problem:\s*(.*?)\s*$`)
	feedbackField       = regexp.MustCompile(`^\s+-\s+(Spec Impact|Proposed Fix|Status):\s*(.*?)\s*$`)
	laneIDPattern       = regexp.MustCompile(`^lane-\d{2}(?:-[a-z0-9][a-z0-9-]*)?$`)
	revisionPattern     = regexp.MustCompile(`^rev-[1-9]\d*$`)
)

func extractProgress(file, body string, report *diagnostic.Report) []progressItem {
	content := sectionContent(body, 2, "Progress Tree")
	items := make([]progressItem, 0)
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if match := progressPattern.FindStringSubmatch(line); match != nil {
			items = append(items, progressItem{id: match[2], done: strings.EqualFold(match[1], "x")})
			continue
		}
		if match := unsupportedProgress.FindStringSubmatch(line); match != nil {
			report.AddError(file, "Progress Tree contains unsupported node: "+match[1])
		}
	}
	return items
}

func extractAcceptanceBlocks(body string) []FieldBlock {
	return extractFieldBlocks(sectionContent(body, 2, "Acceptance Status"), storyBlockPattern)
}

func extractLaneBlocks(body string) []FieldBlock {
	return extractFieldBlocks(sectionContent(body, 2, "Active Lanes"), laneBlockPattern)
}

func extractFieldBlocks(content string, startPattern *regexp.Regexp) []FieldBlock {
	blocks := make([]FieldBlock, 0)
	current := -1
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if match := startPattern.FindStringSubmatch(line); match != nil {
			blocks = append(blocks, FieldBlock{ID: match[1], Fields: map[string]string{}})
			current = len(blocks) - 1
			continue
		}
		if match := blockFieldPattern.FindStringSubmatch(line); match != nil && current >= 0 {
			blocks[current].Fields[match[1]] = match[2]
		}
	}
	return blocks
}

func extractFeedbackBlocks(body string) []feedbackBlock {
	content := sectionContent(body, 2, "Practice Feedback")
	blocks := make([]feedbackBlock, 0)
	current := -1
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if match := feedbackProblem.FindStringSubmatch(line); match != nil {
			blocks = append(blocks, feedbackBlock{problem: match[1], fields: map[string]string{}})
			current = len(blocks) - 1
			continue
		}
		if match := feedbackField.FindStringSubmatch(line); match != nil && current >= 0 {
			blocks[current].fields[match[1]] = match[2]
		}
	}
	return blocks
}

func validateActiveLanes(file, body string, stories map[string]Story, cycleStatus string, report *diagnostic.Report) {
	content := sectionContent(body, 2, "Active Lanes")
	if specdoc.IsExplicitNone(content) {
		return
	}
	lanes := extractLaneBlocks(body)
	if len(lanes) == 0 {
		report.AddError(file, "Active Lanes must contain lane blocks or - none")
		return
	}

	ids := make(map[string]struct{})
	assignedStories := make(map[string]string)
	supportedFields := map[string]struct{}{"Story": {}, "Owner": {}, "Status": {}, "Next": {}, "Spec Changes": {}}
	supportedStatuses := map[string]struct{}{"active": {}, "paused": {}, "blocked": {}, "done": {}}
	for _, lane := range lanes {
		if !laneIDPattern.MatchString(lane.ID) {
			report.AddError(file, "Active Lanes has invalid lane id: "+lane.ID)
		}
		if _, duplicate := ids[lane.ID]; duplicate {
			report.AddError(file, "Active Lanes has duplicate lane id: "+lane.ID)
		}
		ids[lane.ID] = struct{}{}
		for _, field := range sortedMapKeys(lane.Fields) {
			if _, supported := supportedFields[field]; !supported {
				report.AddError(file, "Active Lanes "+lane.ID+" has unsupported field: "+field)
			}
		}
		for _, field := range []string{"Story", "Owner", "Status", "Next"} {
			if lane.Fields[field] == "" {
				report.AddError(file, "Active Lanes "+lane.ID+" missing "+field)
			}
		}
		if !executorIDPattern.MatchString(lane.Fields["Owner"]) {
			report.AddError(file, "Active Lanes "+lane.ID+" Owner must be a stable identifier")
		}
		if storyID := lane.Fields["Story"]; storyID != "" {
			if _, found := stories[storyID]; !found {
				report.AddError(file, "Active Lanes "+lane.ID+" references missing story: "+storyID)
			}
			if existing, duplicate := assignedStories[storyID]; duplicate {
				report.AddError(file, "Story must not have more than one active Lane: "+storyID+" ("+existing+", "+lane.ID+")")
			}
			assignedStories[storyID] = lane.ID
		}
		if _, found := supportedStatuses[lane.Fields["Status"]]; !found {
			report.AddError(file, "Active Lanes "+lane.ID+" has invalid Status: "+lane.Fields["Status"])
		}
		if cycleStatus == "done" && lane.Fields["Status"] != "done" {
			report.AddError(file, "Active Lanes "+lane.ID+" must be done when execution status is done")
		}
		if changes := lane.Fields["Spec Changes"]; changes != "" {
			value, err := strconv.Atoi(changes)
			if err != nil || value < 0 {
				report.AddError(file, "Active Lanes "+lane.ID+" Spec Changes must be a non-negative integer")
			}
		}
	}
}

func validateAcceptanceStatus(file, body string, stories map[string]Story, report *diagnostic.Report) map[string]string {
	blocks := extractAcceptanceBlocks(body)
	if len(blocks) == 0 {
		report.AddError(file, "Acceptance Status must contain Story blocks")
		return map[string]string{}
	}
	statuses := make(map[string]string)
	allowed := map[string]struct{}{"pending": {}, "ready": {}, "accepted": {}, "rejected": {}}
	for _, block := range blocks {
		if _, found := stories[block.ID]; !found {
			report.AddError(file, "Acceptance Status references missing story: "+block.ID)
		}
		if _, duplicate := statuses[block.ID]; duplicate {
			report.AddError(file, "Acceptance Status has duplicate story block: "+block.ID)
		}
		for _, field := range sortedMapKeys(block.Fields) {
			if field != "Acceptance" {
				report.AddError(file, "Acceptance Status "+block.ID+" has unsupported field: "+field)
			}
		}
		acceptance := block.Fields["Acceptance"]
		if acceptance == "" {
			report.AddError(file, "Acceptance Status "+block.ID+" missing Acceptance")
			continue
		}
		status := leadingToken(acceptance)
		if _, found := allowed[status]; !found {
			report.AddError(file, "Acceptance Status "+block.ID+" has invalid Acceptance state: "+acceptance)
		}
		statuses[block.ID] = status
	}
	for _, storyID := range sortedStoryIDs(stories) {
		if _, found := statuses[storyID]; !found {
			report.AddError(file, "Acceptance Status missing story block: "+storyID)
		}
	}
	return statuses
}

func validateFeedback(file, body string, report *diagnostic.Report) {
	content := sectionContent(body, 2, "Practice Feedback")
	if specdoc.IsExplicitNone(content) {
		return
	}
	blocks := extractFeedbackBlocks(body)
	if len(blocks) == 0 {
		report.AddError(file, "Practice Feedback must contain feedback blocks or - none")
		return
	}
	impacts := map[string]struct{}{"instance": {}, "nanospec-rule": {}, "nanospec-template": {}, "nanospec-tool": {}}
	statuses := map[string]struct{}{"open": {}, "evaluating": {}, "fixed": {}, "rejected": {}}
	for _, block := range blocks {
		if block.problem == "" {
			report.AddError(file, "Practice Feedback missing Observed Problem")
		}
		for _, field := range []string{"Spec Impact", "Proposed Fix", "Status"} {
			if block.fields[field] == "" {
				report.AddError(file, "Practice Feedback missing "+field)
			}
		}
		if impact := block.fields["Spec Impact"]; impact != "" {
			if _, found := impacts[impact]; !found {
				report.AddError(file, "Practice Feedback has invalid Spec Impact: "+impact)
			}
		}
		if status := block.fields["Status"]; status != "" {
			if _, found := statuses[status]; !found {
				report.AddError(file, "Practice Feedback has invalid Status: "+status)
			}
		}
	}
}

func validateExecutionState(file, cycleID string, stories map[string]Story, text string, report *diagnostic.Report) {
	document := specdoc.ParseDocument(file, text, report)
	frontmatterKeys := []string{"cycle", "spec_rev", "status", "last_updated_at"}
	requireKeys(file, document.Frontmatter, frontmatterKeys, report)
	validateAllowedKeys(file, document.Frontmatter, frontmatterKeys, "Execution State", report)
	cycle, _ := document.Frontmatter.String("cycle")
	if cycle != cycleID {
		report.AddError(file, "execution-state.cycle must equal "+cycleID)
	}
	revision, _ := document.Frontmatter.String("spec_rev")
	if !revisionPattern.MatchString(revision) {
		report.AddError(file, "spec_rev must match rev-N with N >= 1")
	}
	cycleStatus, _ := document.Frontmatter.String("status")
	validCycleStatuses := map[string]struct{}{"active": {}, "paused": {}, "blocked": {}, "done": {}}
	if _, found := validCycleStatuses[cycleStatus]; !found {
		report.AddError(file, "invalid execution status: "+cycleStatus)
	}
	lastUpdated, _ := document.Frontmatter.String("last_updated_at")
	if !isISOTimestamp(lastUpdated) {
		report.AddError(file, "last_updated_at must be an ISO-8601 timestamp with timezone")
	}

	requiredSections := []string{
		"Progress Tree", "Current Situation", "Active Lanes",
		"Acceptance Status", "Open Issues", "Practice Feedback",
	}
	requireSections(file, document.Body, 2, requiredSections, report)
	validateAllowedSections(file, document.Body, 2, requiredSections, "Execution State", report)
	allowNone := map[string]struct{}{"Current Situation": {}, "Open Issues": {}, "Practice Feedback": {}}
	for _, name := range requiredSections {
		_, allowed := allowNone[name]
		requireMeaningfulSection(file, document.Body, name, allowed, report)
	}

	progressMap := make(map[string]bool)
	for _, item := range extractProgress(file, document.Body, report) {
		if _, found := stories[item.id]; !found {
			report.AddError(file, "Progress Tree references missing story: "+item.id)
		}
		if _, duplicate := progressMap[item.id]; duplicate {
			report.AddError(file, "Progress Tree has duplicate story: "+item.id)
		}
		progressMap[item.id] = item.done
	}
	for _, storyID := range sortedStoryIDs(stories) {
		if _, found := progressMap[storyID]; !found {
			report.AddError(file, "Progress Tree missing story: "+storyID)
		}
	}

	validateActiveLanes(file, document.Body, stories, cycleStatus, report)
	statuses := validateAcceptanceStatus(file, document.Body, stories, report)
	validateFeedback(file, document.Body, report)

	for _, storyID := range sortedStoryIDs(stories) {
		story := stories[storyID]
		done := progressMap[story.ID]
		status := statuses[story.ID]
		accepted := status == "accepted"
		if done && !accepted {
			report.AddError(file, "completed story must have accepted Acceptance status: "+story.ID)
		}
		if done && !story.HasOutcome {
			report.AddError(file, "completed story must contain outcome.md: "+story.ID)
		}
		if accepted && !done {
			report.AddError(file, "accepted story must be checked in Progress Tree: "+story.ID)
		}
		if accepted && !story.HasOutcome {
			report.AddError(file, "accepted story must contain outcome.md: "+story.ID)
		}
		if story.HasOutcome && (!done || !accepted) {
			report.AddError(file, "story with outcome.md must be completed and accepted: "+story.ID)
		}
		if cycleStatus == "done" && !done {
			report.AddError(file, "all stories must be completed when execution status is done: "+story.ID)
		}
	}
}

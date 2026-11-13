package validator

import (
	"os"
	"path/filepath"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

func validateCycle(cycleDirectory string, updates *[]ViewUpdate, validateEvidenceRecords bool, report *diagnostic.Report) {
	cycleID := filepath.Base(cycleDirectory)
	if !cycleIDPattern.MatchString(cycleID) {
		report.AddError(cycleDirectory, "cycle directory must match cycle-NN-slug")
	}
	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	if !fileExists(cycleFile) {
		report.AddError(cycleFile, "missing cycle.md")
	}
	stories := make(map[string]Story)
	entries, err := os.ReadDir(cycleDirectory)
	if err != nil {
		report.AddError(cycleDirectory, "cannot list cycle directory: "+err.Error())
		return
	}
	for _, entry := range entries {
		entryPath := filepath.Join(cycleDirectory, entry.Name())
		if !entry.IsDir() {
			if entry.Name() != "cycle.md" {
				report.AddError(entryPath, "unexpected file in cycle directory")
			}
			continue
		}
		if !storyIDPattern.MatchString(entry.Name()) {
			report.AddError(entryPath, "unexpected directory in cycle directory")
			continue
		}
		story := validateStoryDirectory(entryPath, cycleID, validateEvidenceRecords, report)
		if story.ID != "" {
			stories[story.ID] = story
		}
	}
	if len(stories) == 0 {
		report.AddError(cycleDirectory, "formal cycle must contain at least one story-NN-<slug> directory")
	}
	validateStoryGraph(stories, report)
	if !fileExists(cycleFile) {
		return
	}
	text := readText(cycleFile, report)
	updated, err := RenderCycleViews(text, stories)
	if err != nil {
		report.AddError(cycleFile, err.Error())
	} else if updated != text {
		if updates == nil {
			report.AddError(cycleFile, "Dependency Graph is out of date; run nanospec validate --sync-views")
		} else {
			*updates = append(*updates, ViewUpdate{File: cycleFile, Before: text, After: updated})
			text = updated
		}
	}
	validateCycleFile(cycleFile, cycleID, stories, text, report)
}

func validateCycleFile(file, cycleID string, stories map[string]Story, text string, report *diagnostic.Report) {
	document := specdoc.ParseDocument(file, text, report)
	frontmatterKeys := []string{"id", "window", "spec_rev", "status", "last_updated_at"}
	requireKeys(file, document.Frontmatter, frontmatterKeys, report)
	validateAllowedKeys(file, document.Frontmatter, append(frontmatterKeys, "spec_language"), "Cycle", report)
	if _, err := readSpecLanguage(document.Frontmatter); err != nil {
		report.AddError(file, err.Error())
	}
	id, _ := document.Frontmatter.String("id")
	if id != cycleID {
		report.AddError(file, "cycle.id must equal directory name ("+cycleID+")")
	}
	sections := []string{"Planning Context", "Dependency Graph", "Repartition Records", "Risks", "Current Situation", "Active Lanes", "Acceptance Status", "Open Issues", "Practice Feedback"}
	validateAllowedSections(file, document.Body, 2, sections, "Cycle", report)
	requireSectionOrder(file, document.Body, 2, sections, "Cycle", report)
	for _, name := range []string{"Planning Context", "Dependency Graph", "Risks"} {
		requireSections(file, document.Body, 2, []string{name}, report)
		requireMeaningfulSection(file, document.Body, name, name == "Risks", report)
	}
	validateExecutionState(file, stories, text, report)
}

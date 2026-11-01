package validator

import (
	"os"
	"path/filepath"
	"regexp"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

var validWaveTitle = regexp.MustCompile(`^Wave\s+[1-9]\d*$`)

func validateCycle(
	cycleDirectory string,
	updates *[]ViewUpdate,
	validateEvidenceRecords bool,
	report *diagnostic.Report,
) {
	cycleID := filepath.Base(cycleDirectory)
	if !cycleIDPattern.MatchString(cycleID) {
		report.AddError(cycleDirectory, "cycle directory must match cycle-NN-slug")
	}

	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	stateFile := filepath.Join(cycleDirectory, "execution-state.md")
	if !fileExists(cycleFile) {
		report.AddError(cycleFile, "missing cycle.md")
	}
	if !fileExists(stateFile) {
		report.AddError(stateFile, "missing execution-state.md")
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
			if entry.Name() != "cycle.md" && entry.Name() != "execution-state.md" {
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

	cycleText := ""
	stateText := ""
	if fileExists(cycleFile) {
		cycleText = readText(cycleFile, report)
	}
	if fileExists(stateFile) {
		stateText = readText(stateFile, report)
	}
	if cycleText != "" && stateText != "" {
		views, renderErr := RenderCycleViews(cycleText, stateText, stories)
		if renderErr != nil {
			report.AddError(cycleDirectory, renderErr.Error())
		} else if updates != nil {
			if views.Cycle != cycleText {
				*updates = append(*updates, ViewUpdate{File: cycleFile, Before: cycleText, After: views.Cycle})
			}
			if views.State != stateText {
				*updates = append(*updates, ViewUpdate{File: stateFile, Before: stateText, After: views.State})
			}
			cycleText = views.Cycle
			stateText = views.State
		} else {
			if views.Cycle != cycleText {
				report.AddError(cycleFile, "generated Cycle views are out of date; run nanospec validate --sync-views")
			}
			if views.State != stateText {
				report.AddError(stateFile, "Progress Tree is out of date; run nanospec validate --sync-views")
			}
		}
	}

	if cycleText != "" {
		validateCycleFile(cycleFile, cycleID, stories, cycleText, report)
	}
	if stateText != "" {
		validateExecutionState(stateFile, cycleID, stories, stateText, report)
	}
	if cycleText != "" && stateText != "" {
		for _, issue := range validateOrder(cycleText, stateText, stories) {
			file := stateFile
			if issue.scope == "cycle" {
				file = cycleFile
			}
			report.AddError(file, issue.message)
		}
	}
}

func validateCycleFile(
	file, cycleID string,
	stories map[string]Story,
	text string,
	report *diagnostic.Report,
) {
	document := specdoc.ParseDocument(file, text, report)
	frontmatterKeys := []string{"id", "window"}
	requireKeys(file, document.Frontmatter, frontmatterKeys, report)
	validateAllowedKeys(file, document.Frontmatter, append(frontmatterKeys, "spec_language"), "Cycle", report)
	if _, err := readSpecLanguage(document.Frontmatter); err != nil {
		report.AddError(file, err.Error())
	}
	id, _ := document.Frontmatter.String("id")
	if id != cycleID {
		report.AddError(file, "cycle.id must equal directory name ("+cycleID+")")
	}

	required := []string{"Planning Context", "Story Plan", "Risks"}
	requireSections(file, document.Body, 2, required, report)
	cycleSections := []string{"Planning Context", "Story Plan", "Repartition Records", "Risks"}
	validateAllowedSections(file, document.Body, 2, cycleSections, "Cycle", report)
	requireSectionOrder(file, document.Body, 2, cycleSections, "Cycle", report)
	for _, name := range required {
		requireMeaningfulSection(file, document.Body, name, name == "Risks", report)
	}
	plan := sectionContent(document.Body, 2, "Story Plan")
	planSections := []string{"Dependency Waves", "Dependency Graph"}
	validateAllowedSections(file, plan, 3, planSections, "Story Plan", report)
	requireSectionOrder(file, plan, 3, planSections, "Story Plan", report)
	for _, name := range planSections {
		found, ok := section(plan, 3, name)
		if !ok {
			report.AddError(file, "Story Plan missing subsection: "+name)
			continue
		}
		if !specdoc.Meaningful(found.Content) {
			report.AddError(file, "Story Plan subsection is empty: "+name)
		}
	}

	waveContent := sectionContent(plan, 3, "Dependency Waves")
	for _, heading := range specdoc.ParseMarkdown(waveContent).Headings() {
		if heading.Level == 4 && !validWaveTitle.MatchString(heading.Name) {
			report.AddError(file, "Dependency Waves has unsupported heading: "+heading.Name)
		}
	}
	plannedStories := make(map[string]struct{})
	for _, entry := range flattenWaveEntries(readDependencyWaves(text)) {
		if _, duplicate := plannedStories[entry.ID]; duplicate {
			report.AddError(file, "Dependency Waves has duplicate story: "+entry.ID)
		}
		plannedStories[entry.ID] = struct{}{}
	}
	for _, storyID := range sortedMapKeys(plannedStories) {
		if _, found := stories[storyID]; !found {
			report.AddError(file, "Dependency Waves references missing story: "+storyID)
		}
	}
	for _, storyID := range sortedStoryIDs(stories) {
		if _, found := plannedStories[storyID]; !found {
			report.AddError(file, "Dependency Waves missing story: "+storyID)
		}
	}
}

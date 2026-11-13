package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

// LoadCycleSnapshot validates a Cycle and builds its views without writing files.
func LoadCycleSnapshot(target string) (CycleSnapshot, ValidationResult) {
	return loadCycleSnapshot(target, true)
}

func loadCycleSnapshot(target string, validateEvidenceRecords bool) (CycleSnapshot, ValidationResult) {
	cycleDirectory, inputError := resolveCycleDirectory(target)
	if inputError != "" {
		return CycleSnapshot{}, ValidationResult{InputError: inputError}
	}
	result := validateSpec(cycleDirectory, true, validateEvidenceRecords)
	if result.InputError != "" || len(result.Errors) > 0 {
		return CycleSnapshot{}, result
	}

	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	cycleText, err := os.ReadFile(cycleFile)
	if err != nil {
		return CycleSnapshot{}, ValidationResult{InputError: "cannot read cycle.md: " + err.Error()}
	}

	stories := loadSnapshotStories(cycleDirectory)
	var report diagnostic.Report
	cycleDocument := specdoc.ParseDocument(cycleFile, string(cycleText), &report)
	revision, _ := cycleDocument.Frontmatter.String("spec_rev")
	language, err := readSpecLanguage(cycleDocument.Frontmatter)
	if err != nil {
		return CycleSnapshot{}, ValidationResult{InputError: err.Error()}
	}
	return CycleSnapshot{
		ID:           filepath.Base(cycleDirectory),
		Directory:    cycleDirectory,
		SpecRevision: revision,
		SpecLanguage: language,
		CycleText:    string(cycleText),
		Stories:      stories,
		Acceptance:   extractAcceptanceBlocks(cycleDocument.Body),
		ActiveLanes:  extractLaneBlocks(cycleDocument.Body),
	}, result
}

// View returns a Cycle view as Markdown.
func (snapshot CycleSnapshot) View(name string) (string, error) {
	switch name {
	case "overview":
		return snapshot.overviewView()
	case "graph":
		text, err := RenderCycleViews(snapshot.CycleText, snapshot.Stories)
		if err != nil {
			return "", err
		}
		return "## Dependency Graph\n\n" + sectionContent(text, 2, "Dependency Graph") + "\n", nil
	default:
		return "", fmt.Errorf("unknown view: %s", name)
	}
}

func (snapshot CycleSnapshot) overviewView() (string, error) {
	var output strings.Builder
	fmt.Fprintln(&output, "# Cycle Overview")
	order, err := orderedStoryIDs(snapshot.Stories)
	if err != nil {
		return "", err
	}
	for _, id := range order {
		story := snapshot.Stories[id]
		storyText, err := os.ReadFile(story.File)
		if err != nil {
			return "", fmt.Errorf("cannot read overview Story %s: %w", id, err)
		}
		orientation := sectionContent(string(storyText), 2, "Orientation")
		if strings.TrimSpace(orientation) == "" {
			return "", fmt.Errorf("overview Story has no Orientation: %s", id)
		}
		fmt.Fprintf(&output, "\n## [%s] %s (%s)\n\n", storyStatus(id, snapshot.Acceptance, snapshot.ActiveLanes), story.Title, id)
		fmt.Fprintln(&output, orientationDetails(orientation))
	}

	return output.String(), nil
}

func orientationDetails(content string) string {
	lines := make([]string, 0, 2)
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		matches := orientationFieldPattern.FindStringSubmatch(line)
		if matches != nil && strings.TrimSpace(matches[1]) == "Title" {
			continue
		}
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// ReadyCandidates lists unassigned Stories whose recorded dependency states allow further checks.
// Evidence applicability, environment availability, and required human approval still need a separate check.
func (snapshot CycleSnapshot) ReadyCandidates() []ReadyCandidate {
	acceptance := make(map[string]string, len(snapshot.Acceptance))
	for _, block := range snapshot.Acceptance {
		acceptance[block.ID] = leadingToken(block.Fields["Acceptance"])
	}
	assigned := make(map[string]struct{})
	for _, lane := range snapshot.ActiveLanes {
		assigned[lane.Fields["Story"]] = struct{}{}
	}

	candidates := make([]ReadyCandidate, 0)
	for _, id := range sortedStoryIDs(snapshot.Stories) {
		story := snapshot.Stories[id]
		current := acceptance[id]
		if current != "pending" && current != "rejected" {
			continue
		}
		if _, found := assigned[id]; found {
			continue
		}
		dependencies := dependencyStatuses(story, snapshot)
		allowed := true
		for _, dependency := range dependencies {
			if dependency.Status != StoryReady && dependency.Status != StoryAccepted {
				allowed = false
			}
		}
		if allowed {
			candidates = append(candidates, ReadyCandidate{StoryID: id, Acceptance: current, Dependencies: dependencies})
		}
	}

	return candidates
}

// BuildExecutionPackage gathers the selected Story, state, dependencies, and execution rules without writing files.
func BuildExecutionPackage(target, executionRules string) (ExecutionPackage, ValidationResult) {
	if strings.TrimSpace(executionRules) == "" {
		return ExecutionPackage{}, ValidationResult{InputError: "NanoSpec execution rules are empty"}
	}
	storyDirectory, inputError := resolveStoryDirectory(target)
	if inputError != "" {
		return ExecutionPackage{}, ValidationResult{InputError: inputError}
	}
	snapshot, result := LoadCycleSnapshot(filepath.Dir(storyDirectory))
	if result.InputError != "" || len(result.Errors) > 0 {
		return ExecutionPackage{}, result
	}
	storyID := filepath.Base(storyDirectory)
	story, found := snapshot.Stories[storyID]
	if !found {
		return ExecutionPackage{}, ValidationResult{InputError: "Story is not part of the Cycle: " + storyID}
	}
	storyText, err := os.ReadFile(story.File)
	if err != nil {
		return ExecutionPackage{}, ValidationResult{InputError: "cannot read story.md: " + err.Error()}
	}
	acceptance := acceptanceState(story.ID, snapshot.Acceptance)
	dependencies := dependencyStatuses(story, snapshot)
	openQuestions := hasOpenQuestions(string(storyText))

	var output strings.Builder
	fmt.Fprintln(&output, "# NanoSpec Execution Context")
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "- Cycle: %s\n", snapshot.ID)
	fmt.Fprintf(&output, "- Story: %s\n", story.ID)
	fmt.Fprintf(&output, "- Spec Revision: %s\n", snapshot.SpecRevision)
	fmt.Fprintf(&output, "- Spec Language: %s\n", snapshot.SpecLanguage)
	recordName := "evidence.md"
	if story.HasOutcome {
		recordName = "outcome.md"
	}
	fmt.Fprintf(&output, "- Acceptance Record: %s\n", filepath.Join(storyDirectory, "acceptance", recordName))
	fmt.Fprintf(&output, "- Evidence Directory: %s\n", filepath.Join(storyDirectory, "acceptance", "evidence"))
	fmt.Fprintln(&output)
	writeDependencyContext(&output, story, snapshot.Acceptance)
	writeCurrentStoryState(&output, story.ID, snapshot)
	writeContextSection(&output, "NanoSpec Execution Rules", strings.TrimSpace(executionRules))
	writeContextSection(&output, "Current Situation", sectionContent(snapshot.CycleText, 2, "Current Situation"))
	writeContextSection(&output, "Open Issues", sectionContent(snapshot.CycleText, 2, "Open Issues"))
	writeContextSection(&output, "Story Specification", strings.TrimSpace(string(storyText)))
	if inputError := writeRegressionContext(&output, story, result.Warnings); inputError != "" {
		return ExecutionPackage{}, ValidationResult{InputError: inputError}
	}

	if story.ImplementationBrief == "implementation-brief.md" {
		brief, err := os.ReadFile(filepath.Join(story.Directory, story.ImplementationBrief))
		if err != nil {
			return ExecutionPackage{}, ValidationResult{InputError: "cannot read implementation-brief.md: " + err.Error()}
		}
		writeContextSection(&output, "Implementation Brief", strings.TrimSpace(string(brief)))
	}
	confirmations := []string{"current_environment_available"}
	if len(dependencies) > 0 {
		confirmations = append(confirmations, "dependency_evidence_applicable", "dependency_results_available")
	}
	confirmations = append(confirmations, "declared_human_acceptance_requirements_satisfied")
	return ExecutionPackage{
		CycleID:               snapshot.ID,
		StoryID:               story.ID,
		SpecRevision:          snapshot.SpecRevision,
		SpecLanguage:          snapshot.SpecLanguage,
		Acceptance:            acceptance,
		Status:                storyStatus(story.ID, snapshot.Acceptance, snapshot.ActiveLanes),
		ResumeStatus:          storyResumeStatus(story.ID, snapshot.ActiveLanes),
		Dependencies:          dependencies,
		OpenQuestions:         openQuestions,
		RequiredConfirmations: confirmations,
		ContextMarkdown:       output.String(),
	}, result
}

// ResolvePreparationGates determines startup conditions for the explicit selection.
func ResolvePreparationGates(packages []ExecutionPackage) {
	selected := make(map[string]struct{}, len(packages))
	for _, item := range packages {
		selected[item.CycleID+"/"+item.StoryID] = struct{}{}
	}
	for index := range packages {
		item := &packages[index]
		if item.Status == StoryBlocked {
			item.Gate = "blocked"
			continue
		}
		switch item.Acceptance {
		case "accepted":
			item.Gate = "accepted"
			continue
		case "ready":
			item.Gate = "awaiting_acceptance"
			continue
		}
		if item.OpenQuestions {
			item.Gate = "blocked_by_open_questions"
			continue
		}
		waitingForSelected := false
		blocked := false
		for _, dependency := range item.Dependencies {
			if dependency.Status == StoryReady || dependency.Status == StoryAccepted {
				continue
			}
			if _, found := selected[item.CycleID+"/"+dependency.StoryID]; found {
				waitingForSelected = true
			} else {
				blocked = true
			}
		}
		switch {
		case blocked:
			item.Gate = "blocked_by_dependency"
		case waitingForSelected:
			item.Gate = "waiting_for_selected_dependency"
		default:
			item.Gate = "ready_for_confirmation"
		}
	}
}

func acceptanceState(storyID string, blocks []FieldBlock) string {
	for _, block := range blocks {
		if block.ID == storyID {
			return leadingToken(block.Fields["Acceptance"])
		}
	}
	return ""
}

func dependencyStatuses(story Story, snapshot CycleSnapshot) []DependencyStatus {
	statuses := make(map[string]string, len(snapshot.Acceptance))
	for _, block := range snapshot.Acceptance {
		statuses[block.ID] = leadingToken(block.Fields["Acceptance"])
	}
	dependencies := make([]DependencyStatus, 0, len(story.DependsOn))
	for _, dependency := range story.DependsOn {
		dependencies = append(dependencies, DependencyStatus{StoryID: dependency, Acceptance: statuses[dependency], Status: storyStatus(dependency, snapshot.Acceptance, snapshot.ActiveLanes)})
	}
	return dependencies
}

func hasOpenQuestions(storyText string) bool {
	ledger := sectionContent(storyText, 2, "Decision Ledger")
	questions := sectionContent(ledger, 3, "Open Questions")
	return specdoc.Meaningful(questions) && !specdoc.IsExplicitNone(questions)
}

func writeRegressionContext(output *strings.Builder, story Story, warnings []diagnostic.Issue) string {
	specRoot := filepath.Dir(filepath.Dir(story.Directory))
	if len(story.RegressionOf) > 0 {
		fmt.Fprintln(output, "> Regression references provide frozen context only; they do not transfer acceptance conclusions or supply missing evidence. The current Story requires its own observations, verification, and acceptance.")
		fmt.Fprintln(output)
	}
	for _, reference := range story.RegressionOf {
		parts := strings.SplitN(reference, "/", 2)
		referenceDirectory := filepath.Join(specRoot, parts[0], parts[1])
		storyText, err := os.ReadFile(filepath.Join(referenceDirectory, "story.md"))
		if err != nil {
			return "cannot read regression Story: " + err.Error()
		}
		outcomeText, err := os.ReadFile(filepath.Join(referenceDirectory, "acceptance", "outcome.md"))
		if err != nil {
			return "cannot read regression Outcome: " + err.Error()
		}
		outcomeFile := filepath.Join(referenceDirectory, "acceptance", "outcome.md")
		for _, warning := range warnings {
			if filepath.Clean(warning.File) == filepath.Clean(outcomeFile) {
				fmt.Fprintf(output, "> %s: %s\n\n", reference, warning.Message)
			}
		}
		writeContextSection(output, "Regression Reference "+reference+": Accepted Story", strings.TrimSpace(string(storyText)))
		writeContextSection(output, "Regression Reference "+reference+": Outcome", strings.TrimSpace(string(outcomeText)))
	}
	return ""
}

func resolveCycleDirectory(target string) (string, string) {
	absolute, err := filepath.Abs(target)
	if err != nil {
		return "", err.Error()
	}
	info, err := os.Stat(absolute)
	if os.IsNotExist(err) {
		return "", "target does not exist: " + absolute
	}
	if err != nil {
		return "", "cannot inspect target: " + err.Error()
	}
	if !info.IsDir() || !fileExists(filepath.Join(absolute, "cycle.md")) || !fileExists(filepath.Join(absolute, "cycle.md")) {
		return "", "target must be one Cycle directory: " + absolute
	}
	return absolute, ""
}

func resolveStoryDirectory(target string) (string, string) {
	absolute, err := filepath.Abs(target)
	if err != nil {
		return "", err.Error()
	}
	info, err := os.Stat(absolute)
	if os.IsNotExist(err) {
		return "", "target does not exist: " + absolute
	}
	if err != nil {
		return "", "cannot inspect target: " + err.Error()
	}
	if !info.IsDir() {
		if filepath.Base(absolute) != "story.md" {
			return "", "target must be a Story directory or story.md: " + absolute
		}
		absolute = filepath.Dir(absolute)
	}
	if !storyIDPattern.MatchString(filepath.Base(absolute)) || !fileExists(filepath.Join(absolute, "story.md")) {
		return "", "target must be a Story directory or story.md: " + absolute
	}
	return absolute, ""
}

func loadSnapshotStories(cycleDirectory string) map[string]Story {
	stories := make(map[string]Story)
	entries, _ := os.ReadDir(cycleDirectory)
	for _, entry := range entries {
		if !entry.IsDir() || !storyIDPattern.MatchString(entry.Name()) {
			continue
		}
		directory := filepath.Join(cycleDirectory, entry.Name())
		var report diagnostic.Report
		story := validateStoryFile(filepath.Join(directory, "story.md"), filepath.Base(cycleDirectory), entry.Name(), &report)
		story.Directory = directory
		story.HasOutcome = fileExists(filepath.Join(directory, "acceptance", "outcome.md"))
		stories[story.ID] = story
	}
	return stories
}

func writeDependencyContext(output *strings.Builder, story Story, blocks []FieldBlock) {
	fmt.Fprintln(output, "## Dependency State")
	fmt.Fprintln(output)
	if len(story.DependsOn) == 0 {
		fmt.Fprintln(output, "- none")
		fmt.Fprintln(output)
		return
	}
	statuses := make(map[string]string, len(blocks))
	for _, block := range blocks {
		statuses[block.ID] = block.Fields["Acceptance"]
	}
	for _, dependency := range story.DependsOn {
		fmt.Fprintf(output, "- %s: %s\n", dependency, statuses[dependency])
	}
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Before execution, confirm evidence applicability, current-environment availability, and any required human acceptance.")
	fmt.Fprintln(output)
}

func writeCurrentStoryState(output *strings.Builder, storyID string, snapshot CycleSnapshot) {
	fmt.Fprintln(output, "## Current Story State")
	fmt.Fprintf(output, "- Status: %s\n", storyStatus(storyID, snapshot.Acceptance, snapshot.ActiveLanes))
	fmt.Fprintln(output)
	for _, block := range snapshot.Acceptance {
		if block.ID == storyID {
			fmt.Fprintf(output, "- Acceptance: %s\n", block.Fields["Acceptance"])
		}
	}
	lanes := 0
	for _, lane := range snapshot.ActiveLanes {
		if lane.Fields["Story"] != storyID {
			continue
		}
		lanes++
		fmt.Fprintf(output, "- Lane %s: owner=%s, status=%s, implementation_complete=%s, next=%s\n", lane.ID, lane.Fields["Owner"], lane.Fields["Status"], lane.Fields["Implementation Complete"], lane.Fields["Next"])
		if lane.Fields["Resume Status"] != "" {
			fmt.Fprintf(output, "- Resume Status: %s\n", lane.Fields["Resume Status"])
		}
	}
	if lanes == 0 {
		fmt.Fprintln(output, "- Active Lane: none")
	}
	fmt.Fprintln(output)
}

func writeContextSection(output *strings.Builder, title, content string) {
	fmt.Fprintf(output, "## %s\n\n%s\n\n", title, strings.TrimSpace(content))
}

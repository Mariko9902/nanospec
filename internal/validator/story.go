package validator

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

var (
	briefCheckboxPattern    = regexp.MustCompile(`(?m)^\s*-\s+\[[ xX]\]`)
	orientationFieldPattern = regexp.MustCompile(`^\s*-\s+([^:：]+)[：:]\s*(.*?)\s*$`)
	outcomeResultPattern    = regexp.MustCompile(`^\s*-\s+([A-Za-z][A-Za-z0-9-]*\d+):\s+(\S+)\s*$`)
	outcomeFieldPattern     = regexp.MustCompile(`^\s+-\s+(Observed|Evidence Command|Evidence Result|Evidence Artifacts|Evidence Context):\s*(.*?)\s*$`)
	storyDigestPattern      = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

var allowedStoryAssets = map[string]struct{}{
	"ui-draft.excalidraw": {},
	"ui-draft.drawio":     {},
	"ui-draft.svg":        {},
	"ui-draft.png":        {},
}

// ValidateStoryDirectory validates a Story and returns the fields needed for dependency checks.
func ValidateStoryDirectory(storyDirectory, cycleID string, report *diagnostic.Report) Story {
	return validateStoryDirectory(storyDirectory, cycleID, true, report)
}

func validateStoryDirectory(storyDirectory, cycleID string, validateEvidenceRecords bool, report *diagnostic.Report) Story {
	directoryName := filepath.Base(storyDirectory)
	storyFile := filepath.Join(storyDirectory, "story.md")
	if !fileExists(storyFile) {
		report.AddError(storyFile, "missing story.md")
		return Story{Directory: storyDirectory, File: storyFile}
	}

	story := validateStoryFile(storyFile, cycleID, directoryName, report)
	story.Directory = storyDirectory

	briefFile := filepath.Join(storyDirectory, "implementation-brief.md")
	briefExists := fileExists(briefFile)
	if story.ImplementationBrief == "implementation-brief.md" && !briefExists {
		report.AddError(storyFile, "implementation_brief references missing implementation-brief.md")
	}
	if story.ImplementationBrief == "none" && briefExists {
		report.AddError(briefFile, "implementation-brief.md exists but story declares implementation_brief: none")
	}
	if briefExists {
		validateBrief(briefFile, report)
	}

	entries, err := os.ReadDir(storyDirectory)
	if err != nil {
		report.AddError(storyDirectory, "cannot list story directory: "+err.Error())
	}
	for _, entry := range entries {
		name := entry.Name()
		entryPath := filepath.Join(storyDirectory, name)
		if entry.IsDir() {
			if name == acceptanceDirectory {
				validateAcceptanceDirectory(entryPath, report)
				continue
			}
			report.AddError(entryPath, "unexpected directory in story directory")
			continue
		}
		if name == "story.md" || name == "implementation-brief.md" {
			continue
		}
		if name == "evidence.md" || name == "outcome.md" {
			report.AddError(entryPath, "acceptance record must be inside acceptance/")
			continue
		}
		if _, allowed := allowedStoryAssets[name]; allowed {
			continue
		}
		if strings.HasSuffix(name, ".md") {
			report.AddError(entryPath, "unsupported Markdown file in story directory")
		} else {
			report.AddWarning(entryPath, "unexpected file in story directory")
		}
	}

	outcomeFile := filepath.Join(storyDirectory, "acceptance", "outcome.md")
	evidenceFile := filepath.Join(storyDirectory, "acceptance", "evidence.md")
	story.HasOutcome = fileExists(outcomeFile)
	if story.HasOutcome && fileExists(evidenceFile) {
		report.AddError(storyDirectory, "evidence.md and outcome.md must not coexist")
	}
	if validateEvidenceRecords && fileExists(evidenceFile) {
		validateEvidence(evidenceFile, story, report)
	}
	if story.HasOutcome {
		validateOutcome(outcomeFile, story, report)
	}
	return story
}

func validateStoryFile(file, cycleID, directoryName string, report *diagnostic.Report) Story {
	content := readText(file, report)
	document := specdoc.ParseDocument(file, content, report)
	storyKeys := []string{
		"id", "cycle", "depends_on", "regression_of", "implementation_brief",
		"acceptance_policy", "acceptance_authority", "acceptance_scope", "acceptance_review",
	}
	requireKeys(file, document.Frontmatter, []string{"id", "cycle", "depends_on", "implementation_brief"}, report)
	validateAllowedKeys(file, document.Frontmatter, storyKeys, "Story", report)
	acceptancePolicy := validateAcceptancePolicy(file, document.Frontmatter, report)

	id, _ := document.Frontmatter.String("id")
	if !storyIDPattern.MatchString(id) {
		value := id
		if value == "" {
			value = "<empty>"
		}
		report.AddError(file, "invalid story id: "+value)
	}
	if id != directoryName {
		report.AddError(file, "story.id must equal directory name ("+directoryName+")")
	}
	cycle, _ := document.Frontmatter.String("cycle")
	if cycle != cycleID {
		report.AddError(file, "story.cycle must equal "+cycleID)
	}

	dependsOn := document.Frontmatter.Strings("depends_on")
	for _, dependency := range dependsOn {
		if !storyIDPattern.MatchString(dependency) {
			report.AddError(file, "story.depends_on has invalid story id: "+dependency)
		}
		if dependency == id {
			report.AddError(file, "story.depends_on must not reference itself")
		}
	}
	regressionOf := validateRegressionReferences(file, cycleID, id, document.Frontmatter.Strings("regression_of"), report)

	implementationBrief, _ := document.Frontmatter.String("implementation_brief")
	if implementationBrief != "none" && implementationBrief != "implementation-brief.md" {
		report.AddError(file, "implementation_brief must be none or implementation-brief.md")
	}

	title := validateStorySections(file, document.Body, report)
	validateAcceptanceIDFormat(file, document.Body, report)
	acceptanceIDs := extractAcceptanceIDs(document.Body)
	if len(acceptanceIDs) == 0 {
		report.AddError(file, "Acceptance must contain at least one checkbox with a stable ID")
	}
	seenAcceptance := make(map[string]struct{})
	for _, acceptanceID := range acceptanceIDs {
		if _, duplicate := seenAcceptance[acceptanceID]; duplicate {
			report.AddError(file, "duplicate Acceptance ID: "+acceptanceID)
			continue
		}
		seenAcceptance[acceptanceID] = struct{}{}
	}
	validateAcceptanceFlow(file, document.Body, report)
	validateExecutableChecks(file, document.Body, report)

	order := ""
	if match := storyIDPattern.FindStringSubmatch(id); len(match) > 1 {
		order = match[1]
	}
	return Story{
		ID:                  id,
		Title:               title,
		Order:               order,
		File:                file,
		DependsOn:           dependsOn,
		RegressionOf:        regressionOf,
		ImplementationBrief: implementationBrief,
		AcceptancePolicy:    acceptancePolicy,
		AcceptanceIDs:       acceptanceIDs,
	}
}

func validateRegressionReferences(file, cycleID, storyID string, references []string, report *diagnostic.Report) []string {
	seen := make(map[string]struct{}, len(references))
	valid := make([]string, 0, len(references))
	specRoot := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	for _, reference := range references {
		match := regressionRefPattern.FindStringSubmatch(reference)
		if match == nil {
			report.AddError(file, "regression_of has invalid reference: "+reference)
			continue
		}
		if _, duplicate := seen[reference]; duplicate {
			report.AddError(file, "regression_of has duplicate reference: "+reference)
			continue
		}
		seen[reference] = struct{}{}
		if match[1] == cycleID && match[2] == storyID {
			report.AddError(file, "regression_of must not reference the current Story")
			continue
		}
		referenceDirectory := filepath.Join(specRoot, match[1], match[2])
		referenceStory := filepath.Join(referenceDirectory, "story.md")
		referenceOutcome := filepath.Join(referenceDirectory, "acceptance", "outcome.md")
		if !fileExists(referenceStory) || !fileExists(referenceOutcome) {
			report.AddError(file, "regression_of must reference a Story with outcome.md: "+reference)
			continue
		}
		storySource := readText(referenceStory, report)
		storyDocument := specdoc.ParseDocument(referenceStory, storySource, report)
		referenceModel := Story{
			ID:               match[2],
			File:             referenceStory,
			AcceptancePolicy: validateAcceptancePolicy(referenceStory, storyDocument.Frontmatter, report),
			AcceptanceIDs:    extractAcceptanceIDs(storyDocument.Body),
		}
		outcomeSource := readText(referenceOutcome, report)
		if match[1] == cycleID || !validateHistoricalRegressionOutcome(referenceOutcome, outcomeSource, referenceModel, report) {
			validateOutcomeSource(referenceOutcome, outcomeSource, referenceModel, report)
		}
		valid = append(valid, reference)
	}
	return valid
}

func validateStorySections(file, body string, report *diagnostic.Report) string {
	storySections := []string{"Orientation", "Definition", "Out of Scope", "Decision Ledger", "Acceptance"}
	requireSections(file, body, 2, []string{"Orientation", "Definition", "Acceptance"}, report)
	validateAllowedSections(file, body, 2, storySections, "Story", report)
	requireSectionOrder(file, body, 2, storySections, "Story", report)
	for _, name := range []string{"Orientation", "Definition", "Acceptance"} {
		requireMeaningfulSection(file, body, name, false, report)
	}
	title := validateOrientation(file, sectionContent(body, 2, "Orientation"), report)
	for _, optional := range []string{"Out of Scope", "Decision Ledger"} {
		if content, ok := section(body, 2, optional); ok {
			warnEmptyOptionalSection(file, content.Content, optional, report)
		}
	}

	definition := sectionContent(body, 2, "Definition")
	definitionSections := []string{"Concepts", "Description", "Business Logic", "UI Draft", "Integration Notes", "Applicable Rules", "Considerations"}
	validateAllowedSections(file, definition, 3, definitionSections, "Definition", report)
	requireSectionOrder(file, definition, 3, definitionSections, "Definition", report)
	for _, misplaced := range []string{"Concepts", "Description", "Applicable Rules", "Business Logic", "Considerations", "Integration Notes", "UI Draft", "State Model", "Flow Diagram", "Constraints"} {
		if hasSection(body, 2, misplaced) {
			report.AddError(file, misplaced+" must be nested under Definition")
		}
	}
	if hasSection(body, 3, "Concepts") && !hasSection(definition, 3, "Concepts") {
		report.AddError(file, "Concepts must be nested under Definition")
	}
	for _, heading := range specdoc.ParseMarkdown(body).Headings() {
		if heading.Name == "Concepts" && heading.Level != 2 && heading.Level != 3 {
			report.AddError(file, "Concepts must be a level-3 subsection of Definition")
		}
	}
	description, hasDescription := section(definition, 3, "Description")
	if !hasDescription {
		report.AddError(file, "Definition missing subsection: Description")
	} else if !specdoc.Meaningful(description.Content) {
		report.AddError(file, "Definition subsection is empty: Description")
	}
	for _, optional := range []string{"Concepts", "Business Logic", "Considerations", "Applicable Rules", "Integration Notes", "UI Draft"} {
		if content, ok := section(definition, 3, optional); ok {
			warnEmptyOptionalSection(file, content.Content, optional, report)
		}
	}

	businessLogic := sectionContent(definition, 3, "Business Logic")
	businessLogicSections := []string{"State Model", "Flow Diagram", "Constraints"}
	validateAllowedSections(file, businessLogic, 4, businessLogicSections, "Business Logic", report)
	requireSectionOrder(file, businessLogic, 4, businessLogicSections, "Business Logic", report)
	for _, misplaced := range []string{"State Model", "Flow Diagram", "Constraints"} {
		if hasSection(definition, 3, misplaced) {
			report.AddError(file, misplaced+" must be nested under Business Logic")
		}
		if hasSection(definition, 4, misplaced) && !hasSection(businessLogic, 4, misplaced) {
			report.AddError(file, misplaced+" must be nested under Business Logic")
		}
	}
	for _, optional := range []string{"State Model", "Flow Diagram", "Constraints"} {
		if content, ok := section(businessLogic, 4, optional); ok {
			warnEmptyOptionalSection(file, content.Content, optional, report)
		}
	}

	ledger := sectionContent(body, 2, "Decision Ledger")
	ledgerSections := []string{"Confirmed Facts", "Decisions", "Assumptions", "Open Questions"}
	validateAllowedSections(file, ledger, 3, ledgerSections, "Decision Ledger", report)
	hasLedgerPart := false
	for _, name := range ledgerSections {
		if hasSection(ledger, 3, name) {
			hasLedgerPart = true
			break
		}
	}

	acceptance := sectionContent(body, 2, "Acceptance")
	acceptanceSections := []string{"Acceptance Flow", "Executable Checks"}
	validateAllowedSections(file, acceptance, 3, acceptanceSections, "Acceptance", report)
	requireSectionOrder(file, acceptance, 3, acceptanceSections, "Acceptance", report)
	if specdoc.Meaningful(ledger) && !specdoc.IsExplicitNone(ledger) && !hasLedgerPart {
		report.AddError(file, "Decision Ledger must contain at least one populated subsection")
	}
	requireSectionOrder(file, ledger, 3, ledgerSections, "Decision Ledger", report)
	for _, name := range ledgerSections {
		if hasSection(body, 3, name) && !hasSection(ledger, 3, name) {
			report.AddError(file, name+" must be nested under Decision Ledger")
		}
		if content, ok := section(ledger, 3, name); ok {
			warnEmptyOptionalSection(file, content.Content, name, report)
		}
	}
	return title
}

func validateOrientation(file, content string, report *diagnostic.Report) string {
	required := []string{"Title", "Scenario"}
	allowed := map[string]struct{}{
		"Title":    {},
		"Scenario": {},
	}
	found := make(map[string]bool, len(required))
	values := make(map[string]string, len(required))

	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		matches := orientationFieldPattern.FindStringSubmatch(line)
		if matches == nil {
			report.AddError(file, "Orientation must contain only Title and Scenario labeled bullets")
			continue
		}
		name := strings.TrimSpace(matches[1])
		value := strings.TrimSpace(matches[2])
		if _, ok := allowed[name]; !ok {
			report.AddError(file, "Orientation has unsupported field: "+name)
			continue
		}
		if found[name] {
			report.AddError(file, "Orientation has duplicate field: "+name)
			continue
		}
		found[name] = true
		values[name] = value
		if !specdoc.Meaningful(value) || specdoc.IsExplicitNone(value) {
			report.AddError(file, "Orientation field must contain meaningful content: "+name)
		}
	}

	for _, name := range required {
		if !found[name] {
			report.AddError(file, "Orientation missing field: "+name)
		}
	}
	return values["Title"]
}

func validateBrief(file string, report *diagnostic.Report) {
	body := readText(file, report)
	markdown := specdoc.ParseMarkdown(body)
	if _, ok := markdown.Section(1, "Implementation Brief"); !ok {
		report.AddError(file, "Implementation Brief must contain '# Implementation Brief'")
	}
	allowed := map[string]struct{}{"Repository Findings": {}, "Candidate Directions": {}}
	for _, heading := range markdown.Headings() {
		if heading.Level != 2 {
			continue
		}
		if _, ok := allowed[heading.Name]; !ok {
			report.AddError(file, "Implementation Brief has unsupported section: "+heading.Name)
		}
	}
	briefSections := []string{"Repository Findings", "Candidate Directions"}
	requireSectionOrder(file, body, 2, briefSections, "Implementation Brief", report)
	if !hasSection(body, 2, "Repository Findings") {
		report.AddError(file, "Implementation Brief missing required section: Repository Findings")
	}
	for _, name := range briefSections {
		if content, ok := section(body, 2, name); ok {
			if !specdoc.Meaningful(content.Content) || specdoc.IsExplicitNone(content.Content) {
				report.AddError(file, "Implementation Brief section is empty: "+name)
			}
		}
	}
	if briefCheckboxPattern.MatchString(body) {
		report.AddError(file, "Implementation Brief must not contain completion checkboxes")
	}
}

type acceptanceResult struct {
	id                string
	status            string
	observed          string
	evidenceCommand   string
	evidenceResult    string
	evidenceArtifacts string
	evidenceContext   string
}

func validateOutcome(file string, story Story, report *diagnostic.Report) {
	validateOutcomeSource(file, readText(file, report), story, report)
}

func validateOutcomeSource(file, source string, story Story, report *diagnostic.Report) {
	document := specdoc.ParseDocument(file, source, report)
	outcomeKeys := []string{
		"story", "spec_rev", "story_digest",
		"implemented_by", "verified_by", "verification_mode", "verified_at", "verified_acceptance",
		"first_pass", "rework_count", "spec_change_count",
		"accepted_by", "accepted_at", "acceptance_method",
	}
	requireKeys(file, document.Frontmatter, outcomeKeys, report)
	validateAllowedKeys(file, document.Frontmatter, outcomeKeys, "Outcome", report)
	validateOutcomeBinding(file, document.Frontmatter, story.File, report)
	method := validateOutcomeAcceptance(file, document.Frontmatter, story.AcceptancePolicy, report)
	validateOutcomeVerification(file, document.Frontmatter, story.AcceptanceIDs, method, report)
	outcomeStory, _ := document.Frontmatter.String("story")
	if outcomeStory != story.ID {
		report.AddError(file, "outcome.story must equal "+story.ID)
	}
	if document.Frontmatter.Has("accepted_at") {
		acceptedAt, ok := document.Frontmatter.String("accepted_at")
		if !ok || !isISOTimestamp(acceptedAt) {
			report.AddError(file, "outcome.accepted_at must be an ISO-8601 timestamp with timezone")
		}
	}
	requireSections(file, document.Body, 2, []string{"Acceptance Results"}, report)
	validateAllowedSections(file, document.Body, 2, []string{"Acceptance Results"}, "Outcome", report)
	requireMeaningfulSection(file, document.Body, "Acceptance Results", false, report)
	validateOutcomeResults(file, sectionContent(document.Body, 2, "Acceptance Results"), story.AcceptanceIDs, report)
}

func validateOutcomeBinding(file string, frontmatter specdoc.Frontmatter, storyFile string, report *diagnostic.Report) {
	revision, hasRevision := frontmatter.String("spec_rev")
	digest, hasDigest := frontmatter.String("story_digest")
	if !hasRevision || !hasDigest {
		return
	}
	if !revisionPattern.MatchString(revision) {
		report.AddError(file, "outcome.spec_rev must match rev-N with N >= 1")
	}
	if !storyDigestPattern.MatchString(digest) {
		report.AddError(file, "outcome.story_digest must be sha256 followed by 64 lowercase hexadecimal characters")
		return
	}
	storySource, err := os.ReadFile(storyFile)
	if err != nil {
		report.AddError(file, "cannot read bound story.md: "+err.Error())
		return
	}
	if storyContentDigest(storySource) != digest {
		report.AddError(file, "outcome.story_digest does not match story.md")
	}
}

func validateOutcomeVerification(
	file string,
	frontmatter specdoc.Frontmatter,
	expected []string,
	method AcceptancePolicy,
	report *diagnostic.Report,
) {
	implementedBy, implementedFound := frontmatter.String("implemented_by")
	if !hasText(implementedBy, implementedFound) {
		report.AddError(file, "implemented_by must identify the implementer")
	}
	verifiedBy, verifiedFound := frontmatter.String("verified_by")
	if !hasText(verifiedBy, verifiedFound) {
		report.AddError(file, "verified_by must identify the verifier")
	}
	modeText, modeFound := frontmatter.String("verification_mode")
	mode := VerificationMode(modeText)
	if !modeFound || (mode != VerificationSelf && mode != VerificationIndependent) {
		report.AddError(file, "verification_mode must be self or independent")
	}
	if mode == VerificationSelf && implementedBy != verifiedBy {
		report.AddError(file, "self verification requires implemented_by and verified_by to match")
	}
	if mode == VerificationIndependent && implementedBy == verifiedBy {
		report.AddError(file, "independent verification requires a verifier other than the implementer")
	}
	verifiedAt, verifiedAtFound := frontmatter.String("verified_at")
	if !verifiedAtFound || !isISOTimestamp(verifiedAt) {
		report.AddError(file, "verified_at must be an ISO-8601 timestamp with timezone")
	}
	validateVerificationCoverage(file, frontmatter.Strings("verified_acceptance"), expected, report)
	validateOutcomeMetrics(file, frontmatter, report)
	if method == AcceptanceVerified && mode != VerificationIndependent {
		report.AddError(file, "verified acceptance requires independent verification")
	}
}

func validateOutcomeMetrics(file string, frontmatter specdoc.Frontmatter, report *diagnostic.Report) {
	firstPassText, firstPassFound := frontmatter.String("first_pass")
	firstPass := false
	if firstPassFound {
		if firstPassText != "true" && firstPassText != "false" {
			report.AddError(file, "first_pass must be true or false")
		} else {
			firstPass = firstPassText == "true"
		}
	}
	reworkCount, reworkFound := nonNegativeField(file, frontmatter, "rework_count", report)
	_, _ = nonNegativeField(file, frontmatter, "spec_change_count", report)
	if firstPassFound && (firstPassText == "true" || firstPassText == "false") && reworkFound {
		if firstPass != (reworkCount == 0) {
			report.AddError(file, "first_pass must be true exactly when rework_count is 0")
		}
	}
}

func nonNegativeField(file string, frontmatter specdoc.Frontmatter, field string, report *diagnostic.Report) (int, bool) {
	text, found := frontmatter.String(field)
	if !found {
		return 0, false
	}
	value, err := strconv.Atoi(text)
	if err != nil || value < 0 {
		report.AddError(file, field+" must be a non-negative integer")
		return 0, false
	}
	return value, true
}

func validateVerificationCoverage(file string, actual, expected []string, report *diagnostic.Report) {
	expectedSet := make(map[string]struct{}, len(expected))
	for _, id := range expected {
		expectedSet[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(actual))
	for _, id := range actual {
		if _, duplicate := seen[id]; duplicate {
			report.AddError(file, "verified_acceptance has duplicate Acceptance ID: "+id)
		}
		seen[id] = struct{}{}
		if _, found := expectedSet[id]; !found {
			report.AddError(file, "verified_acceptance references unknown Acceptance ID: "+id)
		}
	}
	for _, id := range expected {
		if _, found := seen[id]; !found {
			report.AddError(file, "verified_acceptance missing Acceptance ID: "+id)
		}
	}
}

func validateOutcomeResults(file, content string, expected []string, report *diagnostic.Report) {
	results := make([]acceptanceResult, 0)
	current := -1
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if match := outcomeResultPattern.FindStringSubmatch(line); match != nil {
			results = append(results, acceptanceResult{id: match[1], status: match[2]})
			current = len(results) - 1
			continue
		}
		if match := outcomeFieldPattern.FindStringSubmatch(line); match != nil && current >= 0 {
			switch match[1] {
			case "Observed":
				results[current].observed = match[2]
			case "Evidence Command":
				results[current].evidenceCommand = match[2]
			case "Evidence Result":
				results[current].evidenceResult = match[2]
			case "Evidence Artifacts":
				results[current].evidenceArtifacts = match[2]
			case "Evidence Context":
				results[current].evidenceContext = match[2]
			}
		}
	}

	expectedSet := make(map[string]struct{}, len(expected))
	for _, id := range expected {
		expectedSet[id] = struct{}{}
	}
	seen := make(map[string]struct{})
	for _, result := range results {
		if !canonicalAcceptanceID.MatchString(result.id) {
			report.AddError(file, "Acceptance Results has invalid Acceptance ID: "+result.id)
		}
		if _, duplicate := seen[result.id]; duplicate {
			report.AddError(file, "Acceptance Results has duplicate result: "+result.id)
		}
		seen[result.id] = struct{}{}
		if _, known := expectedSet[result.id]; !known {
			report.AddError(file, "Acceptance Results references unknown Acceptance ID: "+result.id)
		}
		if result.status != "confirmed" {
			report.AddError(file, "Acceptance Results must confirm "+result.id)
		}
		if !specdoc.Meaningful(result.observed) {
			report.AddError(file, "Acceptance Results "+result.id+" missing Observed")
		}
		fields := []struct {
			name  string
			value string
		}{
			{name: "Evidence Command", value: result.evidenceCommand},
			{name: "Evidence Result", value: result.evidenceResult},
			{name: "Evidence Artifacts", value: result.evidenceArtifacts},
			{name: "Evidence Context", value: result.evidenceContext},
		}
		for _, field := range fields {
			if !specdoc.Meaningful(field.value) {
				report.AddError(file, "Acceptance Results "+result.id+" missing "+field.name)
			}
		}
	}
	for _, id := range expected {
		if _, found := seen[id]; !found {
			report.AddError(file, "Acceptance Results missing confirmed result: "+id)
		}
	}
}

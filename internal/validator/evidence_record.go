package validator

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

func validateEvidence(file string, story Story, report *diagnostic.Report) {
	_, _ = parseEvidenceRecord(file, readText(file, report), story, report)
}

func validateEvidenceSource(file string, story Story, source string) ValidationResult {
	var report diagnostic.Report
	_, _ = parseEvidenceRecord(file, source, story, &report)
	return ValidationResult{Errors: report.Errors(), Warnings: report.Warnings()}
}

func loadEvidenceRecord(file string, story Story) (evidenceRecord, string, ValidationResult) {
	source, err := os.ReadFile(file)
	if err != nil {
		return evidenceRecord{}, "", ValidationResult{InputError: "cannot read evidence.md: " + err.Error()}
	}
	var report diagnostic.Report
	record, _ := parseEvidenceRecord(file, string(source), story, &report)
	return record, string(source), ValidationResult{Errors: report.Errors(), Warnings: report.Warnings()}
}

func loadEvidenceRecordForRevision(file string, story Story) (evidenceRecord, string, ValidationResult) {
	source, err := os.ReadFile(file)
	if err != nil {
		return evidenceRecord{}, "", ValidationResult{InputError: "cannot read evidence.md: " + err.Error()}
	}
	previousStory := story
	previousStory.AcceptanceIDs = evidenceResultIDs(string(source))
	var report diagnostic.Report
	record, _ := parseEvidenceRecord(file, string(source), previousStory, &report)
	return record, string(source), ValidationResult{Errors: report.Errors(), Warnings: report.Warnings()}
}

func evidenceResultIDs(source string) []string {
	ids := make([]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n") {
		if match := outcomeResultPattern.FindStringSubmatch(line); match != nil {
			ids = append(ids, match[1])
		}
	}
	return ids
}

func parseEvidenceRecord(file, source string, story Story, report *diagnostic.Report) (evidenceRecord, bool) {
	document := specdoc.ParseDocument(file, source, report)
	required := []string{
		"story", "spec_rev", "implemented_by", "verification_status", "verified_by", "verification_mode",
		"verified_at", "verified_acceptance", "first_pass", "rework_count", "spec_change_count",
	}
	requireKeys(file, document.Frontmatter, required, report)
	validateAllowedKeys(file, document.Frontmatter, required, "Evidence", report)
	requireSections(file, document.Body, 2, []string{"Acceptance Results"}, report)
	validateAllowedSections(file, document.Body, 2, []string{"Acceptance Results"}, "Evidence", report)
	record := evidenceRecord{Results: make(map[string]acceptanceResult)}
	record.StoryID, _ = document.Frontmatter.String("story")
	record.SpecRevision, _ = document.Frontmatter.String("spec_rev")
	record.ImplementedBy, _ = document.Frontmatter.String("implemented_by")
	record.VerificationStatus, _ = document.Frontmatter.String("verification_status")
	record.VerifiedBy, _ = document.Frontmatter.String("verified_by")
	record.VerificationMode, _ = document.Frontmatter.String("verification_mode")
	record.VerifiedAt, _ = document.Frontmatter.String("verified_at")
	record.VerifiedAcceptance = document.Frontmatter.Strings("verified_acceptance")
	record.ReworkCount, _ = nonNegativeField(file, document.Frontmatter, "rework_count", report)
	record.SpecChangeCount, _ = nonNegativeField(file, document.Frontmatter, "spec_change_count", report)
	firstPass, firstPassFound := document.Frontmatter.String("first_pass")
	if firstPassFound && firstPass != strconv.FormatBool(record.ReworkCount == 0) {
		report.AddError(file, "first_pass must be true exactly when rework_count is 0")
	}
	if record.StoryID != story.ID {
		report.AddError(file, "evidence.story must equal "+story.ID)
	}
	if !revisionPattern.MatchString(record.SpecRevision) {
		report.AddError(file, "evidence.spec_rev must match rev-N with N >= 1")
	}
	if !executorIDPattern.MatchString(record.ImplementedBy) {
		report.AddError(file, "implemented_by must identify the implementer")
	}
	statuses := map[string]struct{}{"pending": {}, "passed": {}, "rework": {}}
	if _, found := statuses[record.VerificationStatus]; !found {
		report.AddError(file, "verification_status must be pending, passed or rework")
	}
	record.Context = evidenceContext(document.Body)
	if !singleLineText(record.Context) {
		report.AddError(file, "Acceptance Results must contain one verification context line")
	} else if inputError := validateObservationContext(record.Context); inputError != "" {
		report.AddError(file, inputError)
	}
	record.Results = parseEvidenceResults(sectionContent(document.Body, 2, "Acceptance Results"), story, record.VerificationStatus, file, report)
	validateEvidenceVerification(file, record, story, report)
	return record, len(report.Errors()) == 0
}

func parseEvidenceResults(content string, story Story, verificationStatus, file string, report *diagnostic.Report) map[string]acceptanceResult {
	results := make(map[string]acceptanceResult)
	current := ""
	allowed := map[string]struct{}{"observed": {}, "stale": {}, "confirmed": {}}
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if match := outcomeResultPattern.FindStringSubmatch(line); match != nil {
			current = match[1]
			if _, duplicate := results[current]; duplicate {
				report.AddError(file, "Acceptance Results has duplicate result: "+current)
			}
			results[current] = acceptanceResult{id: current, status: match[2]}
			continue
		}
		if match := outcomeFieldPattern.FindStringSubmatch(line); match != nil && current != "" {
			item := results[current]
			switch match[1] {
			case "Observed":
				item.observed = match[2]
			case "Evidence Command":
				item.evidenceCommand = match[2]
			case "Evidence Result":
				item.evidenceResult = match[2]
			case "Evidence Artifacts":
				item.evidenceArtifacts = match[2]
			case "Evidence Context":
				item.evidenceContext = match[2]
			}
			results[current] = item
		}
	}
	for id, item := range results {
		if !canonicalAcceptanceID.MatchString(id) {
			report.AddError(file, "Acceptance Results has invalid Acceptance ID: "+id)
		}
		if !containsString(story.AcceptanceIDs, id) {
			report.AddError(file, "Acceptance Results references unknown Acceptance ID: "+id)
		}
		if _, found := allowed[item.status]; !found {
			report.AddError(file, "Acceptance Results has invalid status for "+id+": "+item.status)
		}
		for _, field := range []struct{ name, value string }{
			{"Observed", item.observed}, {"Evidence Command", item.evidenceCommand},
			{"Evidence Result", item.evidenceResult}, {"Evidence Artifacts", item.evidenceArtifacts},
			{"Evidence Context", item.evidenceContext},
		} {
			if !singleLineText(field.value) {
				report.AddError(file, "Acceptance Results "+id+" missing "+field.name)
			}
		}
		if singleLineText(item.evidenceContext) {
			if inputError := validateObservationContext(item.evidenceContext); inputError != "" {
				report.AddError(file, "Acceptance Results "+id+": "+inputError)
			}
		}
		if verificationStatus == "passed" && item.status != "confirmed" {
			report.AddError(file, "passed evidence must confirm "+id)
		}
	}
	if verificationStatus == "passed" {
		for _, id := range story.AcceptanceIDs {
			if _, found := results[id]; !found {
				report.AddError(file, "passed evidence missing Acceptance result: "+id)
			}
		}
	}
	return results
}

func validateEvidenceVerification(file string, record evidenceRecord, story Story, report *diagnostic.Report) {
	if record.VerificationStatus == "pending" {
		if record.VerifiedBy != "none" || record.VerificationMode != "none" || record.VerifiedAt != "none" || len(record.VerifiedAcceptance) != 0 {
			report.AddError(file, "pending evidence must not declare completed verification")
		}
		return
	}
	if !executorIDPattern.MatchString(record.VerifiedBy) {
		report.AddError(file, "verified_by must identify the verifier")
	}
	mode := VerificationMode(record.VerificationMode)
	if mode != VerificationSelf && mode != VerificationIndependent {
		report.AddError(file, "verification_mode must be self or independent")
	}
	if mode == VerificationSelf && record.ImplementedBy != record.VerifiedBy {
		report.AddError(file, "self verification requires implemented_by and verified_by to match")
	}
	if mode == VerificationIndependent && record.ImplementedBy == record.VerifiedBy {
		report.AddError(file, "independent verification requires a verifier other than the implementer")
	}
	if !isISOTimestamp(record.VerifiedAt) {
		report.AddError(file, "verified_at must be an ISO-8601 timestamp with timezone")
	}
	if record.VerificationStatus == "passed" {
		validateVerificationCoverage(file, record.VerifiedAcceptance, story.AcceptanceIDs, report)
	} else {
		validateKnownAcceptanceIDs(file, record.VerifiedAcceptance, story.AcceptanceIDs, report)
	}
}

func validateKnownAcceptanceIDs(file string, actual, expected []string, report *diagnostic.Report) {
	seen := make(map[string]struct{}, len(actual))
	for _, id := range actual {
		if _, duplicate := seen[id]; duplicate {
			report.AddError(file, "verified_acceptance has duplicate Acceptance ID: "+id)
		}
		seen[id] = struct{}{}
		if !containsString(expected, id) {
			report.AddError(file, "verified_acceptance references unknown Acceptance ID: "+id)
		}
	}
}

func renderEvidenceRecord(record evidenceRecord, order []string) string {
	var output strings.Builder
	fmt.Fprintln(&output, "---")
	fmt.Fprintln(&output, "story: "+record.StoryID)
	fmt.Fprintln(&output, "spec_rev: "+record.SpecRevision)
	fmt.Fprintln(&output, "implemented_by: "+record.ImplementedBy)
	fmt.Fprintln(&output, "verification_status: "+record.VerificationStatus)
	fmt.Fprintln(&output, "verified_by: "+record.VerifiedBy)
	fmt.Fprintln(&output, "verification_mode: "+record.VerificationMode)
	fmt.Fprintln(&output, "verified_at: "+record.VerifiedAt)
	fmt.Fprintln(&output, "verified_acceptance: ["+strings.Join(record.VerifiedAcceptance, ", ")+"]")
	fmt.Fprintln(&output, "first_pass: "+strconv.FormatBool(record.ReworkCount == 0))
	fmt.Fprintln(&output, "rework_count: "+strconv.Itoa(record.ReworkCount))
	fmt.Fprintln(&output, "spec_change_count: "+strconv.Itoa(record.SpecChangeCount))
	fmt.Fprintln(&output, "---")
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "## Acceptance Results")
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "Verification Context: "+record.Context)
	for _, id := range order {
		item, found := record.Results[id]
		if !found {
			continue
		}
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "- "+id+": "+item.status)
		fmt.Fprintln(&output, "  - Observed: "+item.observed)
		fmt.Fprintln(&output, "  - Evidence Command: "+item.evidenceCommand)
		fmt.Fprintln(&output, "  - Evidence Result: "+item.evidenceResult)
		fmt.Fprintln(&output, "  - Evidence Artifacts: "+item.evidenceArtifacts)
		fmt.Fprintln(&output, "  - Evidence Context: "+item.evidenceContext)
	}
	return output.String()
}

func renderOutcomeRecord(record evidenceRecord, order []string, request AcceptanceRequest, storyDigest string) string {
	var output strings.Builder
	fmt.Fprintln(&output, "---")
	fmt.Fprintln(&output, "story: "+record.StoryID)
	fmt.Fprintln(&output, "spec_rev: "+record.SpecRevision)
	fmt.Fprintln(&output, "story_digest: "+storyDigest)
	fmt.Fprintln(&output, "implemented_by: "+record.ImplementedBy)
	fmt.Fprintln(&output, "verified_by: "+record.VerifiedBy)
	fmt.Fprintln(&output, "verification_mode: "+record.VerificationMode)
	fmt.Fprintln(&output, "verified_at: "+record.VerifiedAt)
	fmt.Fprintln(&output, "verified_acceptance: ["+strings.Join(record.VerifiedAcceptance, ", ")+"]")
	fmt.Fprintln(&output, "first_pass: "+strconv.FormatBool(record.ReworkCount == 0))
	fmt.Fprintln(&output, "rework_count: "+strconv.Itoa(record.ReworkCount))
	fmt.Fprintln(&output, "spec_change_count: "+strconv.Itoa(record.SpecChangeCount))
	fmt.Fprintln(&output, "accepted_by: "+request.AcceptedBy)
	fmt.Fprintln(&output, "accepted_at: "+request.At.Format(time.RFC3339))
	fmt.Fprintln(&output, "acceptance_method: "+string(request.Method))
	fmt.Fprintln(&output, "---")
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "## Acceptance Results")
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "Verification Context: "+record.Context)
	for _, id := range order {
		item := record.Results[id]
		fmt.Fprintln(&output)
		fmt.Fprintln(&output, "- "+id+": confirmed")
		fmt.Fprintln(&output, "  - Observed: "+item.observed)
		fmt.Fprintln(&output, "  - Evidence Command: "+item.evidenceCommand)
		fmt.Fprintln(&output, "  - Evidence Result: "+item.evidenceResult)
		fmt.Fprintln(&output, "  - Evidence Artifacts: "+item.evidenceArtifacts)
		fmt.Fprintln(&output, "  - Evidence Context: "+item.evidenceContext)
	}
	return output.String()
}

func newEvidenceRecord(story Story, revision string, request EvidenceCaptureRequest, changes int) evidenceRecord {
	return evidenceRecord{
		StoryID: story.ID, SpecRevision: revision, ImplementedBy: request.ImplementedBy,
		VerificationStatus: "pending", VerifiedBy: "none", VerificationMode: "none", VerifiedAt: "none",
		VerifiedAcceptance: []string{}, SpecChangeCount: changes, Context: request.Context,
		Results: make(map[string]acceptanceResult),
	}
}

func evidenceContext(body string) string {
	for _, line := range strings.Split(strings.ReplaceAll(sectionContent(body, 2, "Acceptance Results"), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Verification Context:") {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Verification Context:"))
		}
	}
	return ""
}

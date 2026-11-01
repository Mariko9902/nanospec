package validator

import (
	"strings"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

// validateHistoricalRegressionOutcome reads the original human acceptance format only for external regression references.
func validateHistoricalRegressionOutcome(file, source string, story Story, report *diagnostic.Report) bool {
	document := specdoc.ParseDocument(file, source, report)
	keys := document.Frontmatter.Keys()
	if len(keys) != 3 || !document.Frontmatter.Has("story") || !document.Frontmatter.Has("accepted_by") || !document.Frontmatter.Has("accepted_at") {
		return validateBoundRegressionContext(file, source, story, report)
	}
	id, _ := document.Frontmatter.String("story")
	actor, _ := document.Frontmatter.String("accepted_by")
	acceptedAt, _ := document.Frontmatter.String("accepted_at")
	if id != story.ID || !specdoc.Meaningful(actor) || specdoc.IsExplicitNone(actor) || !isISOTimestamp(acceptedAt) || story.AcceptancePolicy != AcceptanceHuman || len(story.AcceptanceIDs) == 0 {
		report.AddError(file, "historical regression Outcome must identify the Story, human acceptor and acceptance timestamp")
		return true
	}
	requireSections(file, document.Body, 2, []string{"Acceptance Results"}, report)
	validateAllowedSections(file, document.Body, 2, []string{"Acceptance Results", "Delivery Evidence"}, "Historical Outcome", report)
	content := sectionContent(document.Body, 2, "Acceptance Results")
	seen := make(map[string]bool)
	current, observed, evidence := "", false, false
	finish := func() {
		if current != "" && (!observed || !evidence) {
			report.AddError(file, "historical Acceptance Results "+current+" requires Observed and Evidence")
		}
	}
	for _, line := range strings.Split(content, "\n") {
		if match := outcomeResultPattern.FindStringSubmatch(line); match != nil {
			finish()
			current, observed, evidence = match[1], false, false
			if seen[current] || match[2] != "confirmed" || !containsAcceptance(story.AcceptanceIDs, current) {
				report.AddError(file, "historical Acceptance Results has invalid or duplicate confirmation: "+current)
			}
			seen[current] = true
		} else if current != "" {
			text := strings.TrimSpace(line)
			if strings.HasPrefix(text, "- Observed:") {
				observed = specdoc.Meaningful(strings.TrimSpace(strings.TrimPrefix(text, "- Observed:")))
			}
			if strings.HasPrefix(text, "- Evidence:") {
				evidence = specdoc.Meaningful(strings.TrimSpace(strings.TrimPrefix(text, "- Evidence:")))
			}
		}
	}
	finish()
	for _, id := range story.AcceptanceIDs {
		if !seen[id] {
			report.AddError(file, "historical Acceptance Results missing confirmed result: "+id)
		}
	}
	report.AddWarning(file, "historical human acceptance reference: no content digest or structured verification evidence; use original record as historical context only")
	return true
}

func containsAcceptance(ids []string, wanted string) bool {
	for _, id := range ids {
		if id == wanted {
			return true
		}
	}
	return false
}

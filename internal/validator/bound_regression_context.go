package validator

import (
	"strings"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

func validateBoundRegressionContext(file, source string, story Story, report *diagnostic.Report) bool {
	var checked diagnostic.Report
	document := specdoc.ParseDocument(file, source, &checked)
	method, _ := document.Frontmatter.String("acceptance_method")
	actor, _ := document.Frontmatter.String("accepted_by")
	if checked.HasErrors() || story.AcceptancePolicy != AcceptanceHuman || method != string(AcceptanceHuman) ||
		!specdoc.Meaningful(actor) || specdoc.IsExplicitNone(actor) || len(story.AcceptanceIDs) == 0 {
		return false
	}
	for _, line := range strings.Split(document.Body, "\n") {
		if field := outcomeFieldPattern.FindStringSubmatch(line); field != nil &&
			(field[1] == "Evidence Artifacts" || field[1] == "Evidence Context") {
			return false
		}
	}
	validateOutcomeSource(file, source, story, &checked)
	missing := make(map[string]bool, 2*len(story.AcceptanceIDs))
	for _, id := range story.AcceptanceIDs {
		missing["Acceptance Results "+id+" missing Evidence Artifacts"] = true
		missing["Acceptance Results "+id+" missing Evidence Context"] = true
	}
	for _, issue := range checked.Errors() {
		if !missing[issue.Message] {
			return false
		}
		delete(missing, issue.Message)
	}
	if len(missing) != 0 {
		return false
	}
	for _, issue := range checked.Warnings() {
		report.AddWarning(issue.File, issue.Message)
	}
	report.AddWarning(file, "frozen external human acceptance is read-only regression context: Evidence Artifacts and Evidence Context absent for "+
		strings.Join(story.AcceptanceIDs, ", ")+"; missing evidence is not verified or inherited by the current Story")
	return true
}

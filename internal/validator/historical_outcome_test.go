package validator

import (
	"strings"
	"testing"

	"nanospec/internal/diagnostic"
)

const historicalOutcome = "---\nstory: story-01-base\naccepted_by: chen\naccepted_at: 2026-09-08T11:10:25+08:00\n---\n\n## Acceptance Results\n\n- A1: confirmed\n  - Observed: The saved result was checked.\n  - Evidence: The acceptor confirmed the existing record.\n- A2: confirmed\n  - Observed: The query result was checked.\n  - Evidence: The acceptor confirmed the existing page.\n\n## Delivery Evidence\n\n- The original delivery baseline was saved.\n"

func TestHistoricalRegressionOutcomePreservesOriginalAcceptance(t *testing.T) {
	story := Story{ID: "story-01-base", AcceptancePolicy: AcceptanceHuman, AcceptanceIDs: []string{"A1", "A2"}}
	var report diagnostic.Report
	if !validateHistoricalRegressionOutcome("outcome.md", historicalOutcome, story, &report) || report.HasErrors() || len(report.Warnings()) != 1 {
		t.Fatalf("report = %#v", report.Issues())
	}
}

func TestHistoricalRegressionOutcomeRejectsIncompleteOrFalseAcceptance(t *testing.T) {
	story := Story{ID: "story-01-base", AcceptancePolicy: AcceptanceHuman, AcceptanceIDs: []string{"A1", "A2"}}
	for _, source := range []string{
		strings.Replace(historicalOutcome, "accepted_by: chen", "accepted_by: none", 1),
		strings.Replace(historicalOutcome, "A2: confirmed", "A2: pending", 1),
		strings.Replace(historicalOutcome, "A2: confirmed", "A1: confirmed", 1),
		strings.Replace(historicalOutcome, "  - Evidence: The acceptor confirmed the existing record.", "", 1),
		strings.Replace(historicalOutcome, "story: story-01-base", "story: story-99-other", 1),
	} {
		var report diagnostic.Report
		validateHistoricalRegressionOutcome("outcome.md", source, story, &report)
		if !report.HasErrors() {
			t.Fatalf("accepted invalid source: %s", source)
		}
	}
}

func TestHistoricalRegressionOutcomeDoesNotDowngradePartiallyModernRecords(t *testing.T) {
	var report diagnostic.Report
	source := strings.Replace(historicalOutcome, "story: story-01-base", "story: story-01-base\nspec_rev: rev-1", 1)
	if validateHistoricalRegressionOutcome("outcome.md", source, Story{}, &report) {
		t.Fatal("partially modern record downgraded")
	}
}

func TestCurrentOutcomeStillRequiresStructuredVerification(t *testing.T) {
	var report diagnostic.Report
	validateOutcomeSource("outcome.md", historicalOutcome, Story{ID: "story-01-base", AcceptancePolicy: AcceptanceHuman, AcceptanceIDs: []string{"A1", "A2"}}, &report)
	assertIssueContains(t, report.Errors(), "frontmatter missing required key: story_digest")
}

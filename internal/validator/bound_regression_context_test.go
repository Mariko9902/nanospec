package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nanospec/internal/diagnostic"
)

func TestBoundExternalRegressionKeepsFrozenRecordAndReportsEvidenceGaps(t *testing.T) {
	current, reference, source := writeBoundRegressionFixture(t, false)
	var report diagnostic.Report
	ValidateStoryDirectory(current, fixtureCycleID, &report)
	if report.HasErrors() || len(report.Warnings()) != 1 {
		t.Fatalf("issues = %#v", report.Issues())
	}
	assertIssueContains(t, report.Warnings(), "Evidence Artifacts and Evidence Context absent for A1, A2")
	assertIssueContains(t, report.Warnings(), "missing evidence is not verified or inherited")
	if actual := readTestFile(t, filepath.Join(reference, "acceptance", "outcome.md")); actual != source {
		t.Fatal("frozen Outcome changed")
	}
}

func TestBoundRegressionRejectsOtherMissingOrInvalidAcceptanceFacts(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"wrong digest":        func(s string) string { return strings.Replace(s, "story_digest: sha256:", "story_digest: sha256:0", 1) },
		"missing digest":      func(s string) string { return removeOutcomeLines(s, "story_digest:") },
		"missing revision":    func(s string) string { return removeOutcomeLines(s, "spec_rev:") },
		"missing implementer": func(s string) string { return removeOutcomeLines(s, "implemented_by:") },
		"missing verifier":    func(s string) string { return removeOutcomeLines(s, "verified_by:") },
		"missing acceptor":    func(s string) string { return removeOutcomeLines(s, "accepted_by:") },
		"missing timestamp":   func(s string) string { return removeOutcomeLines(s, "accepted_at:") },
		"missing metrics":     func(s string) string { return removeOutcomeLines(s, "rework_count:") },
		"incomplete coverage": func(s string) string {
			return strings.Replace(s, "verified_acceptance: [A1, A2]", "verified_acceptance: [A1]", 1)
		},
		"not confirmed":        func(s string) string { return strings.Replace(s, "A2: confirmed", "A2: pending", 1) },
		"duplicate result":     func(s string) string { return strings.Replace(s, "A2: confirmed", "A1: confirmed", 1) },
		"missing observation":  func(s string) string { return removeOutcomeLines(s, "  - Observed:") },
		"missing command":      func(s string) string { return removeOutcomeLines(s, "  - Evidence Command:") },
		"missing result":       func(s string) string { return removeOutcomeLines(s, "  - Evidence Result:") },
		"partial artifacts":    func(s string) string { return s + "  - Evidence Artifacts: evidence/result.txt\n" },
		"partial context":      func(s string) string { return s + "  - Evidence Context: rev-1, development\n" },
		"empty artifact field": func(s string) string { return s + "  - Evidence Artifacts:\n" },
	} {
		t.Run(name, func(t *testing.T) {
			current, reference, source := writeBoundRegressionFixture(t, false)
			writeFixtureFile(t, filepath.Join(reference, "acceptance", "outcome.md"), mutate(source))
			var report diagnostic.Report
			ValidateStoryDirectory(current, fixtureCycleID, &report)
			if !report.HasErrors() || len(report.Warnings()) != 0 {
				t.Fatalf("issues = %#v", report.Issues())
			}
		})
	}
}

func TestBoundRegressionRejectsChangedStoryAndNonHumanAcceptance(t *testing.T) {
	for _, changedStory := range []bool{true, false} {
		current, reference, source := writeBoundRegressionFixture(t, false)
		if changedStory {
			file := filepath.Join(reference, "story.md")
			writeFixtureFile(t, file, readTestFile(t, file)+"\nContent changed.\n")
		} else {
			writeFixtureFile(t, filepath.Join(reference, "acceptance", "outcome.md"), strings.Replace(source, "acceptance_method: human", "acceptance_method: verified", 1))
		}
		var report diagnostic.Report
		ValidateStoryDirectory(current, fixtureCycleID, &report)
		if !report.HasErrors() || len(report.Warnings()) != 0 {
			t.Fatalf("issues = %#v", report.Issues())
		}
	}
}

func TestBoundRecordStillFailsCurrentAndSameCycleOutcomeValidation(t *testing.T) {
	current, reference, source := writeBoundRegressionFixture(t, true)
	for name, directory := range map[string]string{"same cycle regression": current, "current accepted outcome": reference} {
		t.Run(name, func(t *testing.T) {
			var report diagnostic.Report
			ValidateStoryDirectory(directory, fixtureCycleID, &report)
			assertIssue(t, report.Errors(), "Acceptance Results A1 missing Evidence Artifacts")
			assertIssue(t, report.Errors(), "Acceptance Results A2 missing Evidence Context")
			if len(report.Warnings()) != 0 {
				t.Fatalf("unexpected warnings: %#v", report.Warnings())
			}
		})
	}
	if actual := readTestFile(t, filepath.Join(reference, "acceptance", "outcome.md")); actual != source {
		t.Fatal("current Outcome changed")
	}
}

func TestPreparedBoundRegressionRetainsOriginalAndEvidenceCaveat(t *testing.T) {
	current, reference, source := writeBoundRegressionFixture(t, false)
	prepared, result := BuildExecutionPackage(filepath.Join(current, "story.md"), "# Execution Rules\n\nUse current facts.")
	if result.InputError != "" || len(result.Errors) != 0 || len(result.Warnings) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(prepared.ContextMarkdown, "do not transfer acceptance conclusions or supply missing evidence") || !strings.Contains(prepared.ContextMarkdown, source) {
		t.Fatal("execution context lost frozen source or evidence caveat")
	}
	if !strings.Contains(prepared.ContextMarkdown, "> cycle-00-old/story-02-base: frozen external human acceptance") ||
		!strings.Contains(prepared.ContextMarkdown, "Evidence Artifacts and Evidence Context absent for A1, A2") {
		t.Fatal("execution context lost reference-specific evidence gaps")
	}
	if readTestFile(t, filepath.Join(reference, "acceptance", "outcome.md")) != source {
		t.Fatal("prepare changed original Outcome")
	}
}

func TestCompleteRegressionDoesNotReportMissingEvidence(t *testing.T) {
	current, reference, _ := writeBoundRegressionFixture(t, false)
	source := structuredOutcome(t, reference, "story-02-base", "human", "self")
	writeFixtureFile(t, filepath.Join(reference, "acceptance", "outcome.md"), source)
	prepared, result := BuildExecutionPackage(filepath.Join(current, "story.md"), "# Execution Rules\n\nUse current facts.")
	if result.InputError != "" || len(result.Errors) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if strings.Contains(prepared.ContextMarkdown, "Evidence Artifacts and Evidence Context absent") {
		t.Fatal("complete reference was reported as missing evidence")
	}
}

func writeBoundRegressionFixture(t *testing.T, sameCycle bool) (string, string, string) {
	t.Helper()
	cycle := writeCycleFixture(t, map[string][]string{"story-01-fix": {}})
	current := filepath.Join(cycle, "story-01-fix")
	referenceCycle := "cycle-00-old"
	if sameCycle {
		referenceCycle = fixtureCycleID
	}
	reference := filepath.Join(filepath.Dir(cycle), referenceCycle, "story-02-base")
	if err := os.MkdirAll(reference, 0o755); err != nil {
		t.Fatal(err)
	}
	story := strings.Replace(basicStory("story-02-base", "", "none"), "cycle: "+fixtureCycleID, "cycle: "+referenceCycle, 1)
	writeFixtureFile(t, filepath.Join(reference, "story.md"), story)
	source := removeOutcomeLines(structuredOutcome(t, reference, "story-02-base", "human", "self"), "  - Evidence Artifacts:", "  - Evidence Context:")
	writeFixtureFile(t, filepath.Join(reference, "acceptance", "outcome.md"), source)
	currentFile := filepath.Join(current, "story.md")
	text := strings.Replace(readTestFile(t, currentFile), "depends_on: []\n", "depends_on: []\nregression_of: ["+referenceCycle+"/story-02-base]\n", 1)
	writeFixtureFile(t, currentFile, text)
	return current, reference, source
}

func removeOutcomeLines(source string, prefixes ...string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		remove := false
		for _, prefix := range prefixes {
			remove = remove || strings.HasPrefix(line, prefix)
		}
		if !remove {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

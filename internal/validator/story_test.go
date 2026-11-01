package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nanospec/internal/diagnostic"
)

const fixtureCycleID = "cycle-01-sample"

func TestValidateStoryDirectoryAcceptsCompleteStory(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	var report diagnostic.Report

	story := ValidateStoryDirectory(directory, fixtureCycleID, &report)

	if report.HasErrors() {
		t.Fatalf("errors = %#v", report.Errors())
	}
	if len(report.Warnings()) != 0 {
		t.Fatalf("warnings = %#v", report.Warnings())
	}
	if story.ID != "story-01-order" || story.Title != "Save an order" || story.Order != "01" {
		t.Fatalf("story = %#v", story)
	}
	if len(story.AcceptanceIDs) != 2 || story.AcceptanceIDs[1] != "A2" {
		t.Fatalf("acceptance IDs = %#v", story.AcceptanceIDs)
	}
}

func TestValidateStoryDirectoryChecksAcceptanceFlowCoverage(t *testing.T) {
	document := strings.Replace(
		basicStory("story-01-order", "", "none"),
		"A[\"Save order A1\"] --> B[\"Check result A2\"]",
		"A[\"Save order A1\"] --> B[\"Check result\"]",
		1,
	)
	directory := writeStoryFixture(t, "story-01-order", document)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Acceptance Flow does not reference Acceptance ID: A2")
}

func TestValidateStoryDirectoryRejectsUnknownFlowAcceptanceID(t *testing.T) {
	document := strings.Replace(
		basicStory("story-01-order", "", "none"),
		"A[\"Save order A1\"] --> B[\"Check result A2\"]",
		"A[\"Save order A1\"] --> B[\"Check results A2 and A99\"]",
		1,
	)
	directory := writeStoryFixture(t, "story-01-order", document)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Acceptance Flow references unknown Acceptance ID: A99")
}

func TestValidateStoryDirectoryRejectsNonCanonicalAcceptanceID(t *testing.T) {
	document := strings.ReplaceAll(basicStory("story-01-order", "", "none"), "A1", "QA1")
	directory := writeStoryFixture(t, "story-01-order", document)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "invalid Acceptance ID: QA1; expected A1, A2, ...")
}

func TestValidateStoryDirectoryChecksExecutableCheckReferences(t *testing.T) {
	document := strings.Replace(
		basicStory("story-01-order", "", "none"),
		"- [ ] A1: A successful save shows the persisted result.",
		"- [ ] A1: A successful save shows the persisted result.\n  - Checks: [check-save]",
		1,
	)
	document += "\n### Executable Checks\n\n" +
		"- Check: check-save\n" +
		"  - Run: `go test ./...`\n" +
		"  - Environment: Local database\n" +
		"  - Expected: Save and failure cases pass\n"
	directory := writeStoryFixture(t, "story-01-order", document)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	if report.HasErrors() {
		t.Fatalf("errors = %#v", report.Errors())
	}
}

func TestValidateStoryDirectoryRejectsIncompleteExecutableCheck(t *testing.T) {
	document := strings.Replace(
		basicStory("story-01-order", "", "none"),
		"- [ ] A1: A successful save shows the persisted result.",
		"- [ ] A1: A successful save shows the persisted result.\n  - Checks: [check-missing, check-incomplete]",
		1,
	)
	document += "\n### Executable Checks\n\n" +
		"- Check: check-incomplete\n" +
		"  - Run: `go test ./...`\n" +
		"  - Expected: Save case passes\n"
	directory := writeStoryFixture(t, "story-01-order", document)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Acceptance references missing executable check: check-missing")
	assertIssue(t, report.Errors(), "Executable Check check-incomplete missing Environment")
}

func TestValidateStoryDirectoryAcceptsRegressionReferenceToFrozenOutcome(t *testing.T) {
	root := t.TempDir()
	referenceDirectory := filepath.Join(root, "cycle-00-old", "story-01-base")
	currentDirectory := filepath.Join(root, fixtureCycleID, "story-02-order")
	if err := os.MkdirAll(referenceDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(currentDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	referenceStory := strings.Replace(
		basicStory("story-01-base", "", "none"),
		"cycle: "+fixtureCycleID,
		"cycle: cycle-00-old",
		1,
	)
	writeFixtureFile(t, filepath.Join(referenceDirectory, "story.md"), referenceStory)
	writeFixtureFile(t, filepath.Join(referenceDirectory, "acceptance", "outcome.md"), structuredOutcome(t, referenceDirectory, "story-01-base", "human", "self"))
	document := strings.Replace(
		basicStory("story-02-order", "", "none"),
		"depends_on: []\n",
		"depends_on: []\nregression_of: [cycle-00-old/story-01-base]\n",
		1,
	)
	writeFixtureFile(t, filepath.Join(currentDirectory, "story.md"), document)
	var report diagnostic.Report

	story := ValidateStoryDirectory(currentDirectory, fixtureCycleID, &report)

	if report.HasErrors() {
		t.Fatalf("errors = %#v", report.Errors())
	}
	if len(story.RegressionOf) != 1 || story.RegressionOf[0] != "cycle-00-old/story-01-base" {
		t.Fatalf("regression references = %#v", story.RegressionOf)
	}
}

func TestValidateStoryDirectoryRejectsMissingRegressionOutcome(t *testing.T) {
	document := strings.Replace(
		basicStory("story-01-order", "", "none"),
		"depends_on: []\n",
		"depends_on: []\nregression_of: [cycle-00-old/story-01-base]\n",
		1,
	)
	directory := writeStoryFixture(t, "story-01-order", document)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "regression_of must reference a Story with outcome.md: cycle-00-old/story-01-base")
}

func TestValidateStoryDirectoryChecksBriefAndVerifiedOutcome(t *testing.T) {
	metadata := "acceptance_policy: verified\n" +
		"acceptance_authority: Project decision dated 2026-09-10\n" +
		"acceptance_scope: All A1-A2 conditions in this Story\n" +
		"acceptance_review: A separate reviewer checks every Acceptance item and its evidence coverage\n"
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", metadata, "implementation-brief.md"))
	writeFixtureFile(t, filepath.Join(directory, "implementation-brief.md"),
		"# Implementation Brief\n\n## Repository Findings\n\n- `src/order.go`: The save entry returns the persisted result.\n")
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), structuredOutcome(t, directory, "story-01-order", "verified", "independent"))
	var report diagnostic.Report

	story := ValidateStoryDirectory(directory, fixtureCycleID, &report)

	if report.HasErrors() {
		t.Fatalf("errors = %#v", report.Errors())
	}
	if story.AcceptancePolicy != AcceptanceVerified || !story.HasOutcome {
		t.Fatalf("story = %#v", story)
	}
}

func TestValidateStoryDirectoryAllowsHumanAcceptanceWithVerifiedPolicy(t *testing.T) {
	metadata := "acceptance_policy: verified\n" +
		"acceptance_authority: project rule\n" +
		"acceptance_scope: A1-A2\n" +
		"acceptance_review: independent reviewer\n"
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", metadata, "none"))
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), structuredOutcome(t, directory, "story-01-order", "human", "self"))
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	if report.HasErrors() {
		t.Fatalf("errors = %#v", report.Errors())
	}
	if len(report.Warnings()) != 0 {
		t.Fatalf("warnings = %#v", report.Warnings())
	}
}

func TestValidateStoryDirectoryRejectsMismatchedOutcomeBinding(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	storySource, err := os.ReadFile(filepath.Join(directory, "story.md"))
	if err != nil {
		t.Fatal(err)
	}
	outcome := structuredOutcome(t, directory, "story-01-order", "human", "self")
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), outcome)
	writeFixtureFile(t, filepath.Join(directory, "story.md"), string(storySource)+"\nchanged\n")
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "outcome.story_digest does not match story.md")
}

func TestValidateStoryDirectoryRejectsSelfVerifiedAcceptance(t *testing.T) {
	metadata := "acceptance_policy: verified\n" +
		"acceptance_authority: project rule\n" +
		"acceptance_scope: A1-A2\n" +
		"acceptance_review: independent reviewer\n"
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", metadata, "none"))
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), structuredOutcome(t, directory, "story-01-order", "verified", "self"))
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "verified acceptance requires independent verification")
}

func TestValidateStoryDirectoryAcceptsStructuredHumanOutcome(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), structuredOutcome(t, directory, "story-01-order", "human", "self"))
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	if report.HasErrors() {
		t.Fatalf("errors = %#v", report.Errors())
	}
}

func TestValidateStoryDirectoryRejectsIncompleteStructuredVerification(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	outcome := strings.Replace(structuredOutcome(t, directory, "story-01-order", "human", "independent"), "verified_acceptance: [A1, A2]", "verified_acceptance: [A1]", 1)
	outcome = strings.Replace(outcome, "verified_by: reviewer", "verified_by: implementer", 1)
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), outcome)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "independent verification requires a verifier other than the implementer")
	assertIssue(t, report.Errors(), "verified_acceptance missing Acceptance ID: A2")
}

func TestValidateStoryDirectoryRejectsInconsistentOutcomeMetrics(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	outcome := strings.Replace(structuredOutcome(t, directory, "story-01-order", "human", "self"), "rework_count: 0", "rework_count: 1", 1)
	outcome = strings.Replace(outcome, "spec_change_count: 0", "spec_change_count: many", 1)
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), outcome)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "first_pass must be true exactly when rework_count is 0")
	assertIssue(t, report.Errors(), "spec_change_count must be a non-negative integer")
}

func TestValidateStoryDirectoryRejectsOutcomeWithoutMetrics(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	outcome := structuredOutcome(t, directory, "story-01-order", "human", "self")
	for _, line := range []string{"first_pass: true\n", "rework_count: 0\n", "spec_change_count: 0\n"} {
		outcome = strings.Replace(outcome, line, "", 1)
	}
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), outcome)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "frontmatter missing required key: first_pass")
	assertIssue(t, report.Errors(), "frontmatter missing required key: rework_count")
	assertIssue(t, report.Errors(), "frontmatter missing required key: spec_change_count")
}

func TestValidateStoryDirectoryRequiresEvidenceContext(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	outcome := structuredOutcome(t, directory, "story-01-order", "human", "self")
	outcome = strings.ReplaceAll(outcome, "  - Evidence Context: rev-1, test build, local, 2026-09-10T09:30:00+08:00\n", "")
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), outcome)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Acceptance Results A1 missing Evidence Context")
	assertIssue(t, report.Errors(), "Acceptance Results A2 missing Evidence Context")
}

func TestValidateStoryDirectoryRejectsIncompleteOutcome(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", "", "none"))
	outcome := strings.Replace(structuredOutcome(t, directory, "story-01-order", "human", "self"), "  - Evidence Command: `go test ./...`\n", "", 1)
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), outcome)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Acceptance Results A1 missing Evidence Command")
}

func TestValidateStoryDirectoryRejectsInvalidStructure(t *testing.T) {
	document := strings.Replace(basicStory("story-01-order", "", "none"), "### Description", "## Description", 1)
	document = strings.Replace(document, "- [ ] A2: A failed save leaves the original result unchanged.\n", "- [ ] A1: Duplicate result.\n", 1)
	directory := writeStoryFixture(t, "story-01-order", document)
	writeFixtureFile(t, filepath.Join(directory, "notes.md"), "extra")
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Description must be nested under Definition")
	assertIssue(t, report.Errors(), "Definition missing subsection: Description")
	assertIssue(t, report.Errors(), "duplicate Acceptance ID: A1")
	assertIssue(t, report.Errors(), "unsupported Markdown file in story directory")
}

func TestValidateStoryDirectoryRejectsUnsupportedSchema(t *testing.T) {
	document := strings.Replace(
		basicStory("story-01-order", "unknown_field: value\n", "none"),
		"## Acceptance",
		"## Notes\n\nextra\n\n## Acceptance",
		1,
	)
	directory := writeStoryFixture(t, "story-01-order", document)
	writeFixtureFile(t, filepath.Join(directory, "acceptance", "outcome.md"), strings.Replace(
		structuredOutcome(t, directory, "story-01-order", "human", "self"),
		"## Acceptance Results",
		"## Notes\n\nextra\n\n## Acceptance Results",
		1,
	))
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Story frontmatter has unsupported key: unknown_field")
	assertIssue(t, report.Errors(), "Story has unsupported section: Notes")
	assertIssue(t, report.Errors(), "Outcome has unsupported section: Notes")
}

func TestValidateStoryDirectoryRejectsInvalidOrientation(t *testing.T) {
	document := strings.Replace(
		basicStory("story-01-order", "", "none"),
		"- Title: Save an order\n- Scenario: 『Clerk』 confirms the 『order』 and runs 「Save」; the 『system』 shows the result and preserves the 『original content』 if saving fails.",
		"- Title: none\n- Current Problem: The save result is unclear.",
		1,
	)
	directory := writeStoryFixture(t, "story-01-order", document)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Orientation field must contain meaningful content: Title")
	assertIssue(t, report.Errors(), "Orientation has unsupported field: Current Problem")
	assertIssue(t, report.Errors(), "Orientation missing field: Scenario")
}

func TestValidateStoryDirectoryRejectsInvalidAcceptancePolicy(t *testing.T) {
	directory := writeStoryFixture(t, "story-01-order",
		basicStory("story-01-order", "acceptance_policy: [verified]\n", "none"))
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "acceptance_policy must be human or verified")
}

func TestValidateStoryDirectoryRejectsIncompleteVerifiedPolicy(t *testing.T) {
	metadata := "acceptance_policy: verified\n" +
		"acceptance_authority: project rule\n"
	directory := writeStoryFixture(t, "story-01-order", basicStory("story-01-order", metadata, "none"))
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "verified policy requires acceptance_scope")
	assertIssue(t, report.Errors(), "verified policy requires acceptance_review")
}

func basicStory(id, extraMetadata, brief string) string {
	fence := strings.Repeat(string(rune(96)), 3)
	return "---\n" +
		"id: " + id + "\n" +
		"cycle: " + fixtureCycleID + "\n" +
		"depends_on: []\n" +
		"implementation_brief: " + brief + "\n" +
		extraMetadata +
		"---\n\n" +
		"## Orientation\n\n" +
		"- Title: Save an order\n" +
		"- Scenario: 『Clerk』 confirms the 『order』 and runs 「Save」; the 『system』 shows the result and preserves the 『original content』 if saving fails.\n\n" +
		"## Definition\n\n" +
		"### Description\n\nSave an order from the existing entry. Show the result on success and retain the original data on failure.\n\n" +
		"### Business Logic\n\nShow success only after the order has been persisted.\n\n" +
		"## Acceptance\n\n" +
		"### Acceptance Flow\n\n" +
		fence + "mermaid\n" +
		"flowchart TD\n" +
		"  A[\"Save order A1\"] --> B[\"Check result A2\"]\n" +
		fence + "\n\n" +
		"- [ ] A1: A successful save shows the persisted result.\n" +
		"- [ ] A2: A failed save leaves the original result unchanged.\n"
}

func structuredOutcome(t *testing.T, directory, id, method, mode string) string {
	t.Helper()
	storySource, err := os.ReadFile(filepath.Join(directory, "story.md"))
	if err != nil {
		t.Fatal(err)
	}
	verifiedBy := "implementer"
	if mode == "independent" {
		verifiedBy = "reviewer"
	}
	return "---\n" +
		"story: " + id + "\n" +
		"spec_rev: rev-1\n" +
		"story_digest: " + storyContentDigest(storySource) + "\n" +
		"implemented_by: implementer\n" +
		"verified_by: " + verifiedBy + "\n" +
		"verification_mode: " + mode + "\n" +
		"verified_at: 2026-09-10T09:30:00+08:00\n" +
		"verified_acceptance: [A1, A2]\n" +
		"first_pass: true\n" +
		"rework_count: 0\n" +
		"spec_change_count: 0\n" +
		"accepted_by: product\n" +
		"accepted_at: 2026-09-10T10:00:00+08:00\n" +
		"acceptance_method: " + method + "\n" +
		"---\n\n" +
		"## Acceptance Results\n\n" +
		"- A1: confirmed\n" +
		"  - Observed: The persisted result is shown after a successful save\n" +
		"  - Evidence Command: `go test ./...`\n" +
		"  - Evidence Result: Save case passes\n" +
		"  - Evidence Artifacts: artifacts/save.txt\n" +
		"  - Evidence Context: rev-1, test build, local, 2026-09-10T09:30:00+08:00\n" +
		"- A2: confirmed\n" +
		"  - Observed: A failed save leaves the original result unchanged\n" +
		"  - Evidence Command: `go test ./...`\n" +
		"  - Evidence Result: Failure case passes\n" +
		"  - Evidence Artifacts: artifacts/failure.txt\n" +
		"  - Evidence Context: rev-1, test build, local, 2026-09-10T09:30:00+08:00\n"
}

func writeStoryFixture(t *testing.T, id, document string) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), id)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(directory, "story.md"), document)
	return directory
}

func writeFixtureFile(t *testing.T, file, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertIssue(t *testing.T, issues []diagnostic.Issue, message string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Message == message {
			return
		}
	}
	t.Fatalf("issue %q not found in %#v", message, issues)
}

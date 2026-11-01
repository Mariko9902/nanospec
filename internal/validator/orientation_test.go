package validator

import (
	"strings"
	"testing"

	"nanospec/internal/diagnostic"
)

func TestValidateStoryDirectoryRejectsRemovedOrientationField(t *testing.T) {
	document := strings.Replace(
		basicStory("story-01-order", "", "none"),
		"- Scenario:",
		"- Location: Orders -> Confirm order -> Save\n- Scenario:",
		1,
	)
	directory := writeStoryFixture(t, "story-01-order", document)
	var report diagnostic.Report

	ValidateStoryDirectory(directory, fixtureCycleID, &report)

	assertIssue(t, report.Errors(), "Orientation has unsupported field: Location")
	if len(report.Errors()) != 1 {
		t.Fatalf("errors = %#v", report.Errors())
	}
}

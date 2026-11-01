package diagnostic

import "testing"

func TestReportSeparatesSeverityAndReturnsCopies(t *testing.T) {
	var first Report
	var second Report
	first.AddWarning("cycle.md", "check this")
	first.AddError("story.md", "invalid value")
	second.AddError("other.md", "separate run")

	errors := first.Errors()
	warnings := first.Warnings()
	issues := first.Issues()
	issues[0].Message = "changed"

	if len(errors) != 1 || errors[0].File != "story.md" {
		t.Fatalf("Errors() = %#v", errors)
	}
	if len(warnings) != 1 || warnings[0].File != "cycle.md" {
		t.Fatalf("Warnings() = %#v", warnings)
	}
	if first.Issues()[0].Message != "check this" {
		t.Fatal("Issues() exposed mutable report storage")
	}
	if len(second.Issues()) != 1 || second.Issues()[0].File != "other.md" {
		t.Fatalf("second report = %#v", second.Issues())
	}
	if !first.HasErrors() {
		t.Fatal("HasErrors() = false, want true")
	}
}

func TestEmptyReportHasNoErrors(t *testing.T) {
	var report Report
	if report.HasErrors() {
		t.Fatal("zero Report has errors")
	}
	if len(report.Issues()) != 0 {
		t.Fatalf("zero Report issues = %#v", report.Issues())
	}
}

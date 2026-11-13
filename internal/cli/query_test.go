package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nanospec/internal/validator"
)

func TestRunReadOnlyQueries(t *testing.T) {
	cycleDirectory := writeQueryFixture(t)
	executionRules := "# Execution Rules\n\nUse current facts.\n"
	sync := validator.ValidateSpec(cycleDirectory, true)
	if len(sync.Errors) != 0 {
		t.Fatalf("sync errors = %#v", sync.Errors)
	}
	if err := validator.SaveViewUpdates(sync.Updates); err != nil {
		t.Fatal(err)
	}
	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	beforeCycle := readQueryFile(t, cycleFile)

	checks := []struct {
		args     []string
		contains string
	}{
		{[]string{"show", "overview", cycleDirectory}, "## [pending] Look up the base result (story-01-base)"},
		{[]string{"show", "graph", cycleDirectory}, "## Dependency Graph"},
		{[]string{"ready", cycleDirectory}, "story-01-base | acceptance=pending"},
		{[]string{"prepare", filepath.Join(cycleDirectory, "story-01-base")}, "# NanoSpec Execution Preparation"},
	}
	for _, check := range checks {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := RunWithExecutionRules(check.args, &stdout, &stderr, executionRules); code != 0 {
			t.Fatalf("args = %#v, code = %d, stderr = %q", check.args, code, stderr.String())
		}
		if !strings.Contains(stdout.String(), check.contains) {
			t.Fatalf("args = %#v, stdout = %q", check.args, stdout.String())
		}
	}
	if readQueryFile(t, cycleFile) != beforeCycle {
		t.Fatal("read-only query changed files")
	}
}

func TestRunQueryHelpAndInvalidArguments(t *testing.T) {
	for _, command := range []string{"show", "ready", "prepare"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Run([]string{command, "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Usage: nanospec "+command) {
			t.Fatalf("command = %s, code = %d, stdout = %q", command, code, stdout.String())
		}
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"show", "unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("code = %d", code)
	}
}

func TestRunPrepareRequiresExecutionRules(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"prepare", "story-01-unused"}, &stdout, &stderr)

	if code != 2 || !strings.Contains(stderr.String(), "execution rules are unavailable") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunPrepareJSONUsesStableEnvelope(t *testing.T) {
	cycleDirectory := writeQueryFixture(t)
	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	cycle := strings.Replace(readQueryFile(t, cycleFile), "window:", "spec_language: ja-JP\nwindow:", 1)
	writeQueryFile(t, cycleFile, cycle)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithExecutionRules([]string{"prepare", "--json", filepath.Join(cycleDirectory, "story-01-base")}, &stdout, &stderr, "# Rules\n\nUse facts.")

	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	for _, fragment := range []string{`"schema": "nanospec.prepare.v1"`, `"selection": "explicit"`, `"story": "story-01-base"`, `"spec_language": "ja-JP"`, `"gate": "ready_for_confirmation"`} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout missing %q: %s", fragment, stdout.String())
		}
	}
}

func TestRunPrepareRejectsDuplicateSelection(t *testing.T) {
	cycleDirectory := writeQueryFixture(t)
	target := filepath.Join(cycleDirectory, "story-01-base")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithExecutionRules([]string{"prepare", target, target}, &stdout, &stderr, "# Rules\n\nUse facts.")

	if code != 2 || !strings.Contains(stderr.String(), "duplicate prepared Story") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func writeQueryFixture(t *testing.T) string {
	t.Helper()
	cycleDirectory := filepath.Join(t.TempDir(), "cycle-01-query")
	storyDirectory := filepath.Join(cycleDirectory, "story-01-base")
	if err := os.MkdirAll(storyDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeQueryFile(t, filepath.Join(storyDirectory, "story.md"), `---
id: story-01-base
cycle: cycle-01-query
depends_on: []
implementation_brief: none
---

## Orientation

- Title: Look up the base result
- Scenario: 『Staff』 opens the 『query center』 and runs 「Look up the base result」; the 『system』 shows the 『base result』 for the next step.

## Definition

### Description

The base result is available.

## Acceptance

- [ ] A1: The base result can be observed.

### Acceptance Flow

`+"```"+`mermaid
flowchart TD
  Open["Open result"] --> Check["Confirm A1"]
`+"```"+`
`)
	writeQueryFile(t, filepath.Join(cycleDirectory, "cycle.md"), `---
id: cycle-01-query
window: [current]
spec_rev: rev-1
status: active
last_updated_at: 2026-09-10T10:00:00+08:00
---

## Planning Context

- Current delivery.

## Dependency Graph

- none

## Risks

- none

## Current Situation

- none

## Active Lanes

- none

## Acceptance Status

- Story: story-01-base
  - Acceptance: pending: not observed

## Open Issues

- none

## Practice Feedback

- none
`)
	return cycleDirectory
}

func writeQueryFile(t *testing.T, file, content string) {
	t.Helper()
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readQueryFile(t *testing.T, file string) string {
	t.Helper()
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

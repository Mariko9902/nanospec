package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCycleSnapshotBuildsViewsWithoutWriting(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base":  {},
		"story-02-order": {"story-01-base"},
	})
	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	before := readTestFile(t, cycleFile)

	snapshot, result := LoadCycleSnapshot(cycleDirectory)

	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	view, err := snapshot.View("graph")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(view, "S01 --> S02") {
		t.Fatalf("view = %q", view)
	}
	overview, err := snapshot.View("overview")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"# Cycle Overview", "## [pending] Save an order (story-01-base)", "## [pending] Save an order (story-02-order)", "- Scenario: 『Clerk』 confirms the 『order』 and runs 「Save」"} {
		if !strings.Contains(overview, fragment) {
			t.Fatalf("overview missing %q:\n%s", fragment, overview)
		}
	}
	if strings.Contains(overview, "- Title:") {
		t.Fatalf("overview repeats Title field:\n%s", overview)
	}
	if strings.Contains(overview, "Location") {
		t.Fatalf("overview contains removed field:\n%s", overview)
	}
	if after := readTestFile(t, cycleFile); after != before {
		t.Fatal("LoadCycleSnapshot changed cycle.md")
	}
}

func TestReadyCandidatesUseStoryOrderAndDependencyStatus(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base":  {},
		"story-02-order": {"story-01-base"},
		"story-03-help":  {},
	})
	stateFile := filepath.Join(cycleDirectory, "cycle.md")
	state := readTestFile(t, stateFile)
	state = strings.Replace(state, "Acceptance: pending: not observed", "Acceptance: ready: observed", 1)
	writeFixtureFile(t, stateFile, state)

	snapshot, result := LoadCycleSnapshot(cycleDirectory)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	candidates := snapshot.ReadyCandidates()
	if len(candidates) != 2 {
		t.Fatalf("candidates = %#v", candidates)
	}
	if candidates[0].StoryID != "story-02-order" || candidates[1].StoryID != "story-03-help" {
		t.Fatalf("candidate order = %#v", candidates)
	}
	if len(candidates[0].Dependencies) != 1 || candidates[0].Dependencies[0].Acceptance != "ready" {
		t.Fatalf("dependent candidate = %#v", candidates[1])
	}
}

func TestReadyCandidatesExcludeAssignedStory(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	stateFile := filepath.Join(cycleDirectory, "cycle.md")
	state := readTestFile(t, stateFile)
	state = strings.Replace(state, "## Active Lanes\n\n- none", "## Active Lanes\n\n- Lane: lane-01\n  - Story: story-01-base\n  - Owner: executor-a\n  - Status: active\n  - Next: continue\n  - Implementation Complete: false", 1)
	writeFixtureFile(t, stateFile, state)

	snapshot, result := LoadCycleSnapshot(cycleDirectory)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if candidates := snapshot.ReadyCandidates(); len(candidates) != 0 {
		t.Fatalf("candidates = %#v", candidates)
	}
}

func TestBuildExecutionPackageIncludesStableInputs(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base":  {},
		"story-02-order": {"story-01-base"},
	})
	storyFile := filepath.Join(cycleDirectory, "story-02-order", "story.md")

	executionRules := "# Execution Rules\n\nUse the current specification and record observed facts."
	first, result := BuildExecutionPackage(storyFile, executionRules)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	second, repeated := BuildExecutionPackage(storyFile, executionRules)
	if repeated.InputError != "" || len(repeated.Errors) != 0 || first.ContextMarkdown != second.ContextMarkdown {
		t.Fatal("context output is not stable")
	}
	if first.CycleID != fixtureCycleID || first.StoryID != "story-02-order" || first.SpecRevision != "rev-1" || first.Acceptance != "pending" {
		t.Fatalf("package = %#v", first)
	}
	if len(first.Dependencies) != 1 || first.Dependencies[0].StoryID != "story-01-base" {
		t.Fatalf("dependencies = %#v", first.Dependencies)
	}
	for _, fragment := range []string{"# NanoSpec Execution Context", "Spec Revision: rev-1", "story-01-base: pending: not observed", "## NanoSpec Execution Rules", executionRules, "## Story Specification", "id: story-02-order"} {
		if !strings.Contains(first.ContextMarkdown, fragment) {
			t.Fatalf("context missing %q:\n%s", fragment, first.ContextMarkdown)
		}
	}
}

func TestBuildExecutionPackageIncludesFrozenRegressionSources(t *testing.T) {
	currentDirectory := writeCycleFixture(t, map[string][]string{"story-01-fix": {}})
	root := filepath.Dir(currentDirectory)
	referenceDirectory := filepath.Join(root, "cycle-00-old", "story-01-base")
	if err := os.MkdirAll(referenceDirectory, 0o755); err != nil {
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
	storyFile := filepath.Join(currentDirectory, "story-01-fix", "story.md")
	storyText := readTestFile(t, storyFile)
	storyText = strings.Replace(storyText, "depends_on: []\n", "depends_on: []\nregression_of: [cycle-00-old/story-01-base]\n", 1)
	writeFixtureFile(t, storyFile, storyText)

	prepared, result := BuildExecutionPackage(storyFile, "# Execution Rules\n\nUse current facts.")

	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	for _, fragment := range []string{
		"## Regression Reference cycle-00-old/story-01-base: Accepted Story",
		"id: story-01-base",
		"## Regression Reference cycle-00-old/story-01-base: Outcome",
		"story: story-01-base",
	} {
		if !strings.Contains(prepared.ContextMarkdown, fragment) {
			t.Fatalf("context missing %q:\n%s", fragment, prepared.ContextMarkdown)
		}
	}
}

func TestResolvePreparationGatesUsesExplicitSelection(t *testing.T) {
	packages := []ExecutionPackage{
		{CycleID: "cycle-01-sample", StoryID: "story-01-base", Acceptance: "pending"},
		{CycleID: "cycle-01-sample", StoryID: "story-02-selected", Acceptance: "pending", Dependencies: []DependencyStatus{{StoryID: "story-01-base", Acceptance: "pending"}}},
		{CycleID: "cycle-01-sample", StoryID: "story-03-blocked", Acceptance: "pending", Dependencies: []DependencyStatus{{StoryID: "story-09-missing", Acceptance: "pending"}}},
		{CycleID: "cycle-01-sample", StoryID: "story-04-review", Acceptance: "ready"},
		{CycleID: "cycle-01-sample", StoryID: "story-05-closed", Acceptance: "accepted"},
		{CycleID: "cycle-01-sample", StoryID: "story-06-question", Acceptance: "pending", OpenQuestions: true},
	}

	for index := range packages {
		packages[index].Status = StoryStatus(packages[index].Acceptance)
		for j := range packages[index].Dependencies {
			packages[index].Dependencies[j].Status = StoryStatus(packages[index].Dependencies[j].Acceptance)
		}
	}
	ResolvePreparationGates(packages)

	want := []string{"ready_for_confirmation", "waiting_for_selected_dependency", "blocked_by_dependency", "awaiting_acceptance", "accepted", "blocked_by_open_questions"}
	for index, expected := range want {
		if packages[index].Gate != expected {
			t.Fatalf("package %d gate = %q, want %q", index, packages[index].Gate, expected)
		}
	}
}

func TestQueryRejectsWrongTargetKind(t *testing.T) {
	file := filepath.Join(t.TempDir(), "other.md")
	if err := os.WriteFile(file, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, result := LoadCycleSnapshot(file); !strings.Contains(result.InputError, "one Cycle directory") {
		t.Fatalf("cycle result = %#v", result)
	}
	if _, result := BuildExecutionPackage(file, "rules"); !strings.Contains(result.InputError, "Story directory or story.md") {
		t.Fatalf("story result = %#v", result)
	}
}

func TestBuildExecutionPackageRequiresExecutionRules(t *testing.T) {
	if _, result := BuildExecutionPackage("unused", ""); result.InputError != "NanoSpec execution rules are empty" {
		t.Fatalf("result = %#v", result)
	}
}

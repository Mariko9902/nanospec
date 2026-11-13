package validator

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildCheckpointUpdateCreatesAndReusesLane(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 12, 30, 0, 0, time.FixedZone("CST", 8*60*60))

	created, result := BuildCheckpointUpdate(target, CheckpointRequest{SpecRevision: "rev-1",
		Owner: "executor-a", Status: "active", Next: "implement A1", Situation: "repository facts confirmed", At: at,
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if created.LaneID != "lane-01" || created.SpecRevision != "rev-1" {
		t.Fatalf("created = %#v", created)
	}
	if err := SaveViewUpdates([]ViewUpdate{created.Update}); err != nil {
		t.Fatal(err)
	}

	blocked, result := BuildCheckpointUpdate(target, CheckpointRequest{SpecRevision: "rev-1",
		Owner: "executor-a", Status: "blocked", Next: "wait for test account", Situation: "implementation is available", Issue: "test account is unavailable", At: at.Add(time.Minute),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if blocked.LaneID != "lane-01" {
		t.Fatalf("blocked = %#v", blocked)
	}
	if err := SaveViewUpdates([]ViewUpdate{blocked.Update}); err != nil {
		t.Fatal(err)
	}

	state := readTestFile(t, filepath.Join(cycleDirectory, "cycle.md"))
	for _, fragment := range []string{
		"- Lane: lane-01",
		"  - Story: story-01-base",
		"  - Status: blocked",
		"  - Spec Changes: 0",
		"- story-01-base: implementation is available",
		"- story-01-base: test account is unavailable",
		"last_updated_at: 2026-09-10T12:31:00+08:00",
	} {
		if !strings.Contains(state, fragment) {
			t.Fatalf("state missing %q:\n%s", fragment, state)
		}
	}
	if validated := ValidateSpec(cycleDirectory, false); len(validated.Errors) != 0 {
		t.Fatalf("errors = %#v", validated.Errors)
	}
}

func TestBuildCheckpointUpdateEnforcesRecordedStartGates(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base": {},
		"story-02-help": {"story-01-base"},
	})
	target := filepath.Join(cycleDirectory, "story-02-help")
	at := time.Date(2026, 9, 10, 12, 45, 0, 0, time.UTC)
	request := CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: "active", Next: "continue", Situation: "starting", At: at}

	if _, result := BuildCheckpointUpdate(target, request); !strings.Contains(result.InputError, "dependency is ready: story-01-base") {
		t.Fatalf("dependency gate result = %#v", result)
	}
	blocked, result := BuildCheckpointUpdate(target, CheckpointRequest{SpecRevision: "rev-1",
		Owner: "executor-a", Status: "blocked", Next: "wait for dependency", Situation: "dependency unavailable",
		Issue: "story-01-base is not ready", At: at,
	})
	if result.InputError != "" || len(result.Errors) != 0 || blocked.Status != "blocked" {
		t.Fatalf("blocked checkpoint = %#v, result = %#v", blocked, result)
	}

	storyFile := filepath.Join(cycleDirectory, "story-01-base", "story.md")
	story := readTestFile(t, storyFile)
	story = strings.Replace(story, "## Acceptance", "## Decision Ledger\n\n### Open Questions\n\n- Which account?\n\n## Acceptance", 1)
	writeFixtureFile(t, storyFile, story)
	if _, result := BuildCheckpointUpdate(filepath.Dir(storyFile), request); !strings.Contains(result.InputError, "open questions") {
		t.Fatalf("open question gate result = %#v", result)
	}
}

func TestBuildRevisionUpdateIncrementsCycleAndAffectedLanesOnce(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base": {},
		"story-02-help": {},
	})
	at := time.Date(2026, 9, 10, 13, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	for index, storyID := range []string{"story-01-base", "story-02-help"} {
		checkpoint, result := BuildCheckpointUpdate(filepath.Join(cycleDirectory, storyID), CheckpointRequest{SpecRevision: "rev-1",
			Owner: "executor-a", Status: "active", Next: "continue", Situation: "work started", At: at.Add(time.Duration(index) * time.Minute),
		})
		if result.InputError != "" || len(result.Errors) != 0 {
			t.Fatalf("checkpoint result = %#v", result)
		}
		if err := SaveViewUpdates([]ViewUpdate{checkpoint.Update}); err != nil {
			t.Fatal(err)
		}
	}

	revision, result := BuildRevisionUpdate([]string{
		filepath.Join(cycleDirectory, "story-01-base"),
		filepath.Join(cycleDirectory, "story-02-help"),
	}, RevisionRequest{Reason: "acceptance wording changed", At: at.Add(2 * time.Minute)})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if revision.NewRevision != "rev-2" || len(revision.StoryIDs) != 2 {
		t.Fatalf("revision = %#v", revision)
	}
	if err := SaveViewUpdates([]ViewUpdate{revision.Update}); err != nil {
		t.Fatal(err)
	}

	state := readTestFile(t, filepath.Join(cycleDirectory, "cycle.md"))
	if strings.Count(state, "  - Spec Changes: 1") != 2 {
		t.Fatalf("state = %s", state)
	}
	for _, fragment := range []string{
		"spec_rev: rev-2",
		"Acceptance: pending: specification revised at rev-2; acceptance wording changed",
		"Specification updated to rev-2; acceptance wording changed",
	} {
		if !strings.Contains(state, fragment) {
			t.Fatalf("state missing %q:\n%s", fragment, state)
		}
	}
	if validated := ValidateSpec(cycleDirectory, false); len(validated.Errors) != 0 {
		t.Fatalf("errors = %#v", validated.Errors)
	}
}

func TestBuildRevisionUpdateSynchronizesChangedDependencyViews(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{
		"story-01-base": {},
		"story-02-help": {},
	})
	storyFile := filepath.Join(cycleDirectory, "story-02-help", "story.md")
	story := strings.Replace(readTestFile(t, storyFile), "depends_on: []", "depends_on: [story-01-base]", 1)
	writeFixtureFile(t, storyFile, story)

	revision, result := BuildRevisionUpdate(
		[]string{filepath.Join(cycleDirectory, "story-02-help")},
		RevisionRequest{Reason: "dependency changed", At: time.Date(2026, 9, 10, 13, 30, 0, 0, time.UTC)},
	)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if err := SaveViewUpdates(revision.Updates); err != nil {
		t.Fatal(err)
	}

	cycle := readTestFile(t, filepath.Join(cycleDirectory, "cycle.md"))
	if !strings.Contains(cycle, "S01 --> S02") || !strings.Contains(cycle, "spec_rev: rev-2") {
		t.Fatalf("Cycle dependency or revision was not updated:\n%s", cycle)
	}
	if validated := ValidateSpec(cycleDirectory, false); len(validated.Errors) != 0 {
		t.Fatalf("errors = %#v", validated.Errors)
	}
}

func TestBuildLifecycleUpdatesRejectUnsafeState(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	stateFile := filepath.Join(cycleDirectory, "cycle.md")
	state := strings.Replace(readTestFile(t, stateFile), "Acceptance: pending:", "Acceptance: ready:", 1)
	writeFixtureFile(t, stateFile, state)
	target := filepath.Join(cycleDirectory, "story-01-base")
	at := time.Date(2026, 9, 10, 14, 0, 0, 0, time.UTC)

	if _, result := BuildCheckpointUpdate(target, CheckpointRequest{SpecRevision: "rev-1", Owner: "executor-a", Status: "active", Next: "continue", Situation: "work", At: at}); !strings.Contains(result.InputError, "cannot checkpoint ready as active") {
		t.Fatalf("checkpoint result = %#v", result)
	}
	if _, result := BuildRevisionUpdate([]string{target, target}, RevisionRequest{Reason: "change", At: at}); !strings.Contains(result.InputError, "duplicate revised Story") {
		t.Fatalf("revision result = %#v", result)
	}
}

func TestValidateActiveLanesRejectsDuplicateStory(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	stateFile := filepath.Join(cycleDirectory, "cycle.md")
	state := strings.Replace(readTestFile(t, stateFile), "## Active Lanes\n\n- none", `## Active Lanes

- Lane: lane-01
  - Story: story-01-base
  - Owner: executor-a
  - Status: active
  - Next: continue
- Lane: lane-02
  - Story: story-01-base
  - Owner: executor-b
  - Status: rejected
  - Next: wait`, 1)
	writeFixtureFile(t, stateFile, state)

	result := ValidateSpec(cycleDirectory, false)
	assertIssueContains(t, result.Errors, "must not have more than one active Lane")
}

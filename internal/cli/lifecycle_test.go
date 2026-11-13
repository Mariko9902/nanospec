package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCheckpointAndRevise(t *testing.T) {
	cycleDirectory := writeQueryFixture(t)
	target := filepath.Join(cycleDirectory, "story-01-base")

	var checkpointOut bytes.Buffer
	var checkpointErr bytes.Buffer
	checkpointCode := Run([]string{
		"checkpoint",
		"--spec-revision", "rev-1",
		"--owner", "executor-a",
		"--status", "active",
		"--next", "implement A1",
		"--situation", "repository facts confirmed",
		target,
	}, &checkpointOut, &checkpointErr)
	if checkpointCode != 0 || checkpointErr.Len() != 0 || !strings.Contains(checkpointOut.String(), "Checkpoint saved:") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", checkpointCode, checkpointOut.String(), checkpointErr.String())
	}

	var reviseOut bytes.Buffer
	var reviseErr bytes.Buffer
	reviseCode := Run([]string{"revise", "--reason", "acceptance changed", target}, &reviseOut, &reviseErr)
	if reviseCode != 0 || reviseErr.Len() != 0 || !strings.Contains(reviseOut.String(), "spec_revision=rev-2") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", reviseCode, reviseOut.String(), reviseErr.String())
	}

	state := readQueryFile(t, filepath.Join(cycleDirectory, "cycle.md"))
	for _, fragment := range []string{"Spec Changes: 1", "spec_rev: rev-2", "Acceptance: pending: specification revised at rev-2"} {
		if !strings.Contains(state, fragment) {
			t.Fatalf("state missing %q:\n%s", fragment, state)
		}
	}
}

func TestRunLifecycleHelpAndRequiredArguments(t *testing.T) {
	for _, command := range []string{"checkpoint", "revise"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Run([]string{command, "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Usage: nanospec "+command) {
			t.Fatalf("command = %s, code = %d, stdout = %q", command, code, stdout.String())
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"checkpoint", "story-01-unused"}, &stdout, &stderr); code != 2 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

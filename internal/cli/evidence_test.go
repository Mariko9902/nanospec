package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessLifecycleContract(t *testing.T) {
	cycleDirectory := writeQueryFixture(t)
	target := filepath.Join(cycleDirectory, "story-01-base")
	var prepareOut bytes.Buffer
	var prepareErr bytes.Buffer
	if code := RunWithExecutionRules([]string{"prepare", target}, &prepareOut, &prepareErr, "# Rules\n\nUse current facts."); code != 0 || !strings.Contains(prepareOut.String(), "ready_for_confirmation") {
		t.Fatalf("prepare code = %d, stdout = %q, stderr = %q", code, prepareOut.String(), prepareErr.String())
	}
	commands := [][]string{
		{"checkpoint", "--owner", "executor-a", "--status", "done", "--next", "none", "--situation", "implementation complete", target},
		{"evidence", "--implemented-by", "executor-a", "--acceptance", "A1", "--observed", "result is visible", "--command", "go test ./...", "--result", "check passed", "--artifacts", "artifacts/a1.txt", "--context", "rev-1 local at 2026-09-10T10:00:00+08:00", target},
		{"verify", "--verified-by", "executor-a", "--mode", "self", "--result", "pass", target},
		{"accept", "--accepted-by", "product-owner", "--method", "human", target},
	}
	for _, args := range commands {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("args = %#v, code = %d, stdout = %q, stderr = %q", args, code, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(filepath.Join(target, "acceptance", "evidence.md")); !os.IsNotExist(err) {
		t.Fatalf("evidence.md still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "acceptance", "outcome.md")); err != nil {
		t.Fatal(err)
	}
	state := readQueryFile(t, filepath.Join(cycleDirectory, "execution-state.md"))
	if !strings.Contains(state, "Acceptance: accepted:") || !strings.Contains(state, "- [x] story-01-base") {
		t.Fatalf("state = %s", state)
	}
}

func TestRunEvidenceCommandsHelp(t *testing.T) {
	for _, command := range []string{"evidence", "verify", "accept", "reject"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Run([]string{command, "--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "Usage: nanospec "+command) {
			t.Fatalf("command = %s, code = %d, stdout = %q", command, code, stdout.String())
		}
	}
}

func TestRunRejectReturnsReadyDeliveryToRework(t *testing.T) {
	cycleDirectory := writeQueryFixture(t)
	target := filepath.Join(cycleDirectory, "story-01-base")
	commands := [][]string{
		{"checkpoint", "--owner", "executor-a", "--status", "done", "--next", "none", "--situation", "implementation complete", target},
		{"evidence", "--implemented-by", "executor-a", "--acceptance", "A1", "--observed", "result is visible", "--command", "go test ./...", "--result", "check passed", "--artifacts", "artifacts/a1.txt", "--context", "rev-1 local at 2026-09-10T10:00:00+08:00", target},
		{"verify", "--verified-by", "executor-a", "--mode", "self", "--result", "pass", target},
		{"reject", "--rejected-by", "product-owner", "--reason", "result needs revision", target},
	}
	for _, args := range commands {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("args = %#v, code = %d, stdout = %q, stderr = %q", args, code, stdout.String(), stderr.String())
		}
	}
	state := readQueryFile(t, filepath.Join(cycleDirectory, "execution-state.md"))
	evidence := readQueryFile(t, filepath.Join(target, "acceptance", "evidence.md"))
	if !strings.Contains(state, "Acceptance: rejected:") || !strings.Contains(evidence, "verification_status: rework") {
		t.Fatalf("state = %s\nevidence = %s", state, evidence)
	}
}

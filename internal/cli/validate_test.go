package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidateReportsWarnings(t *testing.T) {
	cycleDirectory := writeQueryFixture(t)
	if err := os.WriteFile(filepath.Join(cycleDirectory, "story-01-base", "attachment.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"validate", "--sync-views", cycleDirectory}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "WARN") || !strings.Contains(stdout.String(), "Spec validation passed") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestRunValidateRejectsSpecRootWithoutChangingCycle(t *testing.T) {
	cycleDirectory := writeQueryFixture(t)
	cycleFile := filepath.Join(cycleDirectory, "cycle.md")
	stateFile := filepath.Join(cycleDirectory, "execution-state.md")
	beforeCycle := readQueryFile(t, cycleFile)
	beforeState := readQueryFile(t, stateFile)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"validate", "--sync-views", filepath.Dir(cycleDirectory)}, &stdout, &stderr)

	if code != 2 || !strings.Contains(stderr.String(), "target must be one Cycle directory") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if readQueryFile(t, cycleFile) != beforeCycle || readQueryFile(t, stateFile) != beforeState {
		t.Fatal("invalid target changed Cycle files")
	}
}

func TestRunValidateRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"validate", "--unknown"},
		{"validate", "--sync-views", "--sync-views"},
		{"validate", "first", "second"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := Run(args, &stdout, &stderr)
		if code != 2 || !strings.Contains(stderr.String(), "Usage: nanospec validate") {
			t.Fatalf("args = %#v, code = %d, stderr = %q", args, code, stderr.String())
		}
	}
}

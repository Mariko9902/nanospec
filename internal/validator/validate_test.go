package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSpecRejectsInvalidCycleDirectoryName(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	draft := filepath.Join(filepath.Dir(cycleDirectory), "draft-cycle")
	if err := os.Rename(cycleDirectory, draft); err != nil {
		t.Fatal(err)
	}

	result := ValidateSpec(draft, true)

	assertIssueContains(t, result.Errors, "cycle directory must match cycle-NN-slug")
}

func TestValidateSpecRejectsRootDirectories(t *testing.T) {
	cycleDirectory := writeCycleFixture(t, map[string][]string{"story-01-base": {}})
	for _, target := range []string{t.TempDir(), filepath.Dir(cycleDirectory)} {
		for _, syncViews := range []bool{false, true} {
			result := ValidateSpec(target, syncViews)
			if !strings.Contains(result.InputError, "target must be one Cycle directory") || len(result.Updates) != 0 {
				t.Fatalf("target = %s, syncViews = %t, result = %#v", target, syncViews, result)
			}
		}
	}
}

func TestValidateSpecReportsMissingCycleFiles(t *testing.T) {
	cycleDirectory := filepath.Join(t.TempDir(), "cycle-01-empty")
	if err := os.Mkdir(cycleDirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	result := ValidateSpec(cycleDirectory, false)

	if result.InputError != "" {
		t.Fatalf("InputError = %q", result.InputError)
	}
	assertIssueContains(t, result.Errors, "missing cycle.md")
}

package validator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveViewUpdatesCreatesFileAndRejectsStaleSource(t *testing.T) {
	directory := t.TempDir()
	existing := filepath.Join(directory, "existing.md")
	created := filepath.Join(directory, "created.md")
	writeFixtureFile(t, existing, "before")

	if err := SaveViewUpdates([]ViewUpdate{
		{File: existing, Before: "before", After: "after"},
		{File: created, Before: "", After: "new"},
	}); err != nil {
		t.Fatal(err)
	}
	if readTestFile(t, existing) != "after" || readTestFile(t, created) != "new" {
		t.Fatal("updates were not saved")
	}
	if err := SaveViewUpdates([]ViewUpdate{{File: existing, Before: "before", After: "other"}}); err == nil {
		t.Fatal("stale source was accepted")
	}
}

func TestSaveViewUpdatesRollsBackAppliedFiles(t *testing.T) {
	directory := t.TempDir()
	first := filepath.Join(directory, "first.md")
	second := filepath.Join(directory, "second.md")
	writeFixtureFile(t, first, "first-before")
	writeFixtureFile(t, second, "second-before")

	renames := 0
	failingRename := func(oldPath, newPath string) error {
		renames++
		if renames == 4 {
			return errors.New("injected rename failure")
		}
		return os.Rename(oldPath, newPath)
	}
	err := saveViewUpdates([]ViewUpdate{
		{File: first, Before: "first-before", After: "first-after"},
		{File: second, Before: "second-before", After: "second-after"},
	}, failingRename)
	if err == nil {
		t.Fatal("expected injected failure")
	}
	if readTestFile(t, first) != "first-before" || readTestFile(t, second) != "second-before" {
		t.Fatal("failed transaction did not restore original files")
	}
}

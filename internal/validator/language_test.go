package validator

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecLanguageSurvivesViewSyncAndPreparation(t *testing.T) {
	cases := []struct {
		name     string
		metadata string
		language string
		title    string
	}{
		{"default", "", "en-US", "Save an order"},
		{"english", "spec_language: en-US\n", "en-US", "Save an order"},
		{"chinese", "spec_language: zh-CN\n", "zh-CN", "保存订单"},
		{"japanese", "spec_language: ja-JP\n", "ja-JP", "注文の保存"},
		{"portuguese", "spec_language: pt-BR\n", "pt-BR", "Salvar um pedido"},
		{"arabic", "spec_language: ar-EG\n", "ar-EG", "حفظ الطلب"},
		{"language name", "spec_language: 日本語 (日本)\n", "日本語 (日本)", "注文の保存"},
		{"quoted", "spec_language: 'en-GB'\n", "en-GB", "Save an order"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
			cycleFile := filepath.Join(directory, "cycle.md")
			cycle := strings.Replace(readTestFile(t, cycleFile), "window:\n", test.metadata+"window:\n", 1)
			writeFixtureFile(t, cycleFile, cycle)
			storyFile := filepath.Join(directory, "story-01-order", "story.md")
			story := strings.Replace(readTestFile(t, storyFile), "- Title: Save an order", "- Title: "+test.title, 1)
			writeFixtureFile(t, storyFile, story)

			views := ValidateSpec(directory, true)
			if views.InputError != "" || len(views.Errors) != 0 {
				t.Fatalf("validation = %#v", views)
			}
			if err := SaveViewUpdates(views.Updates); err != nil {
				t.Fatal(err)
			}
			if validation := ValidateSpec(directory, false); len(validation.Errors) != 0 {
				t.Fatalf("synced validation = %#v", validation)
			}
			before := readTestFile(t, cycleFile)
			snapshot, validation := LoadCycleSnapshot(directory)
			if validation.InputError != "" || len(validation.Errors) != 0 || snapshot.SpecLanguage != test.language {
				t.Fatalf("snapshot = %#v, validation = %#v", snapshot, validation)
			}
			overview, err := snapshot.View("overview")
			if err != nil || !strings.Contains(overview, test.title) || !strings.Contains(before, test.title) {
				t.Fatalf("overview = %q, err = %v, cycle = %s", overview, err, before)
			}
			prepared, validation := BuildExecutionPackage(storyFile, "# Execution Rules\n\nUse current facts.")
			if validation.InputError != "" || len(validation.Errors) != 0 || prepared.SpecLanguage != test.language {
				t.Fatalf("package = %#v, validation = %#v", prepared, validation)
			}
			if !strings.Contains(prepared.ContextMarkdown, "- Spec Language: "+test.language+"\n") || !strings.Contains(prepared.ContextMarkdown, story) {
				t.Fatalf("context = %s", prepared.ContextMarkdown)
			}
			if readTestFile(t, cycleFile) != before || readTestFile(t, storyFile) != story {
				t.Fatal("preparation changed the language setting or Story")
			}
		})
	}
}

func TestSpecLanguageRejectsEmptyAndListValues(t *testing.T) {
	for _, value := range []string{"", "''", "[]", "[en-US]", "[en-US, ja-JP]", "none", "<language>"} {
		t.Run(value, func(t *testing.T) {
			directory := writeCycleFixture(t, map[string][]string{"story-01-order": {}})
			cycleFile := filepath.Join(directory, "cycle.md")
			before := strings.Replace(readTestFile(t, cycleFile), "window:\n", "spec_language: "+value+"\nwindow:\n", 1)
			writeFixtureFile(t, cycleFile, before)

			result := ValidateSpec(directory, true)
			assertDiagnostic(t, result.Errors, "cycle.spec_language must be a non-empty language name or tag")
			if len(result.Updates) != 0 || readTestFile(t, cycleFile) != before {
				t.Fatal("invalid language produced file updates")
			}
			_, result = BuildExecutionPackage(filepath.Join(directory, "story-01-order"), "# Rules\n\nUse current facts.")
			assertDiagnostic(t, result.Errors, "cycle.spec_language must be a non-empty language name or tag")
		})
	}
}

package validator

import (
	"os"
	"path/filepath"

	"nanospec/internal/diagnostic"
)

// ValidateSpec validates one Cycle. It never writes files.
func ValidateSpec(target string, syncViews bool) ValidationResult {
	return validateSpec(target, syncViews, true)
}

func validateSpec(target string, syncViews, validateEvidenceRecords bool) ValidationResult {
	var report diagnostic.Report
	info, err := os.Stat(target)
	if os.IsNotExist(err) {
		return ValidationResult{InputError: "target does not exist: " + target}
	}
	if err != nil {
		return ValidationResult{InputError: "cannot inspect target: " + err.Error()}
	}
	if !info.IsDir() {
		return ValidationResult{InputError: "target is not a directory: " + target}
	}
	if !isCycleDirectory(target) {
		return ValidationResult{InputError: "target must be one Cycle directory: " + target}
	}

	var updates []ViewUpdate
	var updateTarget *[]ViewUpdate
	if syncViews {
		updates = make([]ViewUpdate, 0)
		updateTarget = &updates
	}

	validateCycle(target, updateTarget, validateEvidenceRecords, &report)
	if report.HasErrors() {
		updates = nil
	}
	return ValidationResult{
		Errors:   report.Errors(),
		Warnings: report.Warnings(),
		Updates:  updates,
	}
}

func isCycleDirectory(directory string) bool {
	return cycleIDPattern.MatchString(filepath.Base(directory)) ||
		fileExists(filepath.Join(directory, "cycle.md"))
}

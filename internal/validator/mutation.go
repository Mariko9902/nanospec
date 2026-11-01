package validator

import (
	"os"
	"path/filepath"

	"nanospec/internal/diagnostic"
)

func SaveStoryCheckpoint(target string, request CheckpointRequest) (CheckpointResult, ValidationResult) {
	return mutateStory(target, func() (CheckpointResult, ValidationResult) {
		return BuildCheckpointUpdate(target, request)
	}, func(saved CheckpointResult) error {
		return saveViewUpdates([]ViewUpdate{saved.Update}, os.Rename)
	})
}

func ReviseStories(targets []string, request RevisionRequest) (RevisionResult, ValidationResult) {
	if len(targets) == 0 {
		return RevisionResult{}, ValidationResult{InputError: "revise requires at least one Story"}
	}
	return mutateStory(targets[0], func() (RevisionResult, ValidationResult) {
		return BuildRevisionUpdate(targets, request)
	}, func(saved RevisionResult) error {
		return saveViewUpdates(saved.Updates, os.Rename)
	})
}

func RecordStoryEvidence(target string, request EvidenceCaptureRequest) (EvidenceCaptureResult, ValidationResult) {
	return mutateStory(target, func() (EvidenceCaptureResult, ValidationResult) {
		return BuildEvidenceUpdate(target, request)
	}, func(saved EvidenceCaptureResult) error {
		return saveViewUpdates(saved.Updates, os.Rename)
	})
}

func VerifyStory(target string, request VerificationRequest) (VerificationResult, ValidationResult) {
	return mutateStory(target, func() (VerificationResult, ValidationResult) {
		return BuildVerificationUpdate(target, request)
	}, func(saved VerificationResult) error {
		return saveViewUpdates(saved.Updates, os.Rename)
	})
}

func RejectStory(target string, request RejectionRequest) (RejectionResult, ValidationResult) {
	return mutateStory(target, func() (RejectionResult, ValidationResult) {
		return BuildRejectionUpdate(target, request)
	}, func(saved RejectionResult) error {
		return saveViewUpdates(saved.Updates, os.Rename)
	})
}

func AcceptStory(target string, request AcceptanceRequest) (AcceptanceCommit, ValidationResult) {
	return mutateStory(target, func() (AcceptanceCommit, ValidationResult) {
		return BuildAcceptanceCommit(target, request)
	}, saveAcceptanceCommit)
}

func SynchronizeCycleViews(target string) ValidationResult {
	locks, err := lockWriteDirectories([]string{target})
	if err != nil {
		return mutationWriteResult(target, err)
	}
	defer locks.release()
	result := ValidateSpec(target, true)
	if result.InputError != "" || len(result.Errors) > 0 {
		return result
	}
	if err := saveViewUpdates(result.Updates, os.Rename); err != nil {
		return mutationWriteResult(target, err)
	}
	return result
}

func mutateStory[Result any](target string, build func() (Result, ValidationResult), save func(Result) error) (Result, ValidationResult) {
	var empty Result
	directory, inputError := resolveStoryDirectory(target)
	if inputError != "" {
		return empty, ValidationResult{InputError: inputError}
	}
	locks, err := lockWriteDirectories([]string{filepath.Dir(directory)})
	if err != nil {
		return empty, mutationWriteResult(target, err)
	}
	defer locks.release()
	saved, result := build()
	if result.InputError != "" || len(result.Errors) > 0 {
		return empty, result
	}
	if err := save(saved); err != nil {
		return empty, mutationWriteResult(target, err)
	}
	return saved, result
}

func mutationWriteResult(file string, err error) ValidationResult {
	if err == nil {
		return ValidationResult{}
	}
	return ValidationResult{Errors: []diagnostic.Issue{{Severity: diagnostic.SeverityError, File: file, Message: err.Error()}}}
}

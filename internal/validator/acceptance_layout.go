package validator

import (
	"fmt"
	"os"
	"path/filepath"

	"nanospec/internal/diagnostic"
)

const acceptanceDirectory = "acceptance"

func validateAcceptanceDirectory(directory string, report *diagnostic.Report) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		report.AddError(directory, "cannot list acceptance directory: "+err.Error())
		return
	}
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		if entry.Type()&os.ModeSymlink != 0 {
			report.AddError(path, "acceptance entries must not be symbolic links")
			continue
		}
		if entry.IsDir() && entry.Name() == "evidence" {
			continue
		}
		if !entry.IsDir() && (entry.Name() == "evidence.md" || entry.Name() == "outcome.md") {
			continue
		}
		report.AddError(path, "acceptance contains only evidence.md or outcome.md and the evidence directory")
	}
}

func createAcceptanceDirectories(updates []ViewUpdate) ([]string, error) {
	created := []string{}
	for _, update := range updates {
		parent := filepath.Dir(update.File)
		if filepath.Base(parent) != acceptanceDirectory || filepath.Base(update.File) != "evidence.md" {
			continue
		}
		for _, directory := range []string{parent, filepath.Join(parent, "evidence")} {
			info, err := os.Lstat(directory)
			if err == nil && info.IsDir() {
				continue
			}
			if err == nil {
				removeEmptyAcceptanceDirectories(created)
				return nil, fmt.Errorf("acceptance path is not a directory: %s", directory)
			}
			if !os.IsNotExist(err) {
				removeEmptyAcceptanceDirectories(created)
				return nil, err
			}
			if err := os.Mkdir(directory, 0o755); err != nil {
				removeEmptyAcceptanceDirectories(created)
				return nil, err
			}
			created = append(created, directory)
		}
	}
	return created, nil
}

func removeEmptyAcceptanceDirectories(directories []string) {
	for index := len(directories) - 1; index >= 0; index-- {
		_ = os.Remove(directories[index])
	}
}

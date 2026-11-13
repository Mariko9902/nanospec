package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nanospec/internal/specdoc"
)

type pendingViewFile struct {
	target    string
	temporary string
	backup    string
	existed   bool
	mode      os.FileMode
}

// RenderCycleViews builds the dependency graph and Cycle summary from current facts.
func RenderCycleViews(cycleText string, stories map[string]Story) (string, error) {
	if len(specdoc.ParseMarkdown(cycleText).Sections(2, "Acceptance Status")) != 1 {
		return "", fmt.Errorf("Acceptance Status must have exactly one heading before syncing views")
	}
	for id := range stories {
		if !storyIDPattern.MatchString(id) {
			return "", fmt.Errorf("invalid Story ID: %s", id)
		}
	}
	order, err := orderedStoryIDs(stories)
	if err != nil {
		return "", err
	}
	graph := renderDependencyGraph(order, stories, cycleText)
	updated, err := specdoc.ParseMarkdown(cycleText).ReplaceSection(2, "Dependency Graph", graph)
	if err != nil {
		return "", err
	}
	return specdoc.ReplaceFrontmatterScalar(updated, "status", deriveCycleStatus(extractAcceptanceBlocks(updated), extractLaneBlocks(updated)))
}

func renderDependencyGraph(order []string, stories map[string]Story, cycleText string) string {
	nodes := make(map[string]string)
	fence := strings.Repeat(string(rune(96)), 3)
	lines := []string{fence + "mermaid", "flowchart TD"}
	acceptance := extractAcceptanceBlocks(cycleText)
	lanes := extractLaneBlocks(cycleText)
	for _, id := range order {
		number := storyIDPattern.FindStringSubmatch(id)[1]
		name := "S" + number
		nodes[id] = name
		status := storyStatus(id, acceptance, lanes)
		title := strings.TrimRight(strings.TrimSpace(stories[id].Title), "。. ")
		label := "[" + string(status) + "] " + escapeMermaidLabel(name+" "+title)
		lines = append(lines, "  "+name+"[\""+label+"\"]:::"+string(status))
	}
	for _, id := range order {
		needed := make(map[string]bool)
		for _, dependency := range MinimalDependencies(stories, id) {
			needed[dependency] = true
		}
		for _, source := range order {
			if needed[source] {
				lines = append(lines, "  "+nodes[source]+" --> "+nodes[id])
			}
		}
	}
	lines = append(lines,
		"  classDef pending fill:#f3f4f6,stroke:#6b7280,color:#111827",
		"  classDef active fill:#dbeafe,stroke:#2563eb,color:#1e3a8a",
		"  classDef blocked fill:#fee2e2,stroke:#dc2626,color:#7f1d1d",
		"  classDef ready fill:#ede9fe,stroke:#7c3aed,color:#4c1d95",
		"  classDef rejected fill:#fce7f3,stroke:#be185d,color:#831843",
		"  classDef rework fill:#ffedd5,stroke:#ea580c,color:#7c2d12",
		"  classDef accepted fill:#dcfce7,stroke:#16a34a,color:#14532d", fence)
	return strings.Join(lines, "\n")
}

func escapeMermaidLabel(value string) string {
	var result strings.Builder
	for _, character := range value {
		if strings.ContainsRune("\"<>[]&#\\\x60", character) {
			fmt.Fprintf(&result, "#%d;", character)
		} else {
			result.WriteRune(character)
		}
	}
	return result.String()
}

// SaveViewUpdates saves all changes after checking that their sources still match.
func SaveViewUpdates(updates []ViewUpdate) error {
	locks, err := lockViewUpdates(updates)
	if err != nil {
		return err
	}
	defer locks.release()
	return saveViewUpdates(updates, os.Rename)
}

func saveViewUpdates(updates []ViewUpdate, renameFile func(string, string) error) (resultErr error) {
	seen := make(map[string]struct{}, len(updates))
	pending := make([]pendingViewFile, 0, len(updates))
	for _, update := range updates {
		if _, duplicate := seen[update.File]; duplicate {
			return fmt.Errorf("duplicate update target: %s", update.File)
		}
		seen[update.File] = struct{}{}
		content, err := os.ReadFile(update.File)
		existed := err == nil
		if os.IsNotExist(err) && update.Before == "" {
			existed = false
		} else if err != nil {
			return err
		}
		if existed && string(content) != update.Before {
			return fmt.Errorf("file changed during validation: %s", update.File)
		}
		if !existed && update.Before != "" {
			return fmt.Errorf("file changed during validation: %s", update.File)
		}
		mode := os.FileMode(0o644)
		if existed {
			info, statErr := os.Stat(update.File)
			if statErr != nil {
				return statErr
			}
			mode = info.Mode()
		}
		pending = append(pending, pendingViewFile{target: update.File, existed: existed, mode: mode})
	}

	created, err := createAcceptanceDirectories(updates)
	if err != nil {
		return err
	}
	defer func() {
		if resultErr != nil {
			removeEmptyAcceptanceDirectories(created)
		}
	}()
	for index, update := range updates {
		temporary, err := os.CreateTemp(filepath.Dir(update.File), "."+filepath.Base(update.File)+".tmp-")
		if err != nil {
			cleanupPendingFiles(pending)
			return err
		}
		temporaryName := temporary.Name()
		if _, err = temporary.WriteString(update.After); err == nil {
			err = temporary.Sync()
		}
		closeErr := temporary.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Chmod(temporaryName, pending[index].mode)
		}
		if err != nil {
			_ = os.Remove(temporaryName)
			cleanupPendingFiles(pending)
			return err
		}
		pending[index].temporary = temporaryName
	}

	applied := 0
	for index := range pending {
		file := &pending[index]
		if file.existed {
			backup, err := reserveBackupName(file.target)
			if err != nil {
				rollbackViewUpdates(pending, applied, renameFile)
				cleanupPendingFiles(pending)
				return err
			}
			file.backup = backup
			if err := renameFile(file.target, file.backup); err != nil {
				rollbackViewUpdates(pending, applied, renameFile)
				cleanupPendingFiles(pending)
				return err
			}
		}
		if err := renameFile(file.temporary, file.target); err != nil {
			if file.existed {
				_ = renameFile(file.backup, file.target)
				file.backup = ""
			}
			rollbackViewUpdates(pending, applied, renameFile)
			cleanupPendingFiles(pending)
			return err
		}
		file.temporary = ""
		applied++
	}
	for index := range pending {
		if pending[index].backup != "" {
			_ = os.Remove(pending[index].backup)
			pending[index].backup = ""
		}
	}
	return nil
}

func reserveBackupName(target string) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".backup-")
	if err != nil {
		return "", err
	}
	name := file.Name()
	if closeErr := file.Close(); closeErr != nil {
		_ = os.Remove(name)
		return "", closeErr
	}
	if err := os.Remove(name); err != nil {
		return "", err
	}
	return name, nil
}

func rollbackViewUpdates(files []pendingViewFile, applied int, renameFile func(string, string) error) {
	for index := applied - 1; index >= 0; index-- {
		file := &files[index]
		if file.existed && file.backup != "" {
			_ = renameFile(file.backup, file.target)
			file.backup = ""
			continue
		}
		if !file.existed {
			_ = os.Remove(file.target)
		}
	}
}

func cleanupPendingFiles(files []pendingViewFile) {
	for _, file := range files {
		if file.temporary != "" {
			_ = os.Remove(file.temporary)
		}
		if file.backup != "" {
			_ = os.Remove(file.backup)
		}
	}
}

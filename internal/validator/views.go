package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"nanospec/internal/specdoc"
)

// CycleViews holds the Cycle and state views generated from the same facts.
type CycleViews struct {
	Cycle string
	State string
}

type pendingViewFile struct {
	target    string
	temporary string
	backup    string
	existed   bool
	mode      os.FileMode
}

// RenderCycleViews rebuilds Waves, the dependency graph, and the progress tree.
func RenderCycleViews(cycleText, stateText string, stories map[string]Story) (CycleViews, error) {
	if len(specdoc.ParseMarkdown(cycleText).Sections(3, "Dependency Waves")) != 1 {
		return CycleViews{}, fmt.Errorf("Dependency Waves must have exactly one heading before syncing views")
	}
	if len(specdoc.ParseMarkdown(stateText).Sections(2, "Acceptance Status")) != 1 {
		return CycleViews{}, fmt.Errorf("Acceptance Status must have exactly one heading before syncing views")
	}

	waves := readDependencyWaves(cycleText)
	if err := requireCompletePlan(waves, stories); err != nil {
		return CycleViews{}, err
	}
	grouped, err := groupDependencyWaves(waves, stories)
	if err != nil {
		return CycleViews{}, err
	}
	entries := flattenWaveEntries(grouped)
	statuses := make(map[string]string)
	for _, block := range extractAcceptanceBlocks(stateText) {
		statuses[block.ID] = leadingToken(block.Fields["Acceptance"])
	}

	progressLines := make([]string, 0, len(entries))
	for _, entry := range entries {
		mark := " "
		if statuses[entry.ID] == "accepted" {
			mark = "x"
		}
		progressLines = append(progressLines, "- ["+mark+"] "+entry.ID)
	}

	updatedCycle, err := specdoc.ParseMarkdown(cycleText).ReplaceSection(3, "Dependency Waves", renderDependencyWaves(grouped, stories))
	if err != nil {
		return CycleViews{}, err
	}
	graph, err := renderDependencyGraph(entries, stories)
	if err != nil {
		return CycleViews{}, err
	}
	updatedCycle, err = specdoc.ParseMarkdown(updatedCycle).ReplaceSection(3, "Dependency Graph", graph)
	if err != nil {
		return CycleViews{}, err
	}
	updatedState, err := specdoc.ParseMarkdown(stateText).ReplaceSection(2, "Progress Tree", strings.Join(progressLines, "\n"))
	if err != nil {
		return CycleViews{}, err
	}
	return CycleViews{Cycle: updatedCycle, State: updatedState}, nil
}

func requireCompletePlan(waves []DependencyWave, stories map[string]Story) error {
	seen := make(map[string]struct{})
	for _, entry := range flattenWaveEntries(waves) {
		if _, found := stories[entry.ID]; !found {
			return fmt.Errorf("Dependency Waves must contain each Story exactly once: %s", entry.ID)
		}
		if _, duplicate := seen[entry.ID]; duplicate {
			return fmt.Errorf("Dependency Waves must contain each Story exactly once: %s", entry.ID)
		}
		seen[entry.ID] = struct{}{}
	}
	for _, storyID := range sortedStoryIDs(stories) {
		if _, found := seen[storyID]; !found {
			return fmt.Errorf("Dependency Waves missing story: %s", storyID)
		}
	}
	return nil
}

func groupDependencyWaves(waves []DependencyWave, stories map[string]Story) ([]DependencyWave, error) {
	numbers, err := DependencyWaveNumbers(stories)
	if err != nil {
		return nil, err
	}
	grouped := make(map[int][]WaveEntry)
	for _, entry := range flattenWaveEntries(waves) {
		number, found := numbers[entry.ID]
		if !found {
			return nil, fmt.Errorf("Dependency Waves references missing story: %s", entry.ID)
		}
		grouped[number] = append(grouped[number], entry)
	}
	keys := make([]int, 0, len(grouped))
	for number := range grouped {
		keys = append(keys, number)
	}
	sort.Ints(keys)
	result := make([]DependencyWave, 0, len(keys))
	for _, number := range keys {
		result = append(result, DependencyWave{Number: number, Entries: grouped[number]})
	}
	return result, nil
}

func flattenWaveEntries(waves []DependencyWave) []WaveEntry {
	entries := make([]WaveEntry, 0)
	for _, wave := range waves {
		entries = append(entries, wave.Entries...)
	}
	return entries
}

func renderDependencyWaves(waves []DependencyWave, stories map[string]Story) string {
	quote := string(rune(96))
	sections := make([]string, 0, len(waves))
	for _, wave := range waves {
		lines := []string{fmt.Sprintf("#### Wave %d", wave.Number), ""}
		for index, entry := range wave.Entries {
			line := fmt.Sprintf("%d. %s%s%s: %s", index+1, quote, entry.ID, quote, stories[entry.ID].Title)
			if note := waveEntryNote(entry.Tail); note != "" {
				line += "; " + note
			}
			lines = append(lines, line)
			notes := entry.Notes
			for len(notes) > 0 && strings.TrimSpace(notes[0]) == "" {
				notes = notes[1:]
			}
			for len(notes) > 0 && strings.TrimSpace(notes[len(notes)-1]) == "" {
				notes = notes[:len(notes)-1]
			}
			if len(notes) > 0 {
				lines = append(lines, "")
				lines = append(lines, notes...)
				if index < len(wave.Entries)-1 {
					lines = append(lines, "")
				}
			}
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.Join(sections, "\n\n")
}

func waveEntryNote(tail string) string {
	tail = strings.TrimSpace(tail)
	if separator := strings.IndexAny(tail, ";；"); separator >= 0 {
		_, size := utf8.DecodeRuneInString(tail[separator:])
		return strings.TrimSpace(tail[separator+size:])
	}
	if strings.HasPrefix(tail, ":") || strings.HasPrefix(tail, "：") {
		return ""
	}
	return strings.TrimLeft(tail, ";； ")
}

func renderDependencyGraph(entries []WaveEntry, stories map[string]Story) (string, error) {
	hasDependencies := false
	for _, entry := range entries {
		if len(stories[entry.ID].DependsOn) > 0 {
			hasDependencies = true
			break
		}
	}
	if !hasDependencies {
		return "- none", nil
	}

	type graphNode struct {
		name  string
		label string
	}
	nodes := make(map[string]graphNode)
	fence := strings.Repeat(string(rune(96)), 3)
	lines := []string{fence + "mermaid", "flowchart TD"}
	for _, entry := range entries {
		number := entry.ID
		if match := storyIDPattern.FindStringSubmatch(entry.ID); len(match) > 1 {
			number = match[1]
		}
		name := strings.TrimRight(strings.TrimSpace(stories[entry.ID].Title), "。. ")
		label := "S" + number
		if name != "" {
			label += " " + name
		}
		node := graphNode{name: "S" + number, label: label}
		nodes[entry.ID] = node
		lines = append(lines, "  "+node.name+"[\""+escapeMermaidLabel(node.label)+"\"]")
	}
	lines = append(lines, "")
	for _, entry := range entries {
		dependencies := make(map[string]struct{})
		for _, dependency := range MinimalDependencies(stories, entry.ID) {
			if _, found := nodes[dependency]; !found {
				return "", fmt.Errorf("story.depends_on references missing story: %s", dependency)
			}
			dependencies[dependency] = struct{}{}
		}
		for _, source := range entries {
			if _, found := dependencies[source.ID]; found {
				lines = append(lines, "  "+nodes[source.ID].name+" --> "+nodes[entry.ID].name)
			}
		}
	}
	lines = append(lines, fence)
	return strings.Join(lines, "\n"), nil
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

package validator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type graphOrderNode struct {
	id              string
	repeatsSequence bool
}

type orderIssue struct {
	scope   string
	message string
}

var (
	waveHeadingPattern   = regexp.MustCompile(`^####\s+Wave\s+([1-9]\d*)\s*$`)
	waveEntryPattern     = regexp.MustCompile(`^\s*\d+\.\s+\x60(story-[A-Za-z0-9-]+)\x60\s*(.*)$`)
	graphNodePattern     = regexp.MustCompile(`^\s*S(\d+)\["`)
	graphSequencePattern = regexp.MustCompile(`^\d+\.\s+story-`)
	progressEntryPattern = regexp.MustCompile(`^\s*-\s+\[[ xX]\]\s+story-[A-Za-z0-9-]+`)
	storyTokenPattern    = regexp.MustCompile(`^[A-Za-z0-9-]+`)
)

func readDependencyWaves(text string) []DependencyWave {
	waves := make([]DependencyWave, 0)
	current := -1
	for _, line := range strings.Split(strings.ReplaceAll(sectionContent(text, 3, "Dependency Waves"), "\r\n", "\n"), "\n") {
		if match := waveHeadingPattern.FindStringSubmatch(line); match != nil {
			number, _ := strconv.Atoi(match[1])
			waves = append(waves, DependencyWave{Number: number})
			current = len(waves) - 1
			continue
		}
		match := waveEntryPattern.FindStringSubmatch(line)
		if match == nil {
			if current >= 0 && len(waves[current].Entries) > 0 {
				entries := waves[current].Entries
				last := &entries[len(entries)-1]
				last.Notes = append(last.Notes, line)
			}
			continue
		}
		if current < 0 {
			waves = append(waves, DependencyWave{Number: 0})
			current = 0
		}
		tail := strings.TrimSpace(match[2])
		waves[current].Entries = append(waves[current].Entries, WaveEntry{ID: match[1], Tail: tail})
	}
	return waves
}

func validateOrder(cycleText, stateText string, stories map[string]Story) []orderIssue {
	waves := readDependencyWaves(cycleText)
	planned := flattenWaveIDs(waves)
	if len(planned) == 0 {
		return nil
	}
	progress := readProgressOrder(stateText)
	graph := readGraphOrder(cycleText, planned)
	issues := make([]orderIssue, 0)
	waveNumbers, _ := DependencyWaveNumbers(stories)

	for index, wave := range waves {
		expectedWave := index + 1
		if wave.Number != expectedWave || len(wave.Entries) == 0 {
			issues = append(issues, orderIssue{scope: "cycle", message: "Dependency Waves must use consecutive non-empty Wave headings starting at 1"})
		}
		for _, entry := range wave.Entries {
			if expected, found := waveNumbers[entry.ID]; found && expected != wave.Number {
				issues = append(issues, orderIssue{
					scope:   "cycle",
					message: fmt.Sprintf("Dependency Waves must place %s in Wave %d; actual Wave %d", entry.ID, expected, wave.Number),
				})
			}
		}
	}

	expected := strings.Join(planned, " -> ")
	if !sameOrder(planned, progress) {
		issues = append(issues, orderIssue{
			scope:   "state",
			message: "Progress Tree order must match Dependency Waves; expected " + expected + "; actual " + strings.Join(progress, " -> "),
		})
	}
	if len(graph) > 0 {
		actual := make([]string, 0, len(graph))
		for _, node := range graph {
			actual = append(actual, node.id)
		}
		if !sameOrder(planned, actual) {
			issues = append(issues, orderIssue{
				scope:   "cycle",
				message: "Dependency Graph node order must match Dependency Waves; expected " + expected + "; actual " + strings.Join(actual, " -> "),
			})
		}
		for _, node := range graph {
			if node.repeatsSequence {
				issues = append(issues, orderIssue{
					scope:   "cycle",
					message: "Dependency Graph node " + node.id + " must not repeat the Wave entry number before its Story ID.",
				})
			}
		}
	}
	return issues
}

func flattenWaveIDs(waves []DependencyWave) []string {
	ids := make([]string, 0)
	for _, wave := range waves {
		for _, entry := range wave.Entries {
			ids = append(ids, entry.ID)
		}
	}
	return ids
}

func readGraphOrder(text string, planned []string) []graphOrderNode {
	storyByNumber := make(map[string]string)
	numberPattern := regexp.MustCompile(`^story-(\d+)-`)
	for _, id := range planned {
		match := numberPattern.FindStringSubmatch(id)
		if len(match) < 2 {
			continue
		}
		number, _ := strconv.Atoi(match[1])
		storyByNumber[strconv.Itoa(number)] = id
	}

	graph := make([]graphOrderNode, 0)
	inGraph := false
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if line == "### Dependency Waves" {
			inGraph = false
			continue
		}
		if line == "### Dependency Graph" {
			inGraph = true
			continue
		}
		if inGraph && strings.HasPrefix(line, "## ") {
			inGraph = false
		}
		if !inGraph || !strings.Contains(line, "[\"") {
			continue
		}
		label := line[strings.Index(line, "[\"")+2:]
		id := extractStoryID(line)
		if match := graphNodePattern.FindStringSubmatch(line); id == "" && match != nil {
			number, _ := strconv.Atoi(match[1])
			id = storyByNumber[strconv.Itoa(number)]
		}
		if id != "" {
			graph = append(graph, graphOrderNode{id: id, repeatsSequence: graphSequencePattern.MatchString(label)})
		}
	}
	return graph
}

func readProgressOrder(text string) []string {
	ids := make([]string, 0)
	inProgress := false
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if line == "## Progress Tree" {
			inProgress = true
			continue
		}
		if inProgress && strings.HasPrefix(line, "## ") {
			inProgress = false
		}
		if inProgress && progressEntryPattern.MatchString(line) {
			ids = append(ids, extractStoryID(line))
		}
	}
	return ids
}

func extractStoryID(line string) string {
	start := strings.Index(line, "story-")
	if start < 0 {
		return ""
	}
	return storyTokenPattern.FindString(line[start:])
}

func sameOrder(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for index := range expected {
		if expected[index] != actual[index] {
			return false
		}
	}
	return true
}

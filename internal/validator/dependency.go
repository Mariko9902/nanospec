package validator

import (
	"fmt"
	"sort"

	"nanospec/internal/diagnostic"
)

// TransitiveDependency identifies a direct dependency already covered by another path.
type TransitiveDependency struct {
	Dependency string
	Via        string
}

// FindTransitiveDependencies finds redundant direct dependencies.
func FindTransitiveDependencies(stories map[string]Story, storyID string) []TransitiveDependency {
	dependencies := uniqueStrings(stories[storyID].DependsOn)
	result := make([]TransitiveDependency, 0)
	for _, dependency := range dependencies {
		for _, candidate := range dependencies {
			if candidate != dependency && reachesDependency(stories, candidate, dependency, map[string]struct{}{}) {
				result = append(result, TransitiveDependency{Dependency: dependency, Via: candidate})
				break
			}
		}
	}
	return result
}

// MinimalDependencies removes prerequisites already reachable through another direct dependency.
func MinimalDependencies(stories map[string]Story, storyID string) []string {
	redundant := make(map[string]struct{})
	for _, item := range FindTransitiveDependencies(stories, storyID) {
		redundant[item.Dependency] = struct{}{}
	}
	result := make([]string, 0)
	for _, dependency := range uniqueStrings(stories[storyID].DependsOn) {
		if _, found := redundant[dependency]; !found {
			result = append(result, dependency)
		}
	}
	return result
}

// DependencyWaveNumbers assigns Waves from the longest prerequisite path.
func DependencyWaveNumbers(stories map[string]Story) (map[string]int, error) {
	waves := make(map[string]int)
	visiting := make(map[string]struct{})
	var visit func(string) (int, error)
	visit = func(storyID string) (int, error) {
		if wave, found := waves[storyID]; found {
			return wave, nil
		}
		if _, found := visiting[storyID]; found {
			return 0, fmt.Errorf("story dependency cycle detected at %s", storyID)
		}
		story, found := stories[storyID]
		if !found {
			return 0, fmt.Errorf("story dependency references missing story: %s", storyID)
		}
		visiting[storyID] = struct{}{}
		wave := 1
		for _, dependency := range uniqueStrings(story.DependsOn) {
			dependencyWave, err := visit(dependency)
			if err != nil {
				return 0, err
			}
			if dependencyWave+1 > wave {
				wave = dependencyWave + 1
			}
		}
		delete(visiting, storyID)
		waves[storyID] = wave
		return wave, nil
	}

	for _, storyID := range sortedStoryIDs(stories) {
		if _, err := visit(storyID); err != nil {
			return nil, err
		}
	}
	return waves, nil
}

func validateStoryGraph(stories map[string]Story, report *diagnostic.Report) {
	for _, storyID := range sortedStoryIDs(stories) {
		story := stories[storyID]
		for _, dependency := range story.DependsOn {
			if _, found := stories[dependency]; !found {
				report.AddError(story.File, "story.depends_on references missing story: "+dependency)
			}
		}
	}

	visiting := make(map[string]struct{})
	visited := make(map[string]struct{})
	var visit func(string)
	visit = func(storyID string) {
		if _, found := visiting[storyID]; found {
			file := storyID
			if story, exists := stories[storyID]; exists {
				file = story.File
			}
			report.AddError(file, "story dependency cycle detected at "+storyID)
			return
		}
		if _, found := visited[storyID]; found {
			return
		}
		story, found := stories[storyID]
		if !found {
			return
		}
		visiting[storyID] = struct{}{}
		for _, dependency := range story.DependsOn {
			visit(dependency)
		}
		delete(visiting, storyID)
		visited[storyID] = struct{}{}
	}
	for _, storyID := range sortedStoryIDs(stories) {
		visit(storyID)
	}

	for _, storyID := range sortedStoryIDs(stories) {
		story := stories[storyID]
		for _, item := range FindTransitiveDependencies(stories, storyID) {
			report.AddWarning(story.File, "story.depends_on contains transitive dependency "+item.Dependency+"; already implied by "+item.Via)
		}
	}

	orders := make(map[string]string)
	for _, storyID := range sortedStoryIDs(stories) {
		story := stories[storyID]
		if story.Order == "" {
			continue
		}
		if previous, duplicate := orders[story.Order]; duplicate {
			report.AddError(story.File, "duplicate story number in cycle: "+story.Order)
			report.AddError(previous, "duplicate story number in cycle: "+story.Order)
		} else {
			orders[story.Order] = story.File
		}
	}
}

func reachesDependency(stories map[string]Story, storyID, dependencyID string, visited map[string]struct{}) bool {
	if _, found := visited[storyID]; found {
		return false
	}
	visited[storyID] = struct{}{}
	story, found := stories[storyID]
	if !found {
		return false
	}
	for _, dependency := range story.DependsOn {
		if dependency == dependencyID || reachesDependency(stories, dependency, dependencyID, visited) {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func sortedStoryIDs(stories map[string]Story) []string {
	ids := make([]string, 0, len(stories))
	for id := range stories {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

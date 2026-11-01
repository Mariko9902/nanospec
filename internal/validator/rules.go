package validator

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"nanospec/internal/diagnostic"
	"nanospec/internal/specdoc"
)

var (
	executorIDPattern     = regexp.MustCompile(`^[\pL\pN][\pL\pN._@/-]*$`)
	storyIDPattern        = regexp.MustCompile("^story-(0[1-9]|[1-9][0-9])-[a-z0-9][a-z0-9-]*$")
	cycleIDPattern        = regexp.MustCompile("^cycle-[0-9]{2}-[a-z0-9][a-z0-9-]*$")
	regressionRefPattern  = regexp.MustCompile("^(cycle-[0-9]{2}-[a-z0-9][a-z0-9-]*)/(story-(?:0[1-9]|[1-9][0-9])-[a-z0-9][a-z0-9-]*)$")
	timestampPattern      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$`)
	leadingTokenPattern   = regexp.MustCompile(`^([A-Za-z]+)`)
	storyReferencePattern = regexp.MustCompile(`\b(story-(?:0[1-9]|[1-9]\d)-[a-z0-9][a-z0-9-]*)\b`)
)

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func readText(file string, report *diagnostic.Report) string {
	content, err := os.ReadFile(file)
	if err != nil {
		report.AddError(file, "cannot read file: "+err.Error())
		return ""
	}
	return string(content)
}

func requireKeys(file string, frontmatter specdoc.Frontmatter, keys []string, report *diagnostic.Report) {
	for _, key := range keys {
		if !frontmatter.Has(key) {
			report.AddError(file, "frontmatter missing required key: "+key)
		}
	}
}

func validateAllowedKeys(file string, frontmatter specdoc.Frontmatter, allowed []string, scope string, report *diagnostic.Report) {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	for _, key := range frontmatter.Keys() {
		if _, ok := allowedSet[key]; !ok {
			report.AddError(file, scope+" frontmatter has unsupported key: "+key)
		}
	}
}

func section(body string, level int, name string) (specdoc.Section, bool) {
	return specdoc.ParseMarkdown(body).Section(level, name)
}

func hasSection(body string, level int, name string) bool {
	_, ok := section(body, level, name)
	return ok
}

func sectionContent(body string, level int, name string) string {
	found, ok := section(body, level, name)
	if !ok {
		return ""
	}
	return found.Content
}

func requireSections(file, body string, level int, names []string, report *diagnostic.Report) {
	markdown := specdoc.ParseMarkdown(body)
	for _, name := range names {
		if _, ok := markdown.Section(level, name); !ok {
			report.AddError(file, "missing required section: "+name)
		}
	}
}

func validateAllowedSections(file, body string, level int, allowed []string, scope string, report *diagnostic.Report) {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowedSet[name] = struct{}{}
	}
	seen := make(map[string]struct{})
	for _, heading := range specdoc.ParseMarkdown(body).Headings() {
		if heading.Level != level {
			continue
		}
		if _, ok := allowedSet[heading.Name]; !ok {
			report.AddError(file, scope+" has unsupported section: "+heading.Name)
			continue
		}
		if _, duplicate := seen[heading.Name]; duplicate {
			report.AddError(file, scope+" has duplicate section: "+heading.Name)
		}
		seen[heading.Name] = struct{}{}
	}
}

func requireSectionOrder(file, body string, level int, names []string, scope string, report *diagnostic.Report) {
	markdown := specdoc.ParseMarkdown(body)
	previous := -1
	for _, name := range names {
		found, ok := markdown.Section(level, name)
		if !ok {
			continue
		}
		if found.Heading.HeaderStart < previous {
			report.AddError(file, scope+" section out of order: "+name)
		}
		previous = found.Heading.HeaderStart
	}
}

func requireMeaningfulSection(file, body, name string, allowNone bool, report *diagnostic.Report) {
	content, ok := section(body, 2, name)
	if !ok {
		return
	}
	if allowNone && specdoc.IsExplicitNone(content.Content) {
		return
	}
	if !specdoc.Meaningful(content.Content) {
		report.AddError(file, "section must contain meaningful content: "+name)
	}
}

func warnEmptyOptionalSection(file, content, name string, report *diagnostic.Report) {
	if !specdoc.Meaningful(content) || specdoc.IsExplicitNone(content) {
		report.AddWarning(file, "optional section has no content: "+name)
	}
}

func isISOTimestamp(value string) bool {
	if !timestampPattern.MatchString(value) {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

func hasText(value string, found bool) bool {
	return found && specdoc.Meaningful(value) && !specdoc.IsExplicitNone(value)
}

func leadingToken(value string) string {
	match := leadingTokenPattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func extractStoryIDs(value string) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, match := range storyReferencePattern.FindAllStringSubmatch(value, -1) {
		ids[match[1]] = struct{}{}
	}
	return ids
}

func sortedMapKeys[Value any](values map[string]Value) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

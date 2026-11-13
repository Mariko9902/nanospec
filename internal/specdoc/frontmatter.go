package specdoc

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"nanospec/internal/diagnostic"
)

// ValueKind distinguishes a scalar from a list.
type ValueKind uint8

const (
	ScalarValue ValueKind = iota + 1
	ListValue
)

// Value is a string or a list of strings.
type Value struct {
	kind   ValueKind
	scalar string
	items  []string
}

// Kind returns the value kind.
func (v Value) Kind() ValueKind {
	return v.kind
}

// Scalar returns a scalar, or false for a list.
func (v Value) Scalar() (string, bool) {
	return v.scalar, v.kind == ScalarValue
}

// Items returns a copy of the list, or false for a scalar.
func (v Value) Items() ([]string, bool) {
	if v.kind != ListValue {
		return nil, false
	}
	items := make([]string, len(v.items))
	copy(items, v.items)
	return items, true
}

// Frontmatter holds the supported top-level metadata fields.
type Frontmatter struct {
	fields map[string]Value
}

// Has reports whether a field exists.
func (f Frontmatter) Has(key string) bool {
	_, ok := f.fields[key]
	return ok
}

// Get returns a field by name.
func (f Frontmatter) Get(key string) (Value, bool) {
	value, ok := f.fields[key]
	return value, ok
}

// Keys returns field names in alphabetical order.
func (f Frontmatter) Keys() []string {
	keys := make([]string, 0, len(f.fields))
	for key := range f.fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// String returns a scalar field, or false if the field is missing or is a list.
func (f Frontmatter) String(key string) (string, bool) {
	value, ok := f.Get(key)
	if !ok {
		return "", false
	}
	return value.Scalar()
}

// Strings returns a list, treating a scalar as one item and a missing field as empty.
func (f Frontmatter) Strings(key string) []string {
	value, ok := f.Get(key)
	if !ok {
		return []string{}
	}
	if items, list := value.Items(); list {
		return items
	}
	if scalar, ok := value.Scalar(); ok && scalar != "" {
		return []string{scalar}
	}
	return []string{}
}

// Document contains frontmatter and Markdown text.
type Document struct {
	Frontmatter Frontmatter
	Body        string
}

var (
	frontmatterPattern = regexp.MustCompile("(?s)^---\\r?\\n(.*?)\\r?\\n---(?:\\r?\\n|$)")
	yamlKeyPattern     = regexp.MustCompile("^([A-Za-z0-9_-]+):(?:\\s*(.*))?$")
	yamlItemPattern    = regexp.MustCompile("^\\s*-\\s*(.*)$")
)

// ParseDocument reads the opening frontmatter. Missing frontmatter is an error; the original text is kept.
func ParseDocument(file, source string, report *diagnostic.Report) Document {
	location := frontmatterPattern.FindStringSubmatchIndex(source)
	if location == nil {
		report.AddError(file, "missing YAML frontmatter")
		return Document{Frontmatter: Frontmatter{fields: map[string]Value{}}, Body: source}
	}

	metadataSource := source[location[2]:location[3]]
	return Document{
		Frontmatter: parseFrontmatter(metadataSource),
		Body:        source[location[1]:],
	}
}

// ReplaceFrontmatterScalar updates one metadata field while preserving the body.
func ReplaceFrontmatterScalar(source, key, value string) (string, error) {
	location := frontmatterPattern.FindStringSubmatchIndex(source)
	if location == nil {
		return "", fmt.Errorf("missing YAML frontmatter")
	}
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("frontmatter value must be one line")
	}
	metadata := source[location[2]:location[3]]
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:[^\r\n]*`)
	matches := pattern.FindAllStringIndex(metadata, -1)
	if len(matches) != 1 {
		return "", fmt.Errorf("frontmatter must contain exactly one %s field", key)
	}
	start, end := location[2]+matches[0][0], location[2]+matches[0][1]
	return source[:start] + key + ": " + value + source[end:], nil
}

func parseFrontmatter(source string) Frontmatter {
	fields := make(map[string]Value)
	currentKey := ""
	for _, line := range splitLines(source) {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if matches := yamlKeyPattern.FindStringSubmatch(line); matches != nil {
			currentKey = matches[1]
			text := strings.TrimSpace(stripYAMLComment(matches[2]))
			if text == "" {
				fields[currentKey] = Value{kind: ListValue, items: []string{}}
			} else {
				fields[currentKey] = parseYAMLValue(text)
			}
			continue
		}
		if matches := yamlItemPattern.FindStringSubmatch(line); matches != nil && currentKey != "" {
			value := fields[currentKey]
			if value.kind != ListValue {
				value = Value{kind: ListValue, items: []string{}}
			}
			item := strings.TrimSpace(stripQuotes(stripYAMLComment(matches[1])))
			if item != "" {
				value.items = append(value.items, item)
			}
			fields[currentKey] = value
		}
	}
	return Frontmatter{fields: fields}
}

func parseYAMLValue(source string) Value {
	if source == "[]" {
		return Value{kind: ListValue, items: []string{}}
	}
	if strings.HasPrefix(source, "[") && strings.HasSuffix(source, "]") {
		return Value{kind: ListValue, items: parseInlineList(source[1 : len(source)-1])}
	}
	return Value{kind: ScalarValue, scalar: strings.TrimSpace(stripQuotes(source))}
}

func parseInlineList(source string) []string {
	items := make([]string, 0)
	var current strings.Builder
	var quote rune
	for _, character := range source {
		if quote != 0 {
			if character == quote {
				quote = 0
			}
			current.WriteRune(character)
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			current.WriteRune(character)
			continue
		}
		if character == ',' {
			appendListItem(&items, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(character)
	}
	appendListItem(&items, current.String())
	return items
}

func appendListItem(items *[]string, source string) {
	item := strings.TrimSpace(stripQuotes(source))
	if item != "" {
		*items = append(*items, item)
	}
}

func stripQuotes(source string) string {
	text := strings.TrimSpace(source)
	if len(text) >= 2 && ((text[0] == '"' && text[len(text)-1] == '"') ||
		(text[0] == '\'' && text[len(text)-1] == '\'')) {
		return text[1 : len(text)-1]
	}
	return text
}

func stripYAMLComment(source string) string {
	var result strings.Builder
	var quote rune
	for _, character := range source {
		if quote != 0 {
			if character == quote {
				quote = 0
			}
			result.WriteRune(character)
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			result.WriteRune(character)
			continue
		}
		if character == '#' {
			break
		}
		result.WriteRune(character)
	}
	return result.String()
}

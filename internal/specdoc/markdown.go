package specdoc

import (
	"fmt"
	"regexp"
	"strings"
)

// Heading is an ATX heading outside a fenced code block.
type Heading struct {
	Level        int
	Name         string
	HeaderStart  int
	ContentStart int
}

// Markdown holds Markdown text and its heading index.
type Markdown struct {
	source   string
	headings []Heading
}

// Section runs from a heading to the next heading at the same or a higher level.
type Section struct {
	Heading Heading
	Content string
	Start   int
	End     int
}

type lineRange struct {
	start      int
	textEnd    int
	contentEnd int
}

var (
	headingPattern     = regexp.MustCompile("^(#{1,6})[\\t ]+(.+?)[\\t ]*$")
	fencePattern       = regexp.MustCompile("^[\\t ]*(\\x60{3,}|~{3,})")
	htmlCommentPattern = regexp.MustCompile("(?s)<!--.*?-->")
	htmlTagPattern     = regexp.MustCompile("<[^>\\n]+>")
	contentNoise       = regexp.MustCompile("[-*\\s\\[\\]\\x60|:#]")
	explicitNone       = regexp.MustCompile("(?i)^\\s*-?\\s*none\\s*$")
	lineBreakPattern   = regexp.MustCompile("\\r?\\n")
)

// ParseMarkdown indexes headings outside fenced code blocks.
func ParseMarkdown(source string) Markdown {
	headings := make([]Heading, 0)
	fenceCharacter := byte(0)
	fenceLength := 0

	for _, line := range sourceLines(source) {
		text := source[line.start:line.textEnd]
		if matches := fencePattern.FindStringSubmatch(text); matches != nil {
			marker := matches[1]
			if fenceCharacter == 0 {
				fenceCharacter = marker[0]
				fenceLength = len(marker)
			} else if marker[0] == fenceCharacter && len(marker) >= fenceLength {
				fenceCharacter = 0
				fenceLength = 0
			}
			continue
		}
		if fenceCharacter != 0 {
			continue
		}
		matches := headingPattern.FindStringSubmatch(text)
		if matches == nil {
			continue
		}
		headings = append(headings, Heading{
			Level:        len(matches[1]),
			Name:         strings.TrimSpace(matches[2]),
			HeaderStart:  line.start,
			ContentStart: line.contentEnd,
		})
	}

	return Markdown{source: source, headings: headings}
}

// Headings returns a copy of the heading index.
func (m Markdown) Headings() []Heading {
	return append([]Heading(nil), m.headings...)
}

// Section returns the first section with the requested level and name.
func (m Markdown) Section(level int, name string) (Section, bool) {
	for index, heading := range m.headings {
		if heading.Level == level && heading.Name == name {
			return m.sectionAt(index), true
		}
	}
	return Section{}, false
}

// Sections returns every section with the requested level and name.
func (m Markdown) Sections(level int, name string) []Section {
	sections := make([]Section, 0)
	for index, heading := range m.headings {
		if heading.Level == level && heading.Name == name {
			sections = append(sections, m.sectionAt(index))
		}
	}
	return sections
}

// ReplaceSection replaces one section, preserving the original newline style.
func (m Markdown) ReplaceSection(level int, name, content string) (string, error) {
	sections := m.Sections(level, name)
	if len(sections) != 1 {
		return "", fmt.Errorf("%s must have exactly one heading before syncing views", name)
	}
	found := sections[0]
	newline := "\n"
	if strings.Contains(m.source, "\r\n") {
		newline = "\r\n"
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\n", newline)
	replacement := newline + normalized + newline
	if found.End < len(m.source) {
		replacement += newline
	}
	return m.source[:found.Start] + replacement + m.source[found.End:], nil
}

func (m Markdown) sectionAt(index int) Section {
	heading := m.headings[index]
	end := len(m.source)
	for _, candidate := range m.headings[index+1:] {
		if candidate.Level <= heading.Level {
			end = candidate.HeaderStart
			break
		}
	}
	return Section{
		Heading: heading,
		Content: strings.TrimSpace(m.source[heading.ContentStart:end]),
		Start:   heading.ContentStart,
		End:     end,
	}
}

// Meaningful checks for content after removing comments, tags, and Markdown syntax.
func Meaningful(source string) bool {
	text := htmlCommentPattern.ReplaceAllString(source, "")
	text = htmlTagPattern.ReplaceAllString(text, "")
	text = contentNoise.ReplaceAllString(text, "")
	return strings.TrimSpace(text) != ""
}

// IsExplicitNone recognizes an explicit none value.
func IsExplicitNone(source string) bool {
	return explicitNone.MatchString(strings.TrimSpace(source))
}

func splitLines(source string) []string {
	return lineBreakPattern.Split(source, -1)
}

func sourceLines(source string) []lineRange {
	if source == "" {
		return nil
	}
	lines := make([]lineRange, 0, strings.Count(source, "\n")+1)
	for start := 0; start < len(source); {
		next := strings.IndexByte(source[start:], '\n')
		if next < 0 {
			lines = append(lines, lineRange{start: start, textEnd: len(source), contentEnd: len(source)})
			break
		}
		newline := start + next
		textEnd := newline
		if textEnd > start && source[textEnd-1] == '\r' {
			textEnd--
		}
		lines = append(lines, lineRange{start: start, textEnd: textEnd, contentEnd: newline + 1})
		start = newline + 1
	}
	return lines
}

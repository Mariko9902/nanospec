package validator

import (
	"errors"

	"nanospec/internal/specdoc"
)

func readSpecLanguage(frontmatter specdoc.Frontmatter) (string, error) {
	if !frontmatter.Has("spec_language") {
		return "en-US", nil
	}
	language, scalar := frontmatter.String("spec_language")
	if !hasText(language, scalar) {
		return "", errors.New("cycle.spec_language must be a non-empty language name or tag")
	}
	return language, nil
}

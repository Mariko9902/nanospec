package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type observationContext struct {
	SpecRevision string `json:"spec_revision"`
	CodeRevision string `json:"code_revision"`
	Environment  string `json:"environment"`
	ObservedAt   string `json:"observed_at"`
}

var codeRevisionPattern = regexp.MustCompile(`^(?:[a-fA-F0-9]{40}|sha256:[a-fA-F0-9]{64})$`)

func parseObservationContext(value string) (observationContext, error) {
	var context observationContext
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&context); err != nil {
		return context, fmt.Errorf("context must be a JSON object with spec_revision, code_revision, environment and observed_at")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return context, fmt.Errorf("context must contain exactly one JSON object")
	}
	if !revisionPattern.MatchString(context.SpecRevision) {
		return context, fmt.Errorf("context.spec_revision must match rev-N")
	}
	if !codeRevisionPattern.MatchString(context.CodeRevision) {
		return context, fmt.Errorf("context.code_revision must be a full Git commit or sha256 digest")
	}
	if !singleLineText(context.Environment) {
		return context, fmt.Errorf("context.environment must be one non-empty line")
	}
	if !isISOTimestamp(context.ObservedAt) {
		return context, fmt.Errorf("context.observed_at must be an ISO-8601 timestamp with timezone")
	}
	return context, nil
}

func validateObservationContext(value string) string {
	_, err := parseObservationContext(value)
	if err != nil {
		return err.Error()
	}
	return ""
}

func normalizeObservationContext(value, revision string) (string, error) {
	context, err := parseObservationContext(value)
	if err != nil {
		return "", err
	}
	if context.SpecRevision != revision {
		return "", fmt.Errorf("context.spec_revision must match the prepared spec-revision")
	}
	normalized, err := json.Marshal(context)
	return string(normalized), err
}
